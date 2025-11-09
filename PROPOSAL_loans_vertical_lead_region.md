# Proposal: Add Vertical Lead and Update Region in Loans Table

## Executive Summary

Add vertical lead information to the `loans` table and update the existing `region` column to match officer regions from `verticals.tsv`. This will enable filtering and analysis of loans by vertical leadership structure.

## Current State Analysis

### Loans Table
- **Total Loans**: 17,929
- **Unique Officers**: 693
- **Current Region Values**: All loans show "Nigeria" (generic)
- **Missing Fields**: `vertical_lead_name`, `vertical_lead_email`

### Officers Table (Already Updated)
- **Officers with Vertical Lead Data**: 548
- **Officers in Loans Table with Vertical Lead**: 431
- **Loans that will get Vertical Lead Data**: 14,457 (80.6%)

### Region Mismatch
- **All loans** currently have `region = "Nigeria"`
- **Officers** have specific regions (Ruby, Sapphire, Garnet, etc.)
- **12,521 loans** have mismatched regions (need update)

## Proposed Changes

### 1. Database Migration

**File**: `migrations/032_add_vertical_lead_to_loans.sql`

```sql
-- Add vertical lead columns to loans table
ALTER TABLE loans 
ADD COLUMN IF NOT EXISTS vertical_lead_name VARCHAR(255),
ADD COLUMN IF NOT EXISTS vertical_lead_email VARCHAR(255);

-- Create indexes for query performance
CREATE INDEX IF NOT EXISTS idx_loans_vertical_lead_email 
ON loans(vertical_lead_email);

CREATE INDEX IF NOT EXISTS idx_loans_vertical_lead_name 
ON loans(vertical_lead_name);

-- Add comment
COMMENT ON COLUMN loans.vertical_lead_name IS 'Name of the vertical lead from the officer who disbursed this loan';
COMMENT ON COLUMN loans.vertical_lead_email IS 'Email of the vertical lead from the officer who disbursed this loan';
```

### 2. Data Population

**Script**: `update_loans_vertical_lead_and_region.sh`

**Logic**:
```sql
-- Update vertical lead information from officers table
UPDATE loans l
SET 
    vertical_lead_name = o.vertical_lead_name,
    vertical_lead_email = o.vertical_lead_email,
    region = o.region
FROM officers o
WHERE l.officer_id = o.officer_id;
```

**Expected Results**:
- **14,457 loans** will get vertical lead data (80.6%)
- **3,472 loans** will remain NULL (officers without vertical lead data)
- **12,521 loans** will get updated regions (from "Nigeria" to specific regions)
- **5,408 loans** will keep "Nigeria" (officers still have "Nigeria" as region)

### 3. Backend Model Updates

**File**: `backend/internal/models/loan.go`

Add fields to `Loan` struct:
```go
type Loan struct {
    // ... existing fields ...
    Region              string   `json:"region"`
    VerticalLeadName    *string  `json:"vertical_lead_name,omitempty"`
    VerticalLeadEmail   *string  `json:"vertical_lead_email,omitempty"`
    // ... rest of fields ...
}
```

### 4. Backend Repository Updates

**File**: `backend/internal/repository/loan_repository.go`

Update SQL queries in:
- `GetLoans()` - Add vertical lead columns to SELECT
- `GetLoanByID()` - Add vertical lead columns to SELECT
- Add NULL handling with `sql.NullString`

### 5. Frontend Updates

**File**: `metrics-dashboard/src/components/AllLoans.jsx`

**Changes**:
1. Add 2 new table columns:
   - Vertical Lead Name
   - Vertical Lead Email

2. Add filter dropdowns:
   - Filter by Vertical Lead Email
   - Filter by Region (already exists, will now have more options)

3. Update CSV export to include new columns

4. Transform API response (snake_case → camelCase):
   - `vertical_lead_name` → `verticalLeadName`
   - `vertical_lead_email` → `verticalLeadEmail`

## Implementation Plan

### Phase 1: Database Changes (15 min)
1. ✅ Create migration file
2. ✅ Execute migration on production database
3. ✅ Verify columns added

### Phase 2: Data Population (20 min)
1. ✅ Create update script
2. ✅ Test on sample data
3. ✅ Execute full update
4. ✅ Verify data populated correctly

### Phase 3: Backend Changes (30 min)
1. ✅ Update Loan model
2. ✅ Update GetLoans repository method
3. ✅ Update GetLoanByID repository method
4. ✅ Build and test locally
5. ✅ Deploy to production

### Phase 4: Frontend Changes (30 min)
1. ✅ Update AllLoans component
2. ✅ Add table columns
3. ✅ Add filters
4. ✅ Update CSV export
5. ✅ Update API transformation
6. ✅ Build and deploy

### Phase 5: Testing & Verification (15 min)
1. ✅ Test API endpoints
2. ✅ Test frontend display
3. ✅ Test filtering
4. ✅ Test CSV export
5. ✅ Verify data accuracy

**Total Estimated Time**: ~2 hours

## Expected Coverage

### Vertical Lead Data
- **14,457 loans** (80.6%) will have vertical lead information
- **3,472 loans** (19.4%) will show NULL/"N/A"

### Region Data
- **12,521 loans** (69.8%) will have specific regions (Ruby, Sapphire, etc.)
- **5,408 loans** (30.2%) will remain "Nigeria"

### Breakdown by Region (After Update)
Based on officer distribution:
- Ruby: ~2,455 loans
- Emerald: ~2,270 loans
- Sapphire: ~2,239 loans
- Garnet: ~1,864 loans
- Opal: ~1,609 loans
- Hilander: ~1,099 loans
- Key Accounts 1: ~variable
- Others: ~variable
- Nigeria: ~5,408 loans

## Benefits

1. **Enhanced Filtering**: Users can filter loans by vertical lead
2. **Better Reporting**: Vertical lead performance analysis
3. **Data Consistency**: Loan regions match officer regions
4. **Hierarchical View**: See loans by vertical leadership structure
5. **Improved Analytics**: Track loan performance by vertical

## Risks & Mitigation

### Risk 1: Data Inconsistency
- **Risk**: Some loans may have outdated officer assignments
- **Mitigation**: This is a point-in-time snapshot; future syncs will update

### Risk 2: NULL Values
- **Risk**: 19.4% of loans won't have vertical lead data
- **Mitigation**: Display "N/A" in UI; this is expected for officers not in verticals.tsv

### Risk 3: Performance Impact
- **Risk**: Additional columns may slow queries
- **Mitigation**: Added indexes on new columns

## Rollback Plan

If issues arise:
1. Remove columns: `ALTER TABLE loans DROP COLUMN vertical_lead_name, DROP COLUMN vertical_lead_email;`
2. Revert backend code changes
3. Revert frontend code changes
4. Restart services

## Success Criteria

✅ Migration executes without errors  
✅ 14,457 loans populated with vertical lead data  
✅ 12,521 loans updated with specific regions  
✅ API returns new fields correctly  
✅ Frontend displays new columns  
✅ Filtering works correctly  
✅ CSV export includes new columns  
✅ No performance degradation  

## Next Steps

1. Get approval to proceed
2. Execute Phase 1 (Database Migration)
3. Execute Phase 2 (Data Population)
4. Execute Phase 3 (Backend Changes)
5. Execute Phase 4 (Frontend Changes)
6. Execute Phase 5 (Testing & Verification)

## Questions?

- Should we also add `supervisor_name` and `supervisor_email` to loans table?
- Should we create a scheduled job to keep this data in sync?
- Should we add vertical lead to other tables (e.g., repayments)?

