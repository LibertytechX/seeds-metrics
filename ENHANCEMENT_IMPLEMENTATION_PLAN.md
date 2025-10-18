# Dashboard Enhancement Implementation Plan

## Status: IN PROGRESS

### Completed Tasks ✅

1. **Mock Data Updates** ✅
   - Added `mockTeamMembers` array (7 team members)
   - Updated `mockAgentPerformance` with:
     - `auditStatus` field
     - `lastAuditDate` field  
     - `allTimeFimr` field (8.5%, 15.2%, 11.8%)
   - Updated all 12 `mockFIMRLoans` with:
     - `customerPhone` field (+234 format)
     - `fimrTagged: true` (all FIMR loans are tagged)
   - Updated all 12 `mockEarlyIndicatorLoans` with:
     - `customerPhone` field (+234 format)
     - `fimrTagged` field (5 true, 7 false - ~42% true)

### Remaining Tasks 🚧

#### 2. Update FIMRDrilldown.jsx
- [ ] Add `startDate` and `endDate` to filters state
- [ ] Add date range filter UI (2 date inputs)
- [ ] Update filteredLoans to filter by date range (disbursementDate)
- [ ] Add "Customer Phone Number" column (after Customer Name)
- [ ] Add "FIMR Tagged" column (after Status) with badge
- [ ] Update CSV export to include new columns
- [ ] Update column count: 15 → 17

#### 3. Update FIMRDrilldown.css
- [ ] Add styles for date range filter inputs
- [ ] Add styles for FIMR Tagged badge (true/false)
- [ ] Update table min-width for additional columns

#### 4. Update EarlyIndicatorsDrilldown.jsx
- [ ] Add `startDate` and `endDate` to filters state
- [ ] Add date range filter UI (2 date inputs)
- [ ] Update filteredLoans to filter by date range (disbursementDate)
- [ ] Add "Customer Phone Number" column (after Customer Name)
- [ ] Add "FIMR Tagged" column (after Status) with badge
- [ ] Update CSV export to include new columns
- [ ] Update column count: 17 → 19

#### 5. Update EarlyIndicatorsDrilldown.css
- [ ] Add styles for date range filter inputs
- [ ] Add styles for FIMR Tagged badge (true/false)
- [ ] Update table min-width for additional columns

#### 6. Update AgentPerformance.jsx
- [ ] Add `startDate` and `endDate` to filters state
- [ ] Add date range filter UI (2 date inputs)
- [ ] Update filteredAgents to filter by date range (lastAuditDate)
- [ ] Add "Audit Status" dropdown column (after Risk Band)
- [ ] Add "Last Audit Date" column (after Audit Status)
- [ ] Add "All-Time FIMR" column (after FIMR)
- [ ] Add "Action" dropdown column (last column)
- [ ] Create handleAuditStatusChange function
- [ ] Create handleActionSelect function with 4 actions:
  - Audit 20 Top Risk Loans
  - Freeze Disbursement
  - View Entire Portfolio
  - Export Entire Portfolio
- [ ] Update CSV export to include new columns
- [ ] Update column count: 19 → 23

#### 7. Update AgentPerformance.css
- [ ] Add styles for date range filter inputs
- [ ] Add styles for Audit Status dropdown
- [ ] Add styles for Action dropdown button
- [ ] Add styles for dropdown menus
- [ ] Update table min-width for additional columns

#### 8. Documentation
- [ ] Update NEW_DRILLDOWN_TABS_IMPLEMENTATION.md
- [ ] Update COMPLETE_DASHBOARD_SUMMARY.md
- [ ] Create ENHANCEMENTS_SUMMARY.md

---

## Implementation Details

### Date Range Filter Component (Reusable)

```jsx
<div className="filter-group date-range">
  <label>Date Range</label>
  <div className="date-inputs">
    <input
      type="date"
      value={filters.startDate}
      onChange={(e) => handleFilterChange('startDate', e.target.value)}
      placeholder="Start Date"
    />
    <span>to</span>
    <input
      type="date"
      value={filters.endDate}
      onChange={(e) => handleFilterChange('endDate', e.target.value)}
      placeholder="End Date"
    />
  </div>
</div>
```

### FIMR Tagged Badge Component

```jsx
<span className={`fimr-badge ${loan.fimrTagged ? 'fimr-true' : 'fimr-false'}`}>
  {loan.fimrTagged ? 'True' : 'False'}
</span>
```

### Audit Status Dropdown Component

