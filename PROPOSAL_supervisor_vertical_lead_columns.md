# Proposal: Add Supervisor Email and Vertical Lead Email to Agent Performance Table

**Date**: 2025-11-09  
**Feature**: Add two new columns to the Agent Performance table  
**Status**: Proposal - Awaiting Approval  

---

## Overview

Add two new columns to the Agent Performance table in the Seeds Metrics application:
1. **Supervisor Email** - Email address of the branch supervisor for each loan officer
2. **Vertical Lead Email** - Email address of the vertical lead for each loan officer

---

## Data Source Analysis

### verticals.tsv Structure

The `verticals.tsv` file contains 13 columns with the following relevant fields:

| Column # | Field Name | Description | Sample Value |
|----------|------------|-------------|--------------|
| 1 | `loan_officer_id` | Django customer_user_id | 3513 |
| 2 | `loan_officer_email` | Officer email | josephineakinfoyewa@gmail.com |
| 3 | `loan_officer_name` | Officer name | JOSEPHINE AKINFOYEWA |
| 9 | **`branch_supervisor_email`** | **Supervisor email** | **boluwatifer35@gmail.com** |
| 10 | `branch_supervisor_name` | Supervisor name | boluwatife racheal |
| 12 | **`vertical_lead_email`** | **Vertical lead email** | **taiwolawalyet@gmail.com** |
| 13 | `vertical_lead_name` | Vertical lead name | TAIWO LAWAL |

### Officer Mapping

We have a complete mapping file at `/tmp/officer_mapping.txt` that maps:
- Django `customer_user_id` (from verticals.tsv column 1) → Seeds Metrics `officer_id` (via email matching)
- **100% match rate** for all 550 valid officers

**Mapping Format**:
```
customer_user_id|django_email|seeds_officer_id|seeds_email|match_status
3513|josephineayodele2@gmail.com|678|josephineayodele2@gmail.com|MATCHED
```

---

## Current Database Schema

### officers Table

```sql
Table "public.officers"
      Column       |            Type             
-------------------+-----------------------------
 officer_id        | character varying(50)       -- PRIMARY KEY
 officer_name      | character varying(255)      
 officer_phone     | character varying(20)       
 officer_email     | character varying(255)      
 region            | character varying(100)      
 branch            | character varying(100)      
 employment_status | character varying(50)       
 hire_date         | date                        
 termination_date  | date                        
 primary_channel   | character varying(50)       
 created_at        | timestamp without time zone 
 updated_at        | timestamp without time zone 
 user_type         | character varying(100)      
```

**Missing Fields**: `supervisor_email`, `vertical_lead_email`

---

## Proposed Changes

### 1. Database Schema Changes

Add two new columns to the `officers` table:

```sql
ALTER TABLE officers 
ADD COLUMN supervisor_email VARCHAR(255),
ADD COLUMN vertical_lead_email VARCHAR(255);

-- Add indexes for performance
CREATE INDEX idx_officers_supervisor_email ON officers(supervisor_email);
CREATE INDEX idx_officers_vertical_lead_email ON officers(vertical_lead_email);
```

### 2. Data Population

Populate the new columns using the verticals.tsv data and officer mapping:

```sql
-- Create temporary table with verticals data
CREATE TEMP TABLE temp_verticals (
    django_customer_user_id VARCHAR(50),
    loan_officer_email VARCHAR(255),
    branch_supervisor_email VARCHAR(255),
    vertical_lead_email VARCHAR(255)
);

-- Load data from verticals.tsv (via script)
-- Then update officers table using email matching

UPDATE officers o
SET 
    supervisor_email = tv.branch_supervisor_email,
    vertical_lead_email = tv.vertical_lead_email
FROM temp_verticals tv
WHERE o.officer_email = tv.loan_officer_email;
```

### 3. Backend Model Changes

**File**: `backend/internal/models/dashboard.go`

Update the `DashboardOfficerMetrics` struct:

