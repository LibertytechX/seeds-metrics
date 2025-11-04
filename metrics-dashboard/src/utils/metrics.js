/**
 * Metrics calculation utilities
 * Based on the Build Guide formulas
 */

// Helper to clamp values between 0 and 1
const clamp01 = (x) => Math.max(0, Math.min(1, x));

// Helper to safely divide (avoid division by zero)
const safeDivide = (numerator, denominator, defaultValue = 0) => {
  return denominator === 0 ? defaultValue : numerator / denominator;
};

/**
 * FIMR - First-Installment Miss Rate
 * Formula: FIMR = firstMiss / disbursed
 */
export const calculateFIMR = (firstMiss, disbursed) => {
  return safeDivide(firstMiss, disbursed, 0);
};

/**
 * D0-6 Slippage - Early Slippage
 * Formula: D0-6 Slippage = dpd1to6Bal / amountDue7d
 */
export const calculateD0to6Slippage = (dpd1to6Bal, amountDue7d) => {
  return safeDivide(dpd1to6Bal, amountDue7d, 0);
};

/**
 * Roll 0-6 -> 7-30
 * Formula: Roll = movedTo7to30 / prevDpd1to6Bal
 */
export const calculateRoll = (movedTo7to30, prevDpd1to6Bal) => {
  return safeDivide(movedTo7to30, prevDpd1to6Bal, 0);
};

/**
 * FRR - Fees Realization Rate
 * Formula: FRR = feesCollected / feesDue
 */
export const calculateFRR = (feesCollected, feesDue) => {
  return safeDivide(feesCollected, feesDue, 0);
};

/**
 * AYR - Adjusted Yield Ratio
 * Formula: AYR = (Interest + Fees realized-to-date on officer's book in month) ÷ (PAR15 exposure at half month)
 * where:
 * - Numerator: Interest + Fees collected in the current month for the officer's portfolio
 * - Denominator: PAR15 (Portfolio at Risk >15 days) exposure measured at mid-month
 */
export const calculateAYR = (interestCollected, feesCollected, par15MidMonth) => {
  const numerator = interestCollected + feesCollected;
  return safeDivide(numerator, par15MidMonth, 0);
};

/**
 * DQI - Delinquency Quality Index (0-100)
 * Formula: DQI = round(100 * (0.50 * RQ + 0.35 * OTI + 0.15 * (1 - FIMR)))
 * Note: Channel Purity removed as it's no longer part of Risk Score
 */
export const calculateDQI = (riskScoreNorm, onTimeRate, fimr) => {
  const rq = clamp01(riskScoreNorm);
  const oti = clamp01(onTimeRate);
  const fimrClamped = clamp01(fimr);

  const dqi = 100 * (0.50 * rq + 0.35 * oti + 0.15 * (1 - fimrClamped));
  return Math.round(dqi);
};

/**
 * Composite Officer Risk Score (0-100)
 * NEW Formula: RiskScore = 100 - 20*PORR - 15*FIMR - 10*Roll
 *                          - 40*(1 - repaymentDelayRate/100) - 15*(1 - min(AYR, 1.0))
 *
 * Penalties:
 * - PORR: 20 points max
 * - FIMR: 15 points max
 * - Roll: 10 points max
 * - Repayment Delay Rate: 40 points max
 * - AYR: 15 points max
 * Total: 100 points max
 */
export const calculateRiskScore = (params) => {
  const {
    porr = 0,
    fimr = 0,
    roll = 0,
    repaymentDelayRate = 0,
    ayr = 0,
  } = params;

  let score = 100;

  // PORR penalty (max: 20 points)
  score -= 20 * clamp01(porr);

  // FIMR penalty (max: 15 points)
  score -= 15 * clamp01(fimr);

  // Roll penalty (max: 10 points)
  score -= 10 * clamp01(roll);

  // Repayment Delay Rate penalty (max: 40 points)
  // If repaymentDelayRate = 100%, penalty = 0
  // If repaymentDelayRate = 0%, penalty = 40
  // If repaymentDelayRate is negative, penalty = 40 (capped)
  if (repaymentDelayRate <= 100) {
    const delayRatePenalty = (1 - (repaymentDelayRate / 100)) * 40;
    score -= Math.min(40, Math.max(0, delayRatePenalty));
  }
  // If repaymentDelayRate > 100%, no penalty (better than expected)

  // AYR penalty (max: 15 points)
  // If AYR >= 1.0, penalty = 0
  // If AYR = 0.5, penalty = 7.5
  // If AYR = 0, penalty = 15
  const ayrCapped = Math.min(ayr, 1.0);
  score -= (1 - ayrCapped) * 15;

  return Math.max(0, Math.round(score));
};

/**
 * Get band color and label based on metric value and thresholds
 */
export const getBand = (value, metric) => {
  const thresholds = {
    fimr: { green: 0.03, watch: 0.06 },
    slippage: { green: 0.05, watch: 0.08 },
    roll: { green: 0.25, watch: 0.35 },
    dqi: { flag: 65, watch: 75 },
    ayr: { flag: 0.30, watch: 0.49 },
    riskScore: { red: 40, amber: 59, watch: 80 },
  };

  const t = thresholds[metric];
  if (!t) return { color: 'gray', label: 'Unknown' };

  if (metric === 'riskScore') {
    if (value < t.red) return { color: 'red', label: 'Flag' };
    if (value < t.amber) return { color: 'amber', label: 'Amber' };
    if (value < t.watch) return { color: 'yellow', label: 'Watch' };
    return { color: 'green', label: 'Green' };
  }

  if (metric === 'dqi') {
    if (value < t.flag) return { color: 'red', label: 'Flag' };
    if (value < t.watch) return { color: 'yellow', label: 'Watch' };
    return { color: 'green', label: 'Green' };
  }

  if (metric === 'ayr') {
    if (value < t.flag) return { color: 'red', label: 'Flag' };
    if (value < t.watch) return { color: 'yellow', label: 'Watch' };
    return { color: 'green', label: 'Green' };
  }

  // For percentage-based metrics (fimr, slippage, roll)
  if (value <= t.green) return { color: 'green', label: 'Green' };
  if (value <= t.watch) return { color: 'yellow', label: 'Watch' };
  return { color: 'red', label: 'Flag' };
};

