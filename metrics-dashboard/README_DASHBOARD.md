# Loan Officer Metrics Dashboard

A comprehensive React-based dashboard for monitoring loan officer risk and productivity metrics, built according to the Build Guide and Style Guide specifications.

## 🚀 Features

### Core Metrics
- **FIMR** (First-Installment Miss Rate) - Early indicator of onboarding issues
- **D0-6 Slippage** - Early repayment friction detection
- **Roll 0-6 → 7-30** - Delinquency escalation tracking
- **FRR** (Fees Realization Rate) - Fee collection efficiency
- **AYR** (Adjusted Yield Ratio) - Return vs. overdue exposure
- **DQI** (Delinquency Quality Index) - Composite loan quality score (0-100)
- **Risk Score** - Comprehensive officer risk assessment (0-100)

### Dashboard Components

#### 1. **Header Toolbar** (Persistent)
- Date range selector (Week/Month/Quarter)
- Multi-select filters (Branch, Officer, Product, Region, Channel, Risk Band)
- Global toggles:
  - "Include Watch" - Show watch-list items
  - "DQI×CP" - Apply channel purity multiplier
  - "Show Red Only" - Filter to flagged items only
- Export buttons (CSV/PDF)
- Last refresh timestamp

#### 2. **KPI Overview Strip**
Six horizontally scrollable cards showing:
- Portfolio Overdue >15 Days
- Average DQI
- Average AYR
- Risk Score (Avg)
- Top Performing Officer
- Watchlist Count

#### 3. **Tabbed Data Tables**

**Tab 1: Credit Health Overview**
- Portfolio totals, overdue values, AYR, FIMR, DQI
- Drill-down to loan-level list
- Export functionality

**Tab 2: Officer Performance**
- Officer name, Region, Risk Score, AYR, Yield, Overdue >15D, Rank
- Sortable columns
- Compare officers side-by-side
- Export with filters

**Tab 3: Early Indicators**
- FIMR, D0-6 Slippage, Roll, FRR, Channel Purity
- Trend line toggle
- Export visualization snapshots

### Color-Coded Risk Bands
- 🟩 **Green** - Healthy/Efficient
- 🟧 **Watch** - Monitor closely
- 🔴 **Flag** - Requires action

## 📊 Metric Formulas

### FIMR
```
FIMR = firstMiss / disbursed
Bands: Green ≤ 3% | Watch 3-6% | Flag > 6%
```

### D0-6 Slippage
```
D0-6 Slippage = dpd1to6Bal / amountDue7d
Bands: Green ≤ 5% | Watch 5-8% | Flag > 8%
```

### Roll 0-6 → 7-30
```
Roll = movedTo7to30 / prevDpd1to6Bal
Bands: Green ≤ 25% | Watch 25-35% | Flag > 35%
```

### AYR (Normalized)
```
AYR = (interestCollected + feesCollected) / (1 + overdue15dRatio)
where overdue15dRatio = overdue15d / totalPortfolio
Bands: Flag < 0.30 | Watch 0.30-0.49 | Green ≥ 0.50
```

### DQI
```
DQI = round(100 * (0.4*RQ + 0.35*OTI + 0.25*(1-FIMR)) * CP_toggle)
Bands: Flag < 65 | Watch 65-74 | Green ≥ 75
```

### Risk Score
```
RiskScore = 100 - 20*PORR - 15*FIMR - 10*Roll - 10*(waivers/amountDue7d)
            - 10*(backdated/entries) - 10*(reversals/entries) - 10*(1-FRR)
            - 5*(1-channelPurity) - 10*(hadFloatGap ? 1 : 0)
Bands: Red < 40 | Amber 40-59 | Watch 60-79 | Green ≥ 80
```

## 🛠️ Tech Stack

- **Frontend Framework**: React 18 with Vite
- **UI Components**: Custom CSS + Lucide React icons
- **Charts**: Recharts (for future enhancements)
- **Date Handling**: date-fns
- **Styling**: CSS3 with responsive design

## 📁 Project Structure

```
src/
├── components/
│   ├── Header.jsx          # Filter toolbar
│   ├── Header.css
│   ├── KPIStrip.jsx        # KPI cards
│   ├── KPIStrip.css
│   ├── DataTables.jsx      # Tabbed tables
│   └── DataTables.css
├── utils/
│   ├── metrics.js          # Metric calculation functions
│   └── mockData.js         # Sample data
├── App.jsx                 # Main app component
├── App.css
├── index.css               # Global styles
└── main.jsx
```

## 🚀 Getting Started

### Installation
```bash
cd metrics-dashboard
npm install
```

### Development
```bash
npm run dev
```
The app will be available at `http://localhost:5173`

### Build for Production
```bash
npm run build
```

## 📋 Usage

### Filtering Data
1. Select a date range from the header
2. Choose a branch to filter officers
3. Toggle "Include Watch" to show watch-list items
4. Toggle "Show Red Only" to see only flagged officers

### Viewing Metrics
1. Click on tabs to switch between different metric views
2. Click column headers to sort tables
3. Hover over metrics to see tooltips with formulas
4. Click "View" or "Drill Down" buttons to see loan-level details

### Exporting Data
1. Click "CSV" or "PDF" button in the header
2. Export includes current filters, timestamp, and metric definitions
3. All exports are auditable (logged with user and timestamp)

## 🔄 State Management

The dashboard uses React hooks for state management:
- `filters` - Current filter selections
- `activeTab` - Currently selected tab
- `lastRefresh` - Last data refresh timestamp

## 🎨 Styling

The dashboard uses a professional color scheme:
- Primary Blue: `#3b82f6`
- Success Green: `#10b981`
- Warning Yellow: `#f59e0b`
- Danger Red: `#ef4444`
- Background: `#f0f4f8`

## 📝 Future Enhancements

- [ ] Real-time data integration via API
- [ ] Advanced charting (AYR vs Risk scatter plot)
- [ ] Drilldown modals for loan-level details
- [ ] Export to Excel with multiple sheets
- [ ] Audit trail logging
- [ ] User authentication
- [ ] Threshold configuration UI
- [ ] Historical trend analysis

## 📚 References

- Build Guide: Comprehensive metric definitions and business rules
- Style Guide: UI/UX patterns and layout specifications
- Metric Formulas: All calculations follow the canonical definitions

## 🤝 Contributing

When adding new features:
1. Follow the existing component structure
2. Update metric calculations in `utils/metrics.js`
3. Add corresponding tests
4. Update this README

## 📄 License

Internal use only - Seeds Metrics Dashboard

