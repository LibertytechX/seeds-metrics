/**
 * Metric Information and Descriptions
 * Used for tooltips and help text
 */

export const metricInfo = {
  fimr: {
    name: 'FIMR',
    fullName: 'First-Installment Miss Rate',
    description: 'Proportion of newly disbursed loans whose first installment was missed.',
    formula: 'FIMR = firstMiss / disbursed',
    whatItMeasures: 'Early indicator of onboarding, KYC, or guarantor issues.',
    bands: {
      green: '≤ 3%',
      watch: '3% - 6%',
      flag: '> 6%',
    },
    interpretation: 'Lower is better. High FIMR indicates problems with loan origination or customer quality.',
    example: 'If 150 out of 5,000 disbursed loans missed first payment, FIMR = 3%',
  },

  slippage: {
    name: 'D0-6 Slippage',
    fullName: 'Early Slippage (Days 0-6)',
    description: 'Portion of amount due in next 7 days that is already 1-6 days past due.',
    formula: 'D0-6 Slippage = dpd1to6Bal / amountDue7d',
    whatItMeasures: 'Early sign of repayment friction or channel loss.',
    bands: {
      green: '≤ 5%',
      watch: '5% - 8%',
      flag: '> 8%',
    },
    interpretation: 'Lower is better. Shows how much of near-term payments are already slipping.',
    example: 'If ₦250K is 1-6 days overdue out of ₦5M due in 7 days, Slippage = 5%',
  },

  roll: {
    name: 'Roll',
    fullName: 'Roll 0-6 → 7-30',
    description: 'Share of early delinquency (1-6 days) that worsened into 7-30 days past due.',
    formula: 'Roll = movedTo7to30 / prevDpd1to6Bal',
    whatItMeasures: 'Containment failure from early lateness to material delinquency.',
    bands: {
      green: '≤ 25%',
      watch: '25% - 35%',
      flag: '> 35%',
    },
    interpretation: 'Lower is better. Shows how well early delinquency is being managed.',
    example: 'If ₦180K of ₦720K early delinquency moved to 7-30 days, Roll = 25%',
  },

  frr: {
    name: 'FRR',
    fullName: 'Fees Realization Rate',
    description: 'Proportion of expected fees that are actually collected.',
    formula: 'FRR = feesCollected / feesDue',
    whatItMeasures: 'Fee collection efficiency and potential system/officer issues.',
    bands: {
      note: 'Used in Risk Score calculation',
    },
    interpretation: 'Higher is better. Shortfalls reduce net yield and indicate collection issues.',
    example: 'If ₦450K collected out of ₦500K due, FRR = 90%',
  },

  ayr: {
    name: 'AYR',
    fullName: 'Adjusted Yield Ratio',
    description: 'Return generated relative to PAR15 exposure at mid-month.',
    formula: 'AYR = (Interest + Fees realized-to-date in month) ÷ (PAR15 exposure at half month)',
    whatItMeasures: 'Economic efficiency of revenue generation relative to portfolio at risk.',
    bands: {
      flag: '< 0.30',
      watch: '0.30 - 0.49',
      green: '≥ 0.50',
    },
    interpretation: 'Higher is better. Shows monthly revenue generation relative to mid-month PAR15 exposure.',
    example: 'If ₦2.5M collected this month and PAR15 at mid-month is ₦5M, AYR = 0.50',
  },

  dqi: {
    name: 'DQI',
    fullName: 'Delinquency Quality Index',
    description: 'Composite index capturing loan quality and repayment discipline (0-100).',
    formula: 'DQI = 100 * (0.50*RQ + 0.35*OTI + 0.15*(1-FIMR))',
    whatItMeasures: 'Overall portfolio quality combining risk, on-time rate, and early defaults.',
    bands: {
      flag: '< 65',
      watch: '65 - 74',
      green: '≥ 75',
    },
    interpretation: 'Higher is better. Composite measure of portfolio health.',
    components: {
      rq: '50% - Risk Quality (normalized risk score)',
      oti: '35% - On-Time Rate (repayment discipline)',
      fimr: '15% - First-Installment Miss Rate (early defaults)',
    },
    example: 'With RQ=0.85, OTI=0.92, FIMR=0.03, DQI = 87',
  },

  riskScore: {
    name: 'Risk Score',
    fullName: 'Composite Officer Risk Score',
    description: 'Single number combining portfolio risk, repayment behavior, and yield performance (0-100).',
    formula: 'RiskScore = 100 - 20*PORR - 15*FIMR - 10*Roll - 40*(1-RepaymentDelayRate/100) - 15*(1-min(AYR,1.0))',
    whatItMeasures: 'Overall officer risk assessment across portfolio quality, repayment behavior, and revenue generation.',
    bands: {
      red: '< 40 (Flag)',
      amber: '40 - 59 (Amber)',
      watch: '60 - 79 (Watch)',
      green: '≥ 80 (Green)',
    },
    interpretation: 'Higher is better. Comprehensive risk indicator focusing on portfolio quality, repayment behavior, and yield.',
    factors: {
      porr: 'Portfolio Open Risk Ratio (20 points max)',
      fimr: 'First-Installment Miss Rate (15 points max)',
      roll: 'Delinquency Roll (10 points max)',
      repaymentDelayRate: 'Repayment Delay Rate (40 points max)',
      ayr: 'Adjusted Yield Ratio (15 points max)',
    },
    example: 'Officer with PORR=0.05, FIMR=0.02, Roll=0.15, RepaymentDelayRate=85%, AYR=0.60 scores 91 (Green)',
  },

  porr: {
    name: 'PORR',
    fullName: 'Portfolio Open Risk Ratio',
    description: 'Fraction of portfolio meeting "open risk" criteria.',
    formula: 'PORR = risky loans / total portfolio',
    whatItMeasures: 'Proportion of portfolio exposed to risk.',
    interpretation: 'Lower is better. Shows concentration of risky loans.',
  },

  channelPurity: {
    name: 'Channel Purity',
    fullName: 'Channel Purity Score',
    description: 'Measure of how clean the customer acquisition channel is (0-1).',
    formula: 'Channel Purity = clean customers / total customers',
    whatItMeasures: 'Quality of customer acquisition channel.',
    interpretation: 'Higher is better. Used in DQI and Risk Score calculations.',
  },

  overdue15d: {
    name: 'Overdue >15D',
    fullName: 'Portfolio Overdue More Than 15 Days',
    description: 'Aggregate monetary value of loans overdue by more than 15 days.',
    formula: 'Sum of all loans with DPD > 15',
    whatItMeasures: 'Material delinquency exposure.',
    interpretation: 'Lower is better. Critical for AYR and PAR reporting.',
  },

  yield: {
    name: 'Yield',
    fullName: 'Total Yield',
    description: 'Total interest and fees collected.',
    formula: 'Yield = interestCollected + feesCollected',
    whatItMeasures: 'Revenue generation.',
    interpretation: 'Higher is better. Shows revenue contribution.',
  },

  officerRank: {
    name: 'Officer Rank',
    fullName: 'Officer Rank by Yield/Risk',
    description: 'Composite ranking using AYR vs Risk Score.',
    formula: 'Rank = weighted(60% Risk Score rank, 40% AYR rank)',
    whatItMeasures: 'Overall officer performance.',
    interpretation: 'Higher rank is better. Identifies top performers.',
  },

  timelinessScore: {
    name: 'Timeliness Score',
    fullName: 'Loan Timeliness Score',
    description: 'Score indicating how timely the borrower makes payments (0-100).',
    formula: 'Provided by main platform ETL',
    whatItMeasures: 'Payment punctuality and consistency.',
    interpretation: 'Higher is better. Score of 100 indicates perfect payment timeliness.',
  },

  repaymentHealth: {
    name: 'Repayment Health',
    fullName: 'Loan Repayment Health Score',
    description: 'Overall health score of loan repayment behavior (0-100).',
    formula: 'Provided by main platform ETL',
    whatItMeasures: 'Overall repayment quality and borrower reliability.',
    interpretation: 'Higher is better. Score of 100 indicates excellent repayment health.',
  },

  daysSinceLastRepayment: {
    name: 'Days Since Last Repayment',
    fullName: 'Days Since Last Repayment',
    description: 'Number of days since the borrower made their last payment.',
    formula: 'CURRENT_DATE - last_payment_date',
    whatItMeasures: 'Recency of payment activity.',
    interpretation: 'Lower is better. High values may indicate payment issues or loan inactivity.',
  },

  avgTimelinessScore: {
    name: 'Avg Timeliness Score',
    fullName: 'Average Timeliness Score',
    description: 'Average timeliness score across all active loans with outstanding balance > ₦2,000.',
    formula: 'AVG(timeliness_score) for loans with total_outstanding > 2000',
    whatItMeasures: 'Officer portfolio payment punctuality.',
    interpretation: 'Higher is better. Shows how timely the officer\'s borrowers are with payments.',
  },

  avgRepaymentHealth: {
    name: 'Avg Repayment Health',
    fullName: 'Average Repayment Health',
    description: 'Average repayment health score across all active loans with outstanding balance > ₦2,000.',
    formula: 'AVG(repayment_health) for loans with total_outstanding > 2000',
    whatItMeasures: 'Officer portfolio repayment quality.',
    interpretation: 'Higher is better. Indicates overall health of officer\'s loan portfolio.',
  },

  avgDaysSinceLastRepayment: {
    name: 'Avg Days Since Last Repayment',
    fullName: 'Average Days Since Last Repayment',
    description: 'Average number of days since last payment across all active loans with outstanding balance > ₦2,000.',
    formula: 'AVG(days_since_last_repayment) for loans with total_outstanding > 2000',
    whatItMeasures: 'Payment recency across officer portfolio.',
    interpretation: 'Lower is better. High values indicate stale payments or collection issues.',
  },

  avgLoanAge: {
    name: 'Avg Loan Age',
    fullName: 'Average Loan Age',
    description: 'Average age in days of all active loans with outstanding balance > ₦2,000.',
    formula: 'AVG(CURRENT_DATE - disbursement_date) for loans with total_outstanding > 2000',
    whatItMeasures: 'Portfolio maturity and loan lifecycle stage.',
    interpretation: 'Context metric. Helps interpret days since last repayment.',
  },

  repaymentDelayRate: {
    name: 'Repayment Delay Rate',
    fullName: 'Repayment Delay Rate',
    description: 'Composite metric measuring payment frequency relative to loan age.',
    formula: 'RepaymentDelayRate = (1 - ((avg_days_since_last_repayment / avg_loan_age) / 0.25)) × 100',
    whatItMeasures: 'Payment frequency and consistency.',
    bands: {
      healthy1: '≥ 88.89% (Healthy 1)',
      healthy2: '77.78% - 88.88% (Healthy 2)',
      healthy3: '66.67% - 77.77% (Healthy 3)',
      watch1: '55.56% - 66.66% (Watch 1)',
      watch2: '44.44% - 55.55% (Watch 2)',
      watch3: '33.33% - 44.43% (Watch 3)',
      risky1: '22.22% - 33.32% (Risky 1)',
      risky2: '11.11% - 22.21% (Risky 2)',
      risky3: '< 11.11% (Risky 3)',
    },
    interpretation: 'Higher is better. Negative values indicate very infrequent payments. 100% means payments every 25% of loan age.',
  },
};

