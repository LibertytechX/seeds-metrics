# Quick Start Guide - Loan Officer Metrics Dashboard

## 🚀 Get Started in 2 Minutes

### Step 1: Install Dependencies
```bash
cd metrics-dashboard
npm install
```

### Step 2: Start Development Server
```bash
npm run dev
```

### Step 3: Open in Browser
Navigate to: **http://localhost:5173**

You should see the dashboard with:
- Blue header with filters
- 6 KPI cards showing portfolio metrics
- Three tabs with officer data

---

## 📊 Dashboard Overview

### What You're Looking At

**Header (Blue Bar)**
- Date range selector
- Branch filter
- Three toggles (Include Watch, DQI×CP, Show Red Only)
- Export buttons (CSV/PDF)

**KPI Strip (6 Cards)**
- Portfolio Overdue >15 Days
- Average DQI
- Average AYR
- Average Risk Score
- Top Performing Officer
- Watchlist Count

**Main Content (Tabs)**
1. **Credit Health Overview** - Portfolio-level metrics
2. **Officer Performance** - Sortable officer rankings
3. **Early Indicators** - FIMR, Slippage, Roll, FRR, Channel Purity

---

## 🎮 Try These Interactions

### 1. Filter by Branch
- Click the "Branch" dropdown in the header
- Select "Lagos Main"
- Watch the table update instantly

### 2. Toggle Filters
- Check "Show Red Only" to see only flagged officers
- Uncheck to see all officers

### 3. Sort Table
- Click any column header to sort
- Click again to reverse sort direction

### 4. Switch Tabs
- Click "Officer Performance" tab
- Click "Early Indicators" tab
- Notice each tab has different data

### 5. View Color Bands
- Look for colored badges in the tables
- 🟩 Green = Healthy
- 🟧 Watch = Monitor
- 🔴 Flag = Action needed

---

## 📈 Understanding the Metrics

### Risk Score (0-100)
- **Green (≥80)**: Officer is performing well
- **Watch (60-79)**: Monitor closely
- **Amber (40-59)**: Needs attention
- **Red (<40)**: Immediate action required

### AYR (Adjusted Yield Ratio)
- **Green (≥0.50)**: Good return relative to overdue exposure
- **Watch (0.30-0.49)**: Moderate efficiency
- **Flag (<0.30)**: Low returns, high risk

### DQI (Delinquency Quality Index, 0-100)
- **Green (≥75)**: High quality portfolio
- **Watch (65-74)**: Acceptable quality
- **Flag (<65)**: Quality concerns

### FIMR (First-Installment Miss Rate)
- **Green (≤3%)**: Low early default rate
- **Watch (3-6%)**: Moderate early defaults
- **Flag (>6%)**: High early default rate

---

## 🔍 Sample Data

The dashboard comes with 3 sample officers:

| Officer | Region | Risk Score | AYR | Status |
|---------|--------|-----------|-----|--------|
| John Doe | Lagos | 85 | 0.58 | 🟩 Green |
| Grace Okon | Abuja | 65 | 0.32 | 🟧 Watch |
| Musa Adebayo | Kano | 45 | 0.18 | 🔴 Flag |

---

## 💡 Tips

1. **Hover over values** - Tooltips show formulas and details
2. **Use filters together** - Combine branch + toggles for precise views
3. **Export for analysis** - CSV exports include all current filters
4. **Check timestamps** - "Last refresh" shows data freshness
5. **Color coding** - Scan for red/amber quickly to spot issues

---

## 🛠️ Troubleshooting

### Dashboard not loading?
```bash
# Clear cache and restart
rm -rf node_modules package-lock.json
npm install
npm run dev
```

### Port 5173 already in use?
```bash
# Use a different port
npm run dev -- --port 3000
```

### Seeing old data?
- Hard refresh: `Cmd+Shift+R` (Mac) or `Ctrl+Shift+R` (Windows)
- Clear browser cache

---

## 📚 Learn More

- **README_DASHBOARD.md** - Complete feature documentation
- **IMPLEMENTATION_SUMMARY.md** - Technical details
- **build guide.txt** - Business requirements
- **style guide.txt** - UI/UX specifications

---

## 🎯 Next Steps

1. **Explore the data** - Click through tabs and filters
2. **Try sorting** - Click column headers
3. **Check different branches** - See how data changes
4. **Review metrics** - Understand what each number means
5. **Plan integration** - Think about connecting to real data

---

## 📞 Support

For questions about:
- **Metrics**: See build guide.txt
- **UI/UX**: See style guide.txt
- **Code**: Check inline comments in src/
- **Features**: See README_DASHBOARD.md

---

**Happy analyzing! 📊**

