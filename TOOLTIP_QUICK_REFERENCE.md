# 🎯 Tooltip Quick Reference Card

## Where to Find Tooltips

### In Tables
- **Officer Performance Table** - Hover over: AYR, DQI, Risk Score, Overdue >15D, Yield
- **Credit Health Table** - Hover over: Overdue >15D, AYR, DQI, FIMR
- **Early Indicators Table** - Hover over: FIMR, D0-6 Slippage, Roll, FRR, Channel Purity

### In Tabs
- **Credit Health Overview** - Hover over tab name
- **Officer Performance** - Hover over tab name
- **Early Indicators** - Hover over tab name

---

## 📊 Metric Tooltips at a Glance

| Metric | What It Measures | Formula | Bands | Better |
|--------|------------------|---------|-------|--------|
| **FIMR** | First-installment misses | firstMiss / disbursed | ≤3% (G), 3-6% (W), >6% (F) | Lower |
| **D0-6 Slippage** | Early delinquency | dpd1to6Bal / amountDue7d | ≤5% (G), 5-8% (W), >8% (F) | Lower |
| **Roll** | Delinquency escalation | movedTo7to30 / prevDpd1to6Bal | ≤25% (G), 25-35% (W), >35% (F) | Lower |
| **FRR** | Fee collection | feesCollected / feesDue | N/A | Higher |
| **AYR** | Return vs overdue | (interest+fees) / (1+overdue15dRatio) | <0.30 (F), 0.30-0.49 (W), ≥0.50 (G) | Higher |
| **DQI** | Portfolio quality | 100 * (0.4*RQ + 0.35*OTI + 0.25*(1-FIMR)) * CP | <65 (F), 65-74 (W), ≥75 (G) | Higher |
| **Risk Score** | Officer risk | 100 - (penalties) | <40 (R), 40-59 (A), 60-79 (W), ≥80 (G) | Higher |

**Legend**: G=Green, W=Watch, F=Flag, R=Red, A=Amber

---

## 🎨 Tooltip Appearance

```
┌─────────────────────────────────────┐
│ Metric Name                         │
│                                     │
│ Description of what it measures     │
│                                     │
│ Formula: calculation method         │
│                                     │
│ Bands: Green | Watch | Flag         │
│                                     │
│ Interpretation and example          │
└─────────────────────────────────────┘
     ↑ Arrow points to metric
```

---

## 💡 Quick Tips

### How to Trigger Tooltips
1. **Desktop**: Hover mouse over metric header or tab name
2. **Mobile**: Tap on metric header or tab name
3. **Look for**: ℹ️ icon next to metric names

### What You'll See
- Full metric name
- What it measures
- How it's calculated
- Band thresholds
- Why it matters
- Example calculation

### Tooltip Behavior
- Appears after 0.2 seconds
- Disappears when you move away
- Shows arrow pointing to element
- Dark background for readability
- Fits on screen automatically

---

## 🔍 Finding Specific Metrics

### By Table

**Officer Performance Table**
- AYR → Return vs overdue exposure
- DQI → Portfolio quality score
- Risk Score → Officer risk assessment
- Overdue >15D → Material delinquency
- Yield → Revenue generation

**Credit Health Table**
- Overdue >15D → Material delinquency
- AYR → Return vs overdue exposure
- DQI → Portfolio quality score
- FIMR → First-installment misses

**Early Indicators Table**
- FIMR → First-installment misses
- D0-6 Slippage → Early delinquency
- Roll → Delinquency escalation
- FRR → Fee collection
- Channel Purity → Customer quality

### By Tab

**Credit Health Overview**
- Shows: Overdue >15D, AYR, DQI, FIMR
- Purpose: Monitor portfolio health

**Officer Performance**
- Shows: Risk Score, AYR, Yield, Overdue >15D, DQI
- Purpose: Compare officers

**Early Indicators**
- Shows: FIMR, D0-6 Slippage, Roll, FRR, Channel Purity
- Purpose: Detect early problems

---

## 📱 Mobile Usage

### On Phones/Tablets
1. Tap metric header or tab name
2. Tooltip appears
3. Read the information
4. Tap elsewhere to dismiss
5. Tooltip automatically sizes to fit screen

### Touch Targets
- Larger touch areas for easier tapping
- Tooltips positioned to fit on screen
- Readable font size on small screens

---

## 🎯 Common Questions Answered by Tooltips

### "What does AYR mean?"
→ Hover over AYR column header

### "How is Risk Score calculated?"
→ Hover over Risk Score column header

### "What's the difference between FIMR and D0-6 Slippage?"
→ Hover over both to compare

### "What metrics are in the Early Indicators tab?"
→ Hover over "Early Indicators" tab name

### "What does this band color mean?"
→ Hover over the metric to see band definitions

---

## 🔧 Customizing Tooltips

### To Add New Tooltip
1. Edit `src/utils/metricInfo.js`
2. Add entry to `metricInfo` object
3. Tooltip automatically appears

### To Edit Existing Tooltip
1. Edit `src/utils/metricInfo.js`
2. Update the metric information
3. Changes appear immediately

### To Change Tooltip Style
1. Edit `src/components/Tooltip.css`
2. Modify colors, sizing, animation
3. Changes appear immediately

---

## 📚 Related Documentation

- **TOOLTIP_GUIDE.md** - Complete tooltip documentation
- **README_DASHBOARD.md** - Dashboard features
- **build guide.txt** - Business requirements
- **style guide.txt** - UI/UX specifications

---

## ✨ Tooltip Features

✅ Hover-activated  
✅ Professional design  
✅ Dark theme  
✅ Smooth animation  
✅ Arrow pointers  
✅ Mobile responsive  
✅ Keyboard accessible  
✅ Auto-positioning  
✅ Complete information  
✅ Easy to customize  

---

## 🚀 Try These Now

1. **Hover over "AYR"** in Officer Performance table
2. **Hover over "Risk Score"** in Officer Performance table
3. **Hover over "FIMR"** in Early Indicators table
4. **Hover over "Credit Health Overview"** tab
5. **Hover over "Officer Performance"** tab

---

**Tooltips make the dashboard self-documenting!**

**Hover over any metric to learn what it means.**