/**
 * Tab Information
 */
export const tabInfo = {
  creditHealth: {
    name: 'Credit Health Overview',
    description: 'Portfolio-level metrics showing overall credit quality and delinquency trends.',
    metrics: ['Overdue >15D', 'AYR', 'DQI', 'FIMR'],
    purpose: 'Monitor portfolio health and identify trends.',
  },

  performance: {
    name: 'Officer Performance',
    description: 'Officer-level rankings and metrics showing productivity and risk.',
    metrics: ['Risk Score', 'AYR', 'Yield', 'Overdue >15D', 'DQI'],
    purpose: 'Compare officers and identify top performers and problem areas.',
  },

  earlyIndicators: {
    name: 'Early Indicators',
    description: 'Early warning metrics that predict future delinquency.',
    metrics: ['FIMR', 'D0-6 Slippage', 'Roll', 'Repayment Delay Rate', 'AYR'],
    purpose: 'Detect early signs of problems before they become material.',
  },

  fimrDrilldown: {
    name: 'FIMR Drilldown',
    description: 'Loan-level details of all loans that missed their first installment payment.',
    metrics: ['Loan ID', 'Officer', 'Customer', 'Disbursement Date', 'Days Since Due', 'Outstanding Balance'],
    purpose: 'Investigate individual FIMR cases for collection outreach and root cause analysis.',
  },

  earlyIndicatorsDrilldown: {
    name: 'Early Indicators Drilldown',
    description: 'Loan-level details of loans in early delinquency (D0-6) or that have rolled to D7-30.',
    metrics: ['Loan ID', 'Current DPD', 'Previous DPD Status', 'Roll Direction', 'Amount Due', 'Outstanding Balance'],
    purpose: 'Monitor early warning signals and identify loans at risk of further delinquency before they become material.',
  },

  agentPerformance: {
    name: 'Agent Performance',
    description: 'Comprehensive officer-level performance metrics showing all key indicators in one view.',
    metrics: ['Risk Score', 'AYR', 'DQI', 'FIMR', 'D0-6 Slippage', 'Roll', 'Repayment Delay Rate', 'Portfolio Total', 'Overdue >15D'],
    purpose: 'Compare officer performance across all metrics and identify top performers and high-risk officers.',
  },
};

/**
 * Get metric info by key
 */
export const getMetricInfo = (metricKey) => {
  return metricInfo[metricKey] || null;
};

/**
 * Get tab info by key
 */
export const getTabInfo = (tabKey) => {
  return tabInfo[tabKey] || null;
};

/**
 * Format metric info for tooltip display
 */
export const formatMetricTooltip = (metricKey) => {
  const info = getMetricInfo(metricKey);
  if (!info) return null;

  const bandsText = info.bands
    ? Object.entries(info.bands)
        .map(([key, value]) => `${key.charAt(0).toUpperCase() + key.slice(1)}: ${value}`)
        .join(' | ')
    : '';

  return `
${info.fullName}

${info.description}

Formula: ${info.formula}

${bandsText ? `Bands: ${bandsText}` : ''}

${info.interpretation}
  `.trim();
};

/**
 * Format tab info for tooltip display
 */
export const formatTabTooltip = (tabKey) => {
  const info = getTabInfo(tabKey);
  if (!info) return null;

  return `
${info.name}

${info.description}

Purpose: ${info.purpose}

Metrics: ${info.metrics.join(', ')}
  `.trim();
};

