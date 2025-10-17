# 📚 Tooltip Guide - Metric Information

## Overview

The dashboard includes comprehensive tooltips that provide detailed information about every metric. Hover over any metric header or tab to see detailed explanations.

---

## 🎯 How to Use Tooltips

### Metric Headers
- **Location**: Column headers in all tables
- **How to Access**: Hover over the metric name or the ℹ️ icon
- **Information Shown**:
  - Full metric name
  - Description
  - Formula
  - Band thresholds
  - Interpretation
  - Example calculation

### Tab Headers
- **Location**: Tab names at the top of the main content area
- **How to Access**: Hover over the tab name or the ℹ️ icon
- **Information Shown**:
  - Tab name
  - Description
  - Purpose
  - Metrics included in the tab

---

## 📊 Metric Tooltips

### FIMR (First-Installment Miss Rate)
**Formula**: `FIMR = firstMiss / disbursed`

**What it measures**: Proportion of newly disbursed loans whose first installment was missed.

**Why it matters**: Early indicator of onboarding, KYC, or guarantor issues.

**Bands**:
- 🟩 Green: ≤ 3%
- 🟧 Watch: 3% - 6%
- 🔴 Flag: > 6%

**Interpretation**: Lower is better. High FIMR indicates problems with loan origination or customer quality.

**Example**: If 150 out of 5,000 disbursed loans missed first payment, FIMR = 3%

---

### D0-6 Slippage (Early Slippage)
**Formula**: `D0-6 Slippage = dpd1to6Bal / amountDue7d`

**What it measures**: Portion of amount due in next 7 days that is already 1-6 days past due.

**Why it matters**: Early sign of repayment friction or channel loss.

**Bands**:
- 🟩 Green: ≤ 5%
- 🟧 Watch: 5% - 8%
- 🔴 Flag: > 8%

**Interpretation**: Lower is better. Shows how much of near-term payments are already slipping.

**Example**: If ₦250K is 1-6 days overdue out of ₦5M due in 7 days, Slippage = 5%

---

### Roll (Roll 0-6 → 7-30)
**Formula**: `Roll = movedTo7to30 / prevDpd1to6Bal`

**What it measures**: Share of early delinquency (1-6 days) that worsened into 7-30 days past due.

**Why it matters**: Shows containment failure from early lateness to material delinquency.

**Bands**:
- 🟩 Green: ≤ 25%
- 🟧 Watch: 25% - 35%
- 🔴 Flag: > 35%

**Interpretation**: Lower is better. Shows how well early delinquency is being managed.

**Example**: If ₦180K of ₦720K early delinquency moved to 7-30 days, Roll = 25%

---

### FRR (Fees Realization Rate)
**Formula**: `FRR = feesCollected / feesDue`

**What it measures**: Proportion of expected fees that are actually collected.

**Why it matters**: Fee collection efficiency and potential system/officer issues.

**Interpretation**: Higher is better. Shortfalls reduce net yield and indicate collection issues.

**Example**: If ₦450K collected out of ₦500K due, FRR = 90%

---

### AYR (Adjusted Yield Ratio)
**Formula**: `AYR = (interestCollected + feesCollected) / (1 + overdue15dRatio)`

**What it measures**: Return generated relative to material overdue exposure (>15 days).

**Why it matters**: Shows economic efficiency while accounting for problem loans.

**Bands**:
- 🔴 Flag: < 0.30
- 🟧 Watch: 0.30 - 0.49
- 🟩 Green: ≥ 0.50

**Interpretation**: Higher is better. Shows return generation relative to overdue exposure.

**Example**: If ₦2.55M collected and 2.4% of portfolio is overdue >15 days, AYR ≈ 0.58

---

### DQI (Delinquency Quality Index)
**Formula**: `DQI = 100 * (0.4*RQ + 0.35*OTI + 0.25*(1-FIMR)) * CP_toggle`