```go
type DashboardOfficerMetrics struct {
    ID                int                `json:"id"`
    OfficerID         string             `json:"officer_id"`
    Name              string             `json:"name"`
    Email             string             `json:"email,omitempty"`
    Region            string             `json:"region"`
    Branch            string             `json:"branch"`
    Channel           string             `json:"channel"`
    UserType          *string            `json:"user_type,omitempty"`
    HireDate          *time.Time         `json:"hire_date,omitempty"`
    SupervisorEmail   *string            `json:"supervisor_email,omitempty"`      // NEW
    VerticalLeadEmail *string            `json:"vertical_lead_email,omitempty"`   // NEW
    RawMetrics        *RawMetrics        `json:"rawMetrics"`
    CalculatedMetrics *CalculatedMetrics `json:"calculatedMetrics"`
    RiskBand          string             `json:"riskBand"`
}
```

### 4. Backend Repository Changes

**File**: `backend/internal/repository/dashboard_repository.go`

Update the `GetOfficers` query to SELECT the new fields:

```go
// In GetOfficers function, update the SELECT clause:
SELECT
    o.officer_id,
    o.officer_name,
    o.region,
    o.branch,
    o.primary_channel,
    o.user_type,
    o.hire_date,
    o.supervisor_email,        -- NEW
    o.vertical_lead_email,     -- NEW
    ...
```

Update the row scanning logic:

```go
var supervisorEmail, verticalLeadEmail sql.NullString

err := rows.Scan(
    &officer.OfficerID,
    &officer.Name,
    &officer.Region,
    &officer.Branch,
    &officer.Channel,
    &officer.UserType,
    &officer.HireDate,
    &supervisorEmail,          // NEW
    &verticalLeadEmail,        // NEW
    ...
)

// Null checking
if supervisorEmail.Valid {
    officer.SupervisorEmail = &supervisorEmail.String
}
if verticalLeadEmail.Valid {
    officer.VerticalLeadEmail = &verticalLeadEmail.String
}
```

### 5. Frontend Component Changes

**File**: `metrics-dashboard/src/components/AgentPerformance.jsx`

Add two new columns to the table:

**Table Headers** (around line 300-400):
```jsx
<th onClick={() => handleSort('supervisorEmail')}>
  Supervisor Email
  {sortConfig.key === 'supervisorEmail' && (
    <span className="sort-indicator">
      {sortConfig.direction === 'asc' ? ' ▲' : ' ▼'}
    </span>
  )}
</th>
<th onClick={() => handleSort('verticalLeadEmail')}>
  Vertical Lead Email
  {sortConfig.key === 'verticalLeadEmail' && (
    <span className="sort-indicator">
      {sortConfig.direction === 'asc' ? ' ▲' : ' ▼'}
    </span>
  )}
</th>
```

**Table Body** (around line 400-500):
```jsx
<td className="email">{agent.supervisorEmail || 'N/A'}</td>
<td className="email">{agent.verticalLeadEmail || 'N/A'}</td>
```

**CSV Export** (update headers and data rows):
```jsx
const headers = [
  'Officer ID', 'Name', 'Email', 'Region', 'Branch', 'User Type',
  'Supervisor Email', 'Vertical Lead Email',  // NEW
  'Risk Score', 'Risk Band', 'FIMR', 'Slippage', ...
];

const rows = filteredAgents.map(agent => [
  agent.officer_id,
  agent.name,
  agent.email,
  agent.region,
  agent.branch,
  agent.userType,
  agent.supervisorEmail || 'N/A',      // NEW
  agent.verticalLeadEmail || 'N/A',    // NEW
  agent.riskScore,
  ...
]);
```

---

## Implementation Steps

### Phase 1: Database Migration (Production)

1. **Create migration script**: `migrations/031_add_supervisor_vertical_lead_emails.sql`
2. **Add columns** to officers table
3. **Create indexes** for performance
4. **Test migration** on production database

