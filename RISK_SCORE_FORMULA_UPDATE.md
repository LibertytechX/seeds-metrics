# Risk Score Formula Update - Migration from 9 to 5 Components

**Date**: 2025-11-04  
**Status**: ✅ COMPLETED

---

## 📋 Summary

Updated the Risk Score calculation formula by **removing 6 old penalty components** and **adding 2 new penalty components** to better reflect portfolio quality, repayment behavior, and revenue generation.

---

## 🔄 Changes Overview

### **REMOVED Components (55 points total)**
1. ❌ **Waivers** (10 points max)
2. ❌ **Backdated Entries** (10 points max)
3. ❌ **Reversals** (10 points max)
4. ❌ **FRR - Fees Realization Rate** (10 points max)
5. ❌ **Channel Purity** (5 points max)
6. ❌ **Float Gap** (10 points max)

### **ADDED Components (55 points total)**
1. ✅ **Repayment Delay Rate** (40 points max)
2. ✅ **AYR - Adjusted Yield Ratio** (15 points max)

### **RETAINED Components (45 points total)**
1. ✅ **PORR - Portfolio Open Risk Ratio** (20 points max)
2. ✅ **FIMR - First Installment Miss Rate** (15 points max)
3. ✅ **Roll - Delinquency Roll (0-6 → 7-30)** (10 points max)

---

## 📐 New Risk Score Formula

### **Formula**
```
RiskScore = 100 
  - 20 * PORR
  - 15 * FIMR
  - 10 * Roll
  - 40 * (1 - RepaymentDelayRate / 100)
  - 15 * (1 - min(AYR, 1.0))
```

### **Penalty Breakdown**

| Component | Max Penalty | Formula | Description |
|-----------|-------------|---------|-------------|
| **PORR** | 20 points | `20 * PORR` | Portfolio at Risk Ratio (overdue 15+ days) |
| **FIMR** | 15 points | `15 * FIMR` | First Installment Miss Rate |
| **Roll** | 10 points | `10 * Roll` | Delinquency escalation (D1-6 → D7-30) |
| **Repayment Delay Rate** | 40 points | `40 * (1 - RepaymentDelayRate/100)` | Payment frequency and consistency |
| **AYR** | 15 points | `15 * (1 - min(AYR, 1.0))` | Revenue generation efficiency |
| **TOTAL** | **100 points** | | |

---

## 🎯 Penalty Calculation Examples

### **Example 1: High-Performing Officer**
```
PORR = 0.05 (5%)
FIMR = 0.02 (2%)
Roll = 0.15 (15%)
Repayment Delay Rate = 85%
AYR = 0.60

Risk Score = 100
  - 20 * 0.05 = -1.0
  - 15 * 0.02 = -0.3
  - 10 * 0.15 = -1.5
  - 40 * (1 - 0.85) = -6.0
  - 15 * (1 - 0.60) = -6.0
  
Risk Score = 100 - 14.8 = 85.2 ≈ 85 (Green Band)
```

### **Example 2: Average Officer**
```
PORR = 0.15 (15%)
FIMR = 0.05 (5%)
Roll = 0.30 (30%)
Repayment Delay Rate = 60%
AYR = 0.40

Risk Score = 100
  - 20 * 0.15 = -3.0
  - 15 * 0.05 = -0.75
  - 10 * 0.30 = -3.0
  - 40 * (1 - 0.60) = -16.0
  - 15 * (1 - 0.40) = -9.0
  
Risk Score = 100 - 31.75 = 68.25 ≈ 68 (Watch Band)
```

### **Example 3: At-Risk Officer**
```
PORR = 0.30 (30%)
FIMR = 0.10 (10%)
Roll = 0.50 (50%)
Repayment Delay Rate = 30%
AYR = 0.20

Risk Score = 100
  - 20 * 0.30 = -6.0
  - 15 * 0.10 = -1.5
  - 10 * 0.50 = -5.0
  - 40 * (1 - 0.30) = -28.0
  - 15 * (1 - 0.20) = -12.0
  
Risk Score = 100 - 52.5 = 47.5 ≈ 48 (Amber Band)
```

---

## 🎨 Risk Score Bands (UNCHANGED)

| Band | Score Range | Color | Interpretation |
|------|-------------|-------|----------------|
| **Green** | 80 - 100 | 🟢 | Low risk, high performance |
| **Watch** | 60 - 79 | 🟡 | Moderate risk, needs monitoring |
| **Amber** | 40 - 59 | 🟠 | High risk, requires intervention |
| **Red** | 0 - 39 | 🔴 | Critical risk, immediate action needed |

---

## 📂 Files Modified

