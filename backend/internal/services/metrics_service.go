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
	// Risk Score = 1 - (weighted average of negative indicators)
	// Negative indicators: PORR, FIMR, Roll, Waivers, Backdated, Reversals, Float Gap

	score := 1.0

	// PORR penalty (weight: 0.25)
	score -= calc.PORR * 0.25

	// FIMR penalty (weight: 0.20)
	score -= calc.FIMR * 0.20

	// Roll penalty (weight: 0.15)
	score -= calc.Roll * 0.15

	// Waivers penalty (weight: 0.10)
	if raw.TotalPortfolio > 0 {
		waiversRatio := raw.Waivers / raw.TotalPortfolio
		score -= waiversRatio * 0.10
	}

	// Backdated entries penalty (weight: 0.10)
	if raw.Entries > 0 {
		backdatedRatio := float64(raw.Backdated) / float64(raw.Entries)
		score -= backdatedRatio * 0.10
	}

	// Reversals penalty (weight: 0.10)
	if raw.Entries > 0 {
		reversalsRatio := float64(raw.Reversals) / float64(raw.Entries)
		score -= reversalsRatio * 0.10
	}

	// Float gap penalty (weight: 0.10)
	if raw.HadFloatGap {
		score -= 0.10
	}

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
	// Positive indicators: Risk Score, On-time Rate, Channel Purity
	// Negative indicators: FIMR

	dqi := 0.0

	// Risk Score contribution (weight: 0.40)
	dqi += calc.RiskScoreNorm * 0.40

	// On-time rate contribution (weight: 0.30)
	dqi += calc.OnTimeRate * 0.30

	// Channel purity contribution (weight: 0.20)
	dqi += calc.ChannelPurity * 0.20

	// FIMR penalty (weight: 0.10)
	dqi += (1.0 - calc.FIMR) * 0.10

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
	var topAYR float64
	var topOfficer *models.TopOfficer

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

			// Count watchlist (risk score < 60)
			if officer.CalculatedMetrics.RiskScore < 60 {
				watchlistCount++
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
	portfolio.TotalLoans = totalLoans
	portfolio.TotalPortfolio = totalPortfolio

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
