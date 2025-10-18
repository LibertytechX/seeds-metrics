# ✅ AYR Formula Update - COMPLETE

## Change Summary

Updated the AYR (Adjusted Yield Ratio) formula from the old normalized form to the new business-specified formula.

---

## Old Formula (REMOVED)

```
AYR = (interestCollected + feesCollected) / (1 + overdue15dRatio)
where overdue15dRatio = overdue15d / totalPortfolio
```

**Parameters:**
- `interestCollected` - Interest collected
- `feesCollected` - Fees collected
- `overdue15d` - Overdue amount >15 days
- `totalPortfolio` - Total portfolio value

---

## New Formula (IMPLEMENTED)

```
AYR = (Interest + Fees realized-to-date on officer's book in month) ÷ (PAR15 exposure at half month)
```

**Parameters:**
- `interestCollected` - Interest collected in current month for officer's portfolio
- `feesCollected` - Fees collected in current month for officer's portfolio
- `par15MidMonth` - PAR15 (Portfolio at Risk >15 days) exposure measured at mid-month

**Band Thresholds (UNCHANGED):**
- Green: AYR ≥ 0.50
- Watch: 0.30 – 0.49
- Flag: < 0.30

---

## Files Modified

### 1. `src/utils/metrics.js`
**Function:** `calculateAYR`

**Before:**
```javascript
export const calculateAYR = (interestCollected, feesCollected, overdue15d, totalPortfolio) => {
  const numerator = interestCollected + feesCollected;
  if (totalPortfolio === 0) return 0;
  const overdue15dRatio = overdue15d / totalPortfolio;
  return numerator / (1 + overdue15dRatio);
};
```

**After:**
```javascript
export const calculateAYR = (interestCollected, feesCollected, par15MidMonth) => {
  const numerator = interestCollected + feesCollected;
  return safeDivide(numerator, par15MidMonth, 0);
};
```

**Changes:**
- Removed `overdue15d` and `totalPortfolio` parameters
- Added `par15MidMonth` parameter
- Simplified calculation to direct division
- Uses `safeDivide` helper for zero-denominator handling

---

### 2. `src/utils/metricInfo.js`
**Section:** `ayr` metric information

**Before:**
```javascript
ayr: {
  name: 'AYR',
  fullName: 'Adjusted Yield Ratio',
  description: 'Return generated relative to material overdue exposure (>15 days).',
  formula: 'AYR = (interestCollected + feesCollected) / (1 + overdue15dRatio)',
  whatItMeasures: 'Economic efficiency while accounting for problem loans.',
  interpretation: 'Higher is better. Shows return generation relative to overdue exposure.',
  example: 'If ₦2.55M collected and 2.4% of portfolio is overdue >15 days, AYR ≈ 0.58',
}
```

**After:**
```javascript
ayr: {
  name: 'AYR',
  fullName: 'Adjusted Yield Ratio',
  description: 'Return generated relative to PAR15 exposure at mid-month.',
  formula: 'AYR = (Interest + Fees realized-to-date in month) ÷ (PAR15 exposure at half month)',
  whatItMeasures: 'Economic efficiency of revenue generation relative to portfolio at risk.',
  interpretation: 'Higher is better. Shows monthly revenue generation relative to mid-month PAR15 exposure.',
  example: 'If ₦2.5M collected this month and PAR15 at mid-month is ₦5M, AYR = 0.50',
}
```

**Changes:**
- Updated description to reference PAR15 at mid-month
- Updated formula to match new business specification
- Updated interpretation to clarify monthly revenue vs mid-month PAR15
- Updated example with clearer numbers

---

### 3. `src/utils/mockData.js`
**Changes:**

**Added `par15MidMonth` field to all officers:**

**Officer 1 (John Doe):**
```javascript
par15MidMonth: 4500000, // PAR15 exposure at mid-month
```

**Officer 2 (Grace Okon):**
```javascript
par15MidMonth: 6200000, // PAR15 exposure at mid-month
```

**Officer 3 (Musa Adebayo):**
```javascript
par15MidMonth: 3600000, // PAR15 exposure at mid-month
```

**Updated function call:**
```javascript
// Before
const ayr = calculateAYR(
  officer.interestCollected,
  officer.feesCollected,
  officer.overdue15d,
  officer.totalPortfolio
);

// After
const ayr = calculateAYR(
  officer.interestCollected,
  officer.feesCollected,
  officer.par15MidMonth
);
```

---