**What it measures**: Composite index capturing loan quality and repayment discipline (0-100).

**Why it matters**: Overall portfolio quality combining risk, on-time rate, and early defaults.

**Bands**:
- 🔴 Flag: < 65
- 🟧 Watch: 65 - 74
- 🟩 Green: ≥ 75

**Components**:
- 40% - Risk Quality (normalized risk score)
- 35% - On-Time Rate (repayment discipline)
- 25% - First-Installment Miss Rate (early defaults)
- Channel Purity multiplier (optional toggle)

**Interpretation**: Higher is better. Composite measure of portfolio health.

**Example**: With RQ=0.85, OTI=0.92, FIMR=0.03, CP=0.95, DQI = 82

---

### Risk Score (Composite Officer Risk Score)
**Formula**: `RiskScore = 100 - (penalties for various risk factors)`

**What it measures**: Single number combining portfolio risk, behavior signals, and integrity (0-100).

**Why it matters**: Overall officer risk assessment across multiple dimensions.

**Bands**:
- 🔴 Flag: < 40
- 🟧 Amber: 40 - 59
- 🟨 Watch: 60 - 79
- 🟩 Green: ≥ 80

**Factors Considered**:
- Portfolio Open Risk Ratio (20 points)
- First-Installment Miss Rate (15 points)
- Delinquency Roll (10 points)
- Waiver volume (10 points)
- Backdated entries (10 points)
- Reversals (10 points)
- Fee Realization Rate (10 points)
- Channel Purity (5 points)
- Float/Settlement Gap (10 points)

**Interpretation**: Higher is better. Comprehensive risk indicator.

**Example**: Officer with good metrics across all factors scores 85 (Green)

---

## 📑 Tab Tooltips

### Credit Health Overview
**Description**: Portfolio-level metrics showing overall credit quality and delinquency trends.

**Purpose**: Monitor portfolio health and identify trends.

**Metrics**: Overdue >15D, AYR, DQI, FIMR

---

### Officer Performance
**Description**: Officer-level rankings and metrics showing productivity and risk.

**Purpose**: Compare officers and identify top performers and problem areas.

**Metrics**: Risk Score, AYR, Yield, Overdue >15D, DQI

---

### Early Indicators
**Description**: Early warning metrics that predict future delinquency.

**Purpose**: Detect early signs of problems before they become material.

**Metrics**: FIMR, D0-6 Slippage, Roll, FRR, Channel Purity

---

## 💡 Tips for Using Tooltips

1. **Hover over any metric header** to see detailed information
2. **Look for the ℹ️ icon** next to metric names
3. **Hover over tab names** to understand what each tab shows
4. **Use tooltips to learn** about metrics you're unfamiliar with
5. **Reference tooltips** when explaining metrics to others

---

## 🎨 Tooltip Styling

- **Dark background** - Easy to read on any background
- **Positioned intelligently** - Appears where there's space
- **Smooth animation** - Fades in smoothly
- **Arrow pointer** - Shows which element the tooltip refers to
- **Responsive** - Adapts to screen size

---

## 📱 Mobile Tooltips

On mobile devices:
- Tooltips appear on tap instead of hover
- Positioned to fit on screen
- Larger touch targets for easier interaction
- Dismiss by tapping elsewhere

---

## 🔧 Customizing Tooltips

To add or modify tooltip information:

1. Edit `src/utils/metricInfo.js`
2. Update the `metricInfo` object with new information
3. Tooltips will automatically update

Example:
```javascript
export const metricInfo = {
  myMetric: {
    name: 'My Metric',
    fullName: 'My Metric Full Name',
    description: 'What it does',
    formula: 'How to calculate it',
    // ... more fields
  }
};
```

---

## 📚 Related Documentation

- **README_DASHBOARD.md** - Complete feature guide
- **build guide.txt** - Business requirements
- **style guide.txt** - UI/UX specifications

---

**Tooltips help you understand every metric at a glance!**