```jsx
<select
  value={agent.auditStatus}
  onChange={(e) => handleAuditStatusChange(agent.officerName, e.target.value)}
  className="audit-status-select"
>
  {mockTeamMembers.map(member => (
    <option key={member.id} value={member.name}>
      {member.name}
    </option>
  ))}
</select>
```

### Action Dropdown Component

```jsx
<div className="action-dropdown">
  <button
    className="action-button"
    onClick={() => toggleActionMenu(agent.officerName)}
  >
    Actions ▼
  </button>
  {activeActionMenu === agent.officerName && (
    <div className="action-menu">
      <button onClick={() => handleAction(agent, 'audit20')}>
        Audit 20 Top Risk Loans
      </button>
      <button onClick={() => handleAction(agent, 'freeze')}>
        Freeze Disbursement
      </button>
      <button onClick={() => handleAction(agent, 'viewPortfolio')}>
        View Entire Portfolio
      </button>
      <button onClick={() => handleAction(agent, 'exportPortfolio')}>
        Export Entire Portfolio
      </button>
    </div>
  )}
</div>
```

### Action Handler Functions

```javascript
const handleAction = (agent, actionType) => {
  switch (actionType) {
    case 'audit20':
      console.log(`Auditing top 20 risk loans for ${agent.officerName}`);
      alert(`Audit 20 Top Risk Loans for ${agent.officerName}\n\nThis will open a modal with the top 20 highest-risk loans in this officer's portfolio.`);
      break;
    case 'freeze':
      console.log(`Freezing disbursement for ${agent.officerName}`);
      if (confirm(`Are you sure you want to freeze disbursement for ${agent.officerName}?`)) {
        alert(`Disbursement frozen for ${agent.officerName}`);
      }
      break;
    case 'viewPortfolio':
      console.log(`Viewing entire portfolio for ${agent.officerName}`);
      alert(`View Entire Portfolio for ${agent.officerName}\n\nThis will open a detailed view of all loans in this officer's portfolio.`);
      break;
    case 'exportPortfolio':
      console.log(`Exporting portfolio for ${agent.officerName}`);
      // Trigger CSV export of officer's entire portfolio
      exportOfficerPortfolio(agent);
      break;
    default:
      break;
  }
  setActiveActionMenu(null);
};
```

---

## Column Order Reference

### FIMR Drilldown (17 columns)
1. Loan ID
2. Officer Name
3. Region
4. Branch
5. Customer Name
6. **Customer Phone Number** ← NEW
7. Disbursement Date
8. Loan Amount
9. First Payment Due Date
10. Days Since Due
11. Amount Due (1st Installment)
12. Amount Paid
13. Outstanding Balance
14. Current DPD
15. Channel
16. Status
17. **FIMR Tagged** ← NEW

### Early Indicators Drilldown (19 columns)
1. Loan ID
2. Officer Name
3. Region
4. Branch
5. Customer Name
6. **Customer Phone Number** ← NEW
7. Disbursement Date
8. Loan Amount
9. Current DPD
10. Previous DPD Status
11. Days in Current Status
12. Amount Due
13. Amount Paid
14. Outstanding Balance
15. Channel
16. Status
17. **FIMR Tagged** ← NEW
18. Roll Direction
19. Last Payment Date

### Agent Performance (23 columns)
1. Officer Name
2. Region
3. Branch
4. Risk Score
5. Risk Band
6. **Audit Status** ← NEW (Dropdown)
7. **Last Audit Date** ← NEW
8. AYR
9. DQI
10. FIMR
11. **All-Time FIMR** ← NEW
12. D0-6 Slippage
13. Roll
14. FRR
15. Portfolio Total
16. Overdue >15D
17. Active Loans
18. Channel
19. Yield
20. PORR
21. Channel Purity
22. Rank
23. **Action** ← NEW (Dropdown)

---

## Filter Counts

### FIMR Drilldown
- Officer
- Region
- Branch
- Channel
- Status
- **Start Date** ← NEW
- **End Date** ← NEW
**Total: 7 filters**

### Early Indicators Drilldown
- Officer
- Region
- Branch
- Channel
- Status
- **Start Date** ← NEW
- **End Date** ← NEW
**Total: 7 filters**

### Agent Performance
- Region
- Branch
- Risk Band
- **Start Date** ← NEW
- **End Date** ← NEW
**Total: 5 filters**

---

## Next Steps

1. Update FIMRDrilldown.jsx and .css
2. Update EarlyIndicatorsDrilldown.jsx and .css
3. Update AgentPerformance.jsx and .css
4. Test all changes
5. Update documentation

**Estimated Time**: 2-3 hours
**Files to Modify**: 6 files
**New Features**: 11 new columns, 6 new filters, 2 dropdown components

