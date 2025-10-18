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
    formula: 'DQI = 100 * (0.4*RQ + 0.35*OTI + 0.25*(1-FIMR)) * CP_toggle',
    whatItMeasures: 'Overall portfolio quality combining risk, on-time rate, and early defaults.',
    bands: {
      flag: '< 65',
      watch: '65 - 74',
      green: '≥ 75',
    },
    interpretation: 'Higher is better. Composite measure of portfolio health.',
    components: {
      rq: '40% - Risk Quality (normalized risk score)',
      oti: '35% - On-Time Rate (repayment discipline)',
      fimr: '25% - First-Installment Miss Rate (early defaults)',
      cp: 'Channel Purity multiplier (optional toggle)',
    },
    example: 'With RQ=0.85, OTI=0.92, FIMR=0.03, CP=0.95, DQI = 82',
  },

  riskScore: {
    name: 'Risk Score',
    fullName: 'Composite Officer Risk Score',
    description: 'Single number combining portfolio risk, behavior signals, and integrity (0-100).',
    formula: 'RiskScore = 100 - (penalties for various risk factors)',
    whatItMeasures: 'Overall officer risk assessment across multiple dimensions.',
    bands: {
      red: '< 40 (Flag)',
      amber: '40 - 59 (Amber)',
      watch: '60 - 79 (Watch)',
      green: '≥ 80 (Green)',
    },
    interpretation: 'Higher is better. Comprehensive risk indicator.',
    factors: {
      porr: 'Portfolio Open Risk Ratio (20 points)',
      fimr: 'First-Installment Miss Rate (15 points)',
      roll: 'Delinquency Roll (10 points)',
      waivers: 'Waiver volume (10 points)',
      backdated: 'Backdated entries (10 points)',
      reversals: 'Reversals (10 points)',
      frr: 'Fee Realization Rate (10 points)',
      channelPurity: 'Channel Purity (5 points)',
      floatGap: 'Float/Settlement Gap (10 points)',
    },
    example: 'Officer with good metrics across all factors scores 85 (Green)',
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
    metrics: ['FIMR', 'D0-6 Slippage', 'Roll', 'FRR', 'Channel Purity'],
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
    metrics: ['Risk Score', 'AYR', 'DQI', 'FIMR', 'D0-6 Slippage', 'Roll', 'FRR', 'Portfolio Total', 'Overdue >15D'],
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

