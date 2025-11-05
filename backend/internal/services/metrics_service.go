package services

import (
	"math"

	"github.com/seeds-metrics/analytics-backend/internal/models"
)

// MetricsService handles metric calculations
type MetricsService struct{}

// NewMetricsService creates a new metrics service
func NewMetricsService() *MetricsService {
	return &MetricsService{}
}

// CalculateOfficerMetrics calculates all metrics for an officer
func (s *MetricsService) CalculateOfficerMetrics(raw *models.RawMetrics) *models.CalculatedMetrics {
	calculated := &models.CalculatedMetrics{}

	// FIMR = firstMiss / disbursed
	if raw.Disbursed > 0 {
		calculated.FIMR = float64(raw.FirstMiss) / float64(raw.Disbursed)
	}

	// D0-6 Slippage = dpd1to6Bal / amountDue7d
	if raw.AmountDue7d > 0 {
		calculated.Slippage = raw.Dpd1to6Bal / raw.AmountDue7d
	}

	// Roll = movedTo7to30 / prevDpd1to6Bal
	if raw.PrevDpd1to6Bal > 0 {
		calculated.Roll = raw.MovedTo7to30 / raw.PrevDpd1to6Bal
	}

	// FRR = feesCollected / feesDue
	if raw.FeesDue > 0 {
		calculated.FRR = raw.FeesCollected / raw.FeesDue
	}

	// AYR = (interestCollected + feesCollected) / par15MidMonth
	if raw.Par15MidMonth > 0 {
		calculated.AYR = (raw.InterestCollected + raw.FeesCollected) / raw.Par15MidMonth
	}

	// Yield = interestCollected + feesCollected
	calculated.Yield = raw.InterestCollected + raw.FeesCollected

	// Overdue15dVolume
	calculated.Overdue15dVolume = raw.Overdue15d

	// PORR = overdue15d / totalPortfolio
	if raw.TotalPortfolio > 0 {
		calculated.PORR = raw.Overdue15d / raw.TotalPortfolio
	}

	// Channel Purity (simplified - assume 1.0 for now, should be calculated from actual data)
	calculated.ChannelPurity = 1.0

	// On-time rate (simplified - inverse of slippage)
	calculated.OnTimeRate = 1.0 - calculated.Slippage
	if calculated.OnTimeRate < 0 {
		calculated.OnTimeRate = 0
	}

	// Risk Score Normalized (0-1 scale)
	calculated.RiskScoreNorm = s.CalculateRiskScoreNorm(raw, calculated)

	// Risk Score (0-100 scale)
	calculated.RiskScore = int(calculated.RiskScoreNorm * 100)

	// DQI (Data Quality Index)
	calculated.DQI = s.CalculateDQI(raw, calculated)

	// NEW: Repayment behavior metrics
	calculated.AvgTimelinessScore = raw.AvgTimelinessScore
	calculated.AvgRepaymentHealth = raw.AvgRepaymentHealth
	calculated.AvgDaysSinceLastRepayment = raw.AvgDaysSinceLastRepayment
	calculated.AvgLoanAge = raw.AvgLoanAge

	// NEW: Repayment Delay Rate
	// Formula: (1 - ((avg_days_since_last_repayment / avg_loan_age) / 0.25)) × 100
	// Edge cases:
	// - If avg_loan_age = 0, return 0 (NULL would be better but we'll use 0 for simplicity)
	// - Allow negative values (as per user requirement)
	if raw.AvgLoanAge > 0 {
		ratio := raw.AvgDaysSinceLastRepayment / raw.AvgLoanAge
		normalizedRatio := ratio / 0.25
		calculated.RepaymentDelayRate = (1.0 - normalizedRatio) * 100
	} else {
		calculated.RepaymentDelayRate = 0
	}

	return calculated
}