### Phase 2: Data Population

1. **Create data loading script**: `load_verticals_data.sh`
2. **Parse verticals.tsv** and officer mapping
3. **Generate UPDATE statements** for each officer
4. **Execute updates** on production database
5. **Verify data** - check coverage and accuracy

### Phase 3: Backend Changes

1. **Update model** - Add fields to `DashboardOfficerMetrics`
2. **Update repository** - Modify SQL query and scanning logic
3. **Test API** - Verify new fields are returned
4. **Build and deploy** backend to production

### Phase 4: Frontend Changes

1. **Update component** - Add columns to AgentPerformance table
2. **Update exports** - Include new fields in CSV/PDF exports
3. **Test UI** - Verify columns display correctly
4. **Build and deploy** frontend to production

### Phase 5: Verification

1. **API testing** - Verify `/api/v1/officers` returns new fields
2. **UI testing** - Verify columns display in Agent Performance table
3. **Data validation** - Spot-check supervisor/vertical lead emails
4. **Export testing** - Verify CSV/PDF exports include new columns

---

## Expected Results

### Sample API Response

```json
{
  "status": "success",
  "data": {
    "officers": [
      {
        "officer_id": "678",
        "name": "JOSEPHINE AKINFOYEWA",
        "email": "josephineayodele2@gmail.com",
        "region": "Nigeria",
        "branch": "SABO",
        "supervisor_email": "boluwatifer35@gmail.com",
        "vertical_lead_email": "taiwolawalyet@gmail.com",
        "riskScore": 45,
        "riskBand": "Medium",
        ...
      }
    ]
  }
}
```

### Sample UI Display

| Officer ID | Name | Email | Branch | Supervisor Email | Vertical Lead Email | Risk Score |
|------------|------|-------|--------|------------------|---------------------|------------|
| 678 | JOSEPHINE AKINFOYEWA | josephineayodele2@gmail.com | SABO | boluwatifer35@gmail.com | taiwolawalyet@gmail.com | 45 |
| 510 | OLUCHI NWAKUNA | silvianwakuna@gmail.com | BARIGA | ajayiesther049@gmail.com | taiwolawalyet@gmail.com | 52 |

---

## Data Coverage Estimate

Based on our mapping analysis:
- **558 officers** in verticals.tsv
- **550 valid officers** (98.57% match with Django)
- **550 officers** mapped to Seeds Metrics (100% of valid)
- **Estimated coverage**: ~550 out of 6,800 officers (~8%)

**Note**: Only officers in the verticals.tsv file will have supervisor/vertical lead emails populated. Other officers will show "N/A".

---

## Risks and Mitigation

### Risk 1: Data Quality
**Issue**: Some officers may not have supervisor/vertical lead assigned  
**Mitigation**: Display "N/A" for missing values, don't break the UI

### Risk 2: Email Changes
**Issue**: Supervisor/vertical lead emails may change over time  
**Mitigation**: Create a process to periodically update from verticals.tsv

### Risk 3: Performance Impact
**Issue**: Adding columns may slow down queries  
**Mitigation**: Add indexes on new columns, monitor query performance

---

## Questions for Review

1. **Column Names**: Are `supervisor_email` and `vertical_lead_email` acceptable names?
2. **Data Updates**: How often should we refresh this data from verticals.tsv?
3. **Missing Data**: Is "N/A" acceptable for officers not in verticals.tsv?
4. **Additional Fields**: Should we also add `supervisor_name` and `vertical_lead_name`?
5. **Filtering**: Should we add filters for supervisor/vertical lead in the UI?

---

## Approval Required

Please review this proposal and approve before implementation.

**Estimated Time**: 2-3 hours total
- Database migration: 30 minutes
- Data population: 30 minutes
- Backend changes: 45 minutes
- Frontend changes: 45 minutes
- Testing and deployment: 30 minutes

---

*Proposal prepared by: AI Assistant*  
*Date: 2025-11-09*