### 4. `src/utils/metrics.test.js`
**Updated test cases:**

**Before:**
```javascript
describe('calculateAYR', () => {
  test('should calculate AYR correctly', () => {
    const ayr = calculateAYR(2100000, 450000, 1200000, 50000000);
    expect(ayr).toBeGreaterThan(0);
    expect(ayr).toBeLessThan(1);
  });

  test('should handle zero portfolio', () => {
    expect(calculateAYR(2100000, 450000, 1200000, 0)).toBe(0);
  });
});
```

**After:**
```javascript
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
```

**Changes:**
- Updated test to use new 3-parameter signature
- Added specific expected values with `toBeCloseTo`
- Renamed test from "zero portfolio" to "zero PAR15"
- Added third test case with different values

---

## Example Calculations

### Officer 1 (John Doe)
```
Interest Collected: ₦2,100,000
Fees Collected: ₦450,000
PAR15 Mid-Month: ₦4,500,000

AYR = (2,100,000 + 450,000) / 4,500,000
AYR = 2,550,000 / 4,500,000
AYR = 0.567 (Green - above 0.50 threshold)
```

### Officer 2 (Grace Okon)
```
Interest Collected: ₦1,500,000
Fees Collected: ₦320,000
PAR15 Mid-Month: ₦6,200,000

AYR = (1,500,000 + 320,000) / 6,200,000
AYR = 1,820,000 / 6,200,000
AYR = 0.294 (Flag - below 0.30 threshold)
```

### Officer 3 (Musa Adebayo)
```
Interest Collected: ₦900,000
Fees Collected: ₦180,000
PAR15 Mid-Month: ₦3,600,000

AYR = (900,000 + 180,000) / 3,600,000
AYR = 1,080,000 / 3,600,000
AYR = 0.300 (Watch - exactly at 0.30 threshold)
```

---

## Impact on Dashboard

### Tooltip Changes
When users hover over "AYR" in the dashboard, they will now see:

```
Adjusted Yield Ratio

Return generated relative to PAR15 exposure at mid-month.

Formula: AYR = (Interest + Fees realized-to-date in month) ÷ (PAR15 exposure at half month)

Bands: Flag: < 0.30 | Watch: 0.30 - 0.49 | Green: ≥ 0.50

Higher is better. Shows monthly revenue generation relative to mid-month PAR15 exposure.
```

### Visual Changes
- AYR values will be recalculated based on new formula
- Color bands remain the same (Green/Watch/Flag)
- Officers may move between bands due to formula change

---

## Business Logic Changes

### What Changed
1. **Denominator:** Changed from `(1 + overdue15dRatio)` to `par15MidMonth`
2. **Measurement Point:** Now uses mid-month snapshot instead of current overdue
3. **Interpretation:** More directly measures revenue efficiency against at-risk portfolio

### Why This Matters
- **Old formula:** Normalized yield by total portfolio risk ratio
- **New formula:** Direct comparison of monthly revenue to mid-month PAR15
- **Better alignment:** Matches business requirement to measure revenue against portfolio at risk

---

## Data Requirements

### For Backend Integration
When connecting to real data, ensure each officer record includes:

```javascript
{
  interestCollected: number,    // Interest collected in current month
  feesCollected: number,         // Fees collected in current month
  par15MidMonth: number,         // PAR15 exposure at mid-month (15th of month)
}
```

### PAR15 Definition
**PAR15** = Portfolio at Risk >15 days
- Sum of all loan balances where days past due > 15
- Measured at mid-month (typically 15th of the month)
- Used as denominator in AYR calculation

---

## Testing

### Manual Testing
1. Open http://localhost:5173
2. Navigate to "Officer Performance" tab
3. Check AYR column values
4. Hover over "AYR" header to see updated tooltip
5. Verify color bands are correct

### Expected Results
- John Doe: AYR ≈ 0.57 (Green)
- Grace Okon: AYR ≈ 0.29 (Flag)
- Musa Adebayo: AYR = 0.30 (Watch)

---

## Status

✅ **COMPLETE**

All files updated, tests updated, mock data updated, and tooltip information updated.

---

## Next Steps

When integrating with backend:
1. Ensure API provides `par15MidMonth` field
2. Verify PAR15 calculation matches business definition
3. Confirm mid-month measurement timing (15th of month)
4. Update any reports or exports that reference AYR formula

---

**Updated**: 2025-10-17  
**Status**: ✅ Complete  
**Tested**: ✅ Yes (with mock data)