// CalculateRiskScoreNorm calculates normalized risk score (0-1)
func (s *MetricsService) CalculateRiskScoreNorm(raw *models.RawMetrics, calc *models.CalculatedMetrics) float64 {
	// NEW Risk Score Formula = 1 - (weighted penalties)
	// Penalties:
	// - PORR: 20 points max (0.20 weight)
	// - FIMR: 15 points max (0.15 weight)
	// - Roll: 10 points max (0.10 weight)
	// - Repayment Delay Rate: 40 points max (0.40 weight)
	// - AYR: 15 points max (0.15 weight)
	// Total: 100 points max

	score := 1.0

	// PORR penalty (max: 20 points = 0.20 weight)
	score -= calc.PORR * 0.20

	// FIMR penalty (max: 15 points = 0.15 weight)
	score -= calc.FIMR * 0.15

	// Roll penalty (max: 10 points = 0.10 weight)
	score -= calc.Roll * 0.10

	// Repayment Delay Rate penalty (max: 40 points = 0.40 weight)
	// Formula: penalty = (1 - (repayment_delay_rate / 100)) * 0.40
	// If repayment_delay_rate = 100%, penalty = 0
	// If repayment_delay_rate = 0%, penalty = 0.40
	// If repayment_delay_rate is negative, penalty > 0.40 (capped at 0.40)
	if calc.RepaymentDelayRate <= 100 {
		delayRatePenalty := (1.0 - (calc.RepaymentDelayRate / 100.0)) * 0.40
		// Cap penalty at 0.40 (for negative delay rates)
		if delayRatePenalty > 0.40 {
			delayRatePenalty = 0.40
		}
		score -= delayRatePenalty
	}
	// If repayment_delay_rate > 100%, no penalty (better than expected)

	// AYR penalty (max: 15 points = 0.15 weight)
	// Formula: penalty = (1 - min(AYR, 1.0)) * 0.15
	// If AYR >= 1.0, penalty = 0
	// If AYR = 0.5, penalty = 0.075 (7.5 points)
	// If AYR = 0, penalty = 0.15 (15 points)
	ayrCapped := calc.AYR
	if ayrCapped > 1.0 {
		ayrCapped = 1.0
	}
	score -= (1.0 - ayrCapped) * 0.15

	// Ensure score is between 0 and 1
	if score < 0 {
		score = 0
	}
	if score > 1 {
		score = 1
	}

	return score
}

// CalculateDQI calculates Data Quality Index (0-100)
func (s *MetricsService) CalculateDQI(raw *models.RawMetrics, calc *models.CalculatedMetrics) int {
	// DQI = weighted average of positive indicators
	// Positive indicators: Risk Score, On-time Rate
	// Negative indicators: FIMR
	// Note: Channel Purity removed as it's no longer part of Risk Score

	dqi := 0.0

	// Risk Score contribution (weight: 0.50 - increased from 0.40)
	dqi += calc.RiskScoreNorm * 0.50

	// On-time rate contribution (weight: 0.35 - increased from 0.30)
	dqi += calc.OnTimeRate * 0.35

	// FIMR penalty (weight: 0.15 - increased from 0.10)
	dqi += (1.0 - calc.FIMR) * 0.15

	// Convert to 0-100 scale
	dqiScore := int(dqi * 100)

	// Ensure DQI is between 0 and 100
	if dqiScore < 0 {
		dqiScore = 0
	}
	if dqiScore > 100 {
		dqiScore = 100
	}

	return dqiScore
}