### **Backend**
1. ✅ `backend/internal/services/metrics_service.go`
   - Updated `CalculateRiskScoreNorm()` function
   - Removed: Waivers, Backdated, Reversals, FRR, Channel Purity, Float Gap penalties
   - Added: Repayment Delay Rate (40 max), AYR (15 max) penalties
   - Updated `CalculateDQI()` function to remove Channel Purity component

### **Frontend**
2. ✅ `metrics-dashboard/src/utils/metrics.js`
   - Updated `calculateRiskScore()` function
   - Updated `calculateDQI()` function (removed Channel Purity parameter)
   - Simplified function signatures

3. ✅ `metrics-dashboard/src/utils/metricInfo.js`
   - Updated Risk Score tooltip documentation
   - Updated DQI tooltip documentation
   - Updated Early Indicators tab info
   - Updated Agent Performance tab info

---

## 🔍 Technical Details

### **Repayment Delay Rate Penalty (40 points max)**

**Formula:**
```javascript
if (repaymentDelayRate <= 100) {
  penalty = (1 - (repaymentDelayRate / 100)) * 40;
  penalty = Math.min(40, Math.max(0, penalty)); // Cap at 40
}
```

**Examples:**
- `RepaymentDelayRate = 100%` → Penalty = 0 points (perfect)
- `RepaymentDelayRate = 75%` → Penalty = 10 points
- `RepaymentDelayRate = 50%` → Penalty = 20 points
- `RepaymentDelayRate = 25%` → Penalty = 30 points
- `RepaymentDelayRate = 0%` → Penalty = 40 points (maximum)
- `RepaymentDelayRate < 0%` → Penalty = 40 points (capped)
- `RepaymentDelayRate > 100%` → Penalty = 0 points (better than expected)

---

### **AYR Penalty (15 points max)**

**Formula:**
```javascript
const ayrCapped = Math.min(ayr, 1.0);
penalty = (1 - ayrCapped) * 15;
```

**Examples:**
- `AYR = 1.0 or higher` → Penalty = 0 points (excellent)
- `AYR = 0.75` → Penalty = 3.75 points
- `AYR = 0.50` → Penalty = 7.5 points
- `AYR = 0.25` → Penalty = 11.25 points
- `AYR = 0.0` → Penalty = 15 points (maximum)

---

## 📊 DQI Formula Update

The DQI (Delinquency Quality Index) was also updated to remove the Channel Purity component:

### **Old DQI Formula**
```
DQI = 100 * (0.40*RQ + 0.35*OTI + 0.25*(1-FIMR)) * CP
```

### **New DQI Formula**
```
DQI = 100 * (0.50*RQ + 0.35*OTI + 0.15*(1-FIMR))
```

**Changes:**
- Risk Quality (RQ) weight: 40% → **50%** (increased)
- On-Time Rate (OTI) weight: 35% → **35%** (unchanged)
- FIMR weight: 25% → **15%** (decreased)
- Channel Purity: **REMOVED**

---

## ✅ Testing Checklist

- [x] Backend Risk Score calculation updated
- [x] Frontend Risk Score calculation updated
- [x] Backend DQI calculation updated
- [x] Frontend DQI calculation updated
- [x] Risk Score tooltip documentation updated
- [x] DQI tooltip documentation updated
- [x] No TypeScript/JavaScript errors
- [x] No Go compilation errors
- [ ] Manual testing with sample data
- [ ] Verify Risk Score bands display correctly
- [ ] Verify tooltip content is accurate

---

## 🚀 Deployment Notes

1. **Backend Changes**: Requires backend restart to apply new Risk Score calculation
2. **Frontend Changes**: Requires frontend rebuild and deployment
3. **Data Migration**: No database migration needed - calculations are done in real-time
4. **Backward Compatibility**: Risk Score values will change for all officers after deployment

---

## 📈 Expected Impact

### **Risk Score Distribution Changes**
- Officers with **good repayment behavior** (high Repayment Delay Rate) will see **higher scores**
- Officers with **strong revenue generation** (high AYR) will see **higher scores**
- Officers previously penalized for waivers/backdated/reversals will see **score increases**
- Officers with **poor repayment frequency** will see **significant score decreases** (up to 40 points)

### **Business Benefits**
1. **Better Focus**: Emphasizes repayment behavior (40 points) over administrative issues
2. **Revenue Alignment**: Rewards officers who generate strong yields (15 points)
3. **Simplified Metrics**: Reduces from 9 to 5 components for easier interpretation
4. **Actionable Insights**: Officers can improve scores by focusing on repayment frequency and yield

---

## 📝 Next Steps

1. ✅ Deploy backend changes to production
2. ✅ Deploy frontend changes to production
3. ⏳ Monitor Risk Score distribution after deployment
4. ⏳ Communicate changes to stakeholders
5. ⏳ Update training materials and documentation
6. ⏳ Gather feedback from users

---

**End of Document**

