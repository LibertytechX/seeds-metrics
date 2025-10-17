/**
 * Mock data for the dashboard
 */

import {
  calculateFIMR,
  calculateD0to6Slippage,
  calculateRoll,
  calculateFRR,
  calculateAYR,
  calculateDQI,
  calculateRiskScore,
} from './metrics';

const officers = [
  {
    id: 1,
    name: 'John Doe',
    region: 'Lagos',
    branch: 'Lagos Main',
    channel: 'Direct',
    // Raw metrics
    firstMiss: 150,
    disbursed: 5000,
    dpd1to6Bal: 250000,
    amountDue7d: 5000000,
    movedTo7to30: 180000,
    prevDpd1to6Bal: 720000,
    feesCollected: 450000,
    feesDue: 500000,
    interestCollected: 2100000,
    overdue15d: 1200000,
    totalPortfolio: 50000000,
    riskScoreNorm: 0.85,
    onTimeRate: 0.92,
    channelPurity: 0.95,
    porr: 0.02,
    waivers: 50000,
    backdated: 5,
    entries: 200,
    reversals: 2,
    hadFloatGap: false,
  },
  {
    id: 2,
    name: 'Grace Okon',
    region: 'Abuja',
    branch: 'Abuja Central',
    channel: 'Partner',
    firstMiss: 280,
    disbursed: 4200,
    dpd1to6Bal: 450000,
    amountDue7d: 4500000,
    movedTo7to30: 320000,
    prevDpd1to6Bal: 800000,
    feesCollected: 320000,
    feesDue: 500000,
    interestCollected: 1500000,
    overdue15d: 2800000,
    totalPortfolio: 45000000,
    riskScoreNorm: 0.65,
    onTimeRate: 0.78,
    channelPurity: 0.80,
    porr: 0.06,
    waivers: 120000,
    backdated: 12,
    entries: 180,
    reversals: 5,
    hadFloatGap: true,
  },
  {
    id: 3,
    name: 'Musa Adebayo',
    region: 'Kano',
    branch: 'Kano North',
    channel: 'Direct',
    firstMiss: 420,
    disbursed: 3800,
    dpd1to6Bal: 680000,
    amountDue7d: 3800000,
    movedTo7to30: 520000,
    prevDpd1to6Bal: 1200000,
    feesCollected: 180000,
    feesDue: 500000,
    interestCollected: 900000,
    overdue15d: 4200000,
    totalPortfolio: 35000000,
    riskScoreNorm: 0.45,
    onTimeRate: 0.62,
    channelPurity: 0.65,
    porr: 0.12,
    waivers: 250000,
    backdated: 25,
    entries: 150,
    reversals: 10,
    hadFloatGap: true,
  },
];

// Calculate derived metrics for each officer
export const generateOfficerMetrics = (officer) => {
  const fimr = calculateFIMR(officer.firstMiss, officer.disbursed);
  const slippage = calculateD0to6Slippage(officer.dpd1to6Bal, officer.amountDue7d);
  const roll = calculateRoll(officer.movedTo7to30, officer.prevDpd1to6Bal);
  const frr = calculateFRR(officer.feesCollected, officer.feesDue);
  const ayr = calculateAYR(
    officer.interestCollected,
    officer.feesCollected,
    officer.overdue15d,
    officer.totalPortfolio
  );
  const dqi = calculateDQI(
    officer.riskScoreNorm,
    officer.onTimeRate,
    fimr,
    officer.channelPurity,
    true
  );
  const riskScore = calculateRiskScore({
    porr: officer.porr,
    fimr,
    roll,
    waivers: officer.waivers,
    amountDue7d: officer.amountDue7d,
    backdated: officer.backdated,
    entries: officer.entries,
    reversals: officer.reversals,
    frr,
    channelPurity: officer.channelPurity,
    hadFloatGap: officer.hadFloatGap,
  });

  return {
    ...officer,
    fimr,
    slippage,
    roll,
    frr,
    ayr,
    dqi,
    riskScore,
    yield: officer.interestCollected + officer.feesCollected,
    overdue15dVolume: officer.overdue15d,
  };
};

export const mockOfficers = officers.map(generateOfficerMetrics);

export const mockPortfolioMetrics = {
  totalOverdue15d: mockOfficers.reduce((sum, o) => sum + o.overdue15d, 0),
  avgDQI: Math.round(mockOfficers.reduce((sum, o) => sum + o.dqi, 0) / mockOfficers.length),
  avgAYR: (mockOfficers.reduce((sum, o) => sum + o.ayr, 0) / mockOfficers.length).toFixed(2),
  avgRiskScore: Math.round(mockOfficers.reduce((sum, o) => sum + o.riskScore, 0) / mockOfficers.length),
  topOfficer: mockOfficers.reduce((best, o) => (o.ayr > best.ayr ? o : best)),
  watchlistCount: mockOfficers.filter(o => o.riskScore < 80).length,
};

export const mockLoans = [
  { id: 101, officerId: 1, amount: 500000, daysOverdue: 0, status: 'Current', channel: 'Direct' },
  { id: 102, officerId: 1, amount: 300000, daysOverdue: 5, status: 'Early', channel: 'Direct' },
  { id: 103, officerId: 2, amount: 450000, daysOverdue: 20, status: 'Overdue', channel: 'Partner' },
  { id: 104, officerId: 3, amount: 600000, daysOverdue: 35, status: 'Overdue', channel: 'Direct' },
];

