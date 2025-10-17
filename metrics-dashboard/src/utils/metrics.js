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
 * AYR - Adjusted Yield Ratio (normalized form)
 * Formula: AYR_normalized = (interestCollected + feesCollected) / (1 + overdue15dRatio)
 * where overdue15dRatio = overdue15d / totalPortfolio
 */
export const calculateAYR = (interestCollected, feesCollected, overdue15d, totalPortfolio) => {
  const numerator = interestCollected + feesCollected;
  if (totalPortfolio === 0) return 0;
  const overdue15dRatio = overdue15d / totalPortfolio;
  return numerator / (1 + overdue15dRatio);
};

/**
 * DQI - Delinquency Quality Index (0-100)
 * Formula: DQI = round(100 * (0.4 * RQ + 0.35 * OTI + 0.25 * (1 - FIMR)) * CP_toggle)
 */
export const calculateDQI = (riskScoreNorm, onTimeRate, fimr, channelPurity, cpToggle = true) => {
  const rq = clamp01(riskScoreNorm);
  const oti = clamp01(onTimeRate);
  const fimrClamped = clamp01(fimr);
  const cp = cpToggle ? clamp01(channelPurity) : 1;
  
  const dqi = 100 * (0.4 * rq + 0.35 * oti + 0.25 * (1 - fimrClamped)) * cp;
  return Math.round(dqi);
};

/**
 * Composite Officer Risk Score (0-100)
 * Formula: RiskScore = 100 - 20*PORR - 15*FIMR - 10*Roll - 10*(waivers/amountDue7d)
 *                      - 10*(backdated/entries) - 10*(reversals/entries) - 10*(1-FRR)
 *                      - 5*(1-channelPurity) - 10*(hadFloatGap ? 1 : 0)
 */
export const calculateRiskScore = (params) => {
  const {
    porr = 0,
    fimr = 0,
    roll = 0,
    waivers = 0,
    amountDue7d = 0,
    backdated = 0,
    entries = 0,
    reversals = 0,
    frr = 0,
    channelPurity = 1,
    hadFloatGap = false,
  } = params;

  let score = 100;
  score -= 20 * clamp01(porr);
  score -= 15 * clamp01(fimr);
  score -= 10 * clamp01(roll);
  score -= 10 * safeDivide(waivers, amountDue7d, 0);
  score -= 10 * safeDivide(backdated, entries, 0);
  score -= 10 * safeDivide(reversals, entries, 0);
  score -= 10 * clamp01(1 - frr);
  score -= 5 * clamp01(1 - channelPurity);
  score -= 10 * (hadFloatGap ? 1 : 0);

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

