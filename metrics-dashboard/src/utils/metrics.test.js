/**
 * Unit tests for metrics calculations
 * Run with: npm test
 */

import {
  calculateFIMR,
  calculateD0to6Slippage,
  calculateRoll,
  calculateFRR,
  calculateAYR,
  calculateDQI,
  calculateRiskScore,
  getBand,
} from './metrics';

describe('Metrics Calculations', () => {
  describe('calculateFIMR', () => {
    test('should calculate FIMR correctly', () => {
      expect(calculateFIMR(150, 5000)).toBeCloseTo(0.03);
      expect(calculateFIMR(0, 5000)).toBe(0);
      expect(calculateFIMR(100, 0)).toBe(0); // Division by zero
    });
  });

  describe('calculateD0to6Slippage', () => {
    test('should calculate D0-6 Slippage correctly', () => {
      expect(calculateD0to6Slippage(250000, 5000000)).toBeCloseTo(0.05);
      expect(calculateD0to6Slippage(0, 5000000)).toBe(0);
      expect(calculateD0to6Slippage(100000, 0)).toBe(0); // Division by zero
    });
  });

  describe('calculateRoll', () => {
    test('should calculate Roll correctly', () => {
      expect(calculateRoll(180000, 720000)).toBeCloseTo(0.25);
      expect(calculateRoll(0, 720000)).toBe(0);
      expect(calculateRoll(100000, 0)).toBe(0); // Division by zero
    });
  });

  describe('calculateFRR', () => {
    test('should calculate FRR correctly', () => {
      expect(calculateFRR(450000, 500000)).toBeCloseTo(0.9);
      expect(calculateFRR(0, 500000)).toBe(0);
      expect(calculateFRR(100000, 0)).toBe(0); // Division by zero
    });
  });

  describe('calculateAYR', () => {
    test('should calculate AYR correctly', () => {
      // AYR = (Interest + Fees) / PAR15 at mid-month
      const ayr = calculateAYR(2100000, 450000, 4500000);
      expect(ayr).toBeCloseTo(0.567, 2); // (2100000 + 450000) / 4500000 = 0.567
    });

    test('should handle zero PAR15', () => {
      expect(calculateAYR(2100000, 450000, 0)).toBe(0);
    });

    test('should calculate AYR with different values', () => {
      const ayr = calculateAYR(1500000, 320000, 6200000);
      expect(ayr).toBeCloseTo(0.294, 2); // (1500000 + 320000) / 6200000 = 0.294
    });
  });

  describe('calculateDQI', () => {
    test('should calculate DQI correctly', () => {
      const dqi = calculateDQI(0.85, 0.92, 0.03, 0.95, true);
      expect(dqi).toBeGreaterThanOrEqual(0);
      expect(dqi).toBeLessThanOrEqual(100);
    });

    test('should apply CP toggle', () => {
      const dqiWithCP = calculateDQI(0.85, 0.92, 0.03, 0.95, true);
      const dqiWithoutCP = calculateDQI(0.85, 0.92, 0.03, 0.95, false);
      expect(dqiWithCP).toBeLessThanOrEqual(dqiWithoutCP);
    });
  });

  describe('calculateRiskScore', () => {
    test('should calculate Risk Score correctly', () => {
      const score = calculateRiskScore({
        porr: 0.02,
        fimr: 0.03,
        roll: 0.25,
        waivers: 50000,
        amountDue7d: 5000000,
        backdated: 5,
        entries: 200,
        reversals: 2,
        frr: 0.9,
        channelPurity: 0.95,
        hadFloatGap: false,
      });
      expect(score).toBeGreaterThanOrEqual(0);
      expect(score).toBeLessThanOrEqual(100);
    });

    test('should penalize float gaps', () => {
      const scoreWithoutGap = calculateRiskScore({
        porr: 0.02,
        fimr: 0.03,
        roll: 0.25,
        waivers: 50000,
        amountDue7d: 5000000,
        backdated: 5,
        entries: 200,
        reversals: 2,
        frr: 0.9,
        channelPurity: 0.95,
        hadFloatGap: false,
      });

      const scoreWithGap = calculateRiskScore({
        porr: 0.02,
        fimr: 0.03,
        roll: 0.25,
        waivers: 50000,
        amountDue7d: 5000000,
        backdated: 5,
        entries: 200,
        reversals: 2,
        frr: 0.9,
        channelPurity: 0.95,
        hadFloatGap: true,
      });

      expect(scoreWithoutGap).toBeGreaterThan(scoreWithGap);
    });
  });

  describe('getBand', () => {
    test('should return correct band for FIMR', () => {
      expect(getBand(0.02, 'fimr')).toEqual({ color: 'green', label: 'Green' });
      expect(getBand(0.04, 'fimr')).toEqual({ color: 'yellow', label: 'Watch' });
      expect(getBand(0.08, 'fimr')).toEqual({ color: 'red', label: 'Flag' });
    });

    test('should return correct band for Risk Score', () => {
      expect(getBand(85, 'riskScore')).toEqual({ color: 'green', label: 'Green' });
      expect(getBand(70, 'riskScore')).toEqual({ color: 'yellow', label: 'Watch' });
      expect(getBand(50, 'riskScore')).toEqual({ color: 'amber', label: 'Amber' });
      expect(getBand(35, 'riskScore')).toEqual({ color: 'red', label: 'Flag' });
    });

    test('should return correct band for AYR', () => {
      expect(getBand(0.55, 'ayr')).toEqual({ color: 'green', label: 'Green' });
      expect(getBand(0.40, 'ayr')).toEqual({ color: 'yellow', label: 'Watch' });
      expect(getBand(0.25, 'ayr')).toEqual({ color: 'red', label: 'Flag' });
    });

    test('should return correct band for DQI', () => {
      expect(getBand(80, 'dqi')).toEqual({ color: 'green', label: 'Green' });
      expect(getBand(70, 'dqi')).toEqual({ color: 'yellow', label: 'Watch' });
      expect(getBand(60, 'dqi')).toEqual({ color: 'red', label: 'Flag' });
    });
  });
});