// CalculatePortfolioMetrics calculates portfolio-level metrics from officer metrics
func (s *MetricsService) CalculatePortfolioMetrics(officers []*models.DashboardOfficerMetrics) *models.PortfolioMetrics {
	if len(officers) == 0 {
		return &models.PortfolioMetrics{}
	}

	portfolio := &models.PortfolioMetrics{
		TotalOfficers: len(officers),
	}

	var totalOverdue15d float64
	var totalDQI int
	var totalAYR float64
	var totalRiskScore int
	var totalLoans int
	var totalPortfolio float64
	var watchlistCount int
	var watchlistPortfolio float64
	var topAYR float64
	var topOfficer *models.TopOfficer

	// New metrics aggregation variables
	var totalRepaymentDelayRate float64
	var officersWithDelayRate int
	var atRiskOfficersCount int

	for _, officer := range officers {
		if officer.CalculatedMetrics != nil {
			totalOverdue15d += officer.CalculatedMetrics.Overdue15dVolume
			totalDQI += officer.CalculatedMetrics.DQI
			totalAYR += officer.CalculatedMetrics.AYR
			totalRiskScore += officer.CalculatedMetrics.RiskScore

			// Track top officer by AYR
			if officer.CalculatedMetrics.AYR > topAYR {
				topAYR = officer.CalculatedMetrics.AYR
				topOfficer = &models.TopOfficer{
					OfficerID: officer.OfficerID,
					Name:      officer.Name,
					AYR:       officer.CalculatedMetrics.AYR,
				}
			}

			// Count watchlist (risk score < 40: Red band only - target <50% of officers)
			if officer.CalculatedMetrics.RiskScore < 40 {
				watchlistCount++
				// Add this officer's portfolio to watchlist portfolio
				if officer.RawMetrics != nil {
					watchlistPortfolio += officer.RawMetrics.TotalPortfolio
				}
			}

			// Aggregate repayment delay rate
			if officer.CalculatedMetrics.RepaymentDelayRate != 0 {
				totalRepaymentDelayRate += officer.CalculatedMetrics.RepaymentDelayRate
				officersWithDelayRate++
			}

			// Check if officer is at risk (avg DPD > 10 AND avg loan age > 14)
			// Note: We use AvgDaysSinceLastRepayment as a proxy for DPD and AvgLoanAge from calculated metrics
			avgLoanAge := officer.CalculatedMetrics.AvgLoanAge
			avgDaysSinceLastRepayment := officer.CalculatedMetrics.AvgDaysSinceLastRepayment
			if avgDaysSinceLastRepayment > 10 && avgLoanAge > 14 {
				atRiskOfficersCount++
			}
		}

		if officer.RawMetrics != nil {
			totalLoans += officer.RawMetrics.Disbursed
			totalPortfolio += officer.RawMetrics.TotalPortfolio
		}
	}

	portfolio.TotalOverdue15d = totalOverdue15d
	portfolio.AvgDQI = totalDQI / len(officers)
	portfolio.AvgAYR = totalAYR / float64(len(officers))
	portfolio.AvgRiskScore = totalRiskScore / len(officers)
	portfolio.TopOfficer = topOfficer
	portfolio.WatchlistCount = watchlistCount
	portfolio.WatchlistPortfolio = watchlistPortfolio
	portfolio.TotalLoans = totalLoans
	portfolio.TotalPortfolio = totalPortfolio

	// Calculate average repayment delay rate
	if officersWithDelayRate > 0 {
		portfolio.AvgRepaymentDelayRate = totalRepaymentDelayRate / float64(officersWithDelayRate)
	}

	// Calculate at-risk officers percentage
	if len(officers) > 0 {
		portfolio.AtRiskOfficersCount = atRiskOfficersCount
		portfolio.AtRiskOfficersPercentage = (float64(atRiskOfficersCount) / float64(len(officers))) * 100
	}

	return portfolio
}

// CalculateRollDirection determines if a loan is worsening, stable, or improving
func (s *MetricsService) CalculateRollDirection(currentDPD, previousDPD int) string {
	if currentDPD > previousDPD {
		return "Worsening"
	} else if currentDPD < previousDPD {
		return "Improving"
	}
	return "Stable"
}

// Round rounds a float to n decimal places
func Round(val float64, places int) float64 {
	multiplier := math.Pow(10, float64(places))
	return math.Round(val*multiplier) / multiplier
}
