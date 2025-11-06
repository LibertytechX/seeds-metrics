# Credit Health Overview - Filter Enhancement Analysis

## Overview

This document analyzes the filter parameters available in the Agent Performance endpoint (GetOfficers) and identifies which filters need to be added to the Credit Health Overview endpoint (GetBranches).

## Current Filters in Agent Performance (GetOfficers)

**Handler**: `GetOfficers()` in `backend/internal/handlers/dashboard_handler.go` (lines 117-193)

### Query Parameters Supported:
1. **branch** - Filter by branch name
2. **region** - Filter by region name
3. **channel** - Filter by channel
4. **wave** - Filter by wave (cohort)
5. **user_type** - Filter by user type
6. **sort_by** - Sort field (e.g., risk_score, total_portfolio)
7. **sort_dir** - Sort direction (asc/desc)
8. **page** - Page number (default: 1)
9. **limit** - Items per page (default: 50)

### Code Reference:
```go
// Lines 137-157 in dashboard_handler.go
if branch := c.Query("branch"); branch != "" {
    filters["branch"] = branch
}
if region := c.Query("region"); region != "" {
    filters["region"] = region
}
if channel := c.Query("channel"); channel != "" {
    filters["channel"] = channel
}
if wave := c.Query("wave"); wave != "" {
    filters["wave"] = wave
}
if userType := c.Query("user_type"); userType != "" {
    filters["user_type"] = userType
}
if sortBy := c.Query("sort_by"); sortBy != "" {
    filters["sort_by"] = sortBy
}
if sortDir := c.Query("sort_dir"); sortDir != "" {
    filters["sort_dir"] = sortDir
}
```

## Current Filters in Credit Health Overview (GetBranches)

**Handler**: `GetBranches()` in `backend/internal/handlers/dashboard_handler.go` (lines 530-593)

### Query Parameters Currently Supported:
1. **region** - Filter by region name
2. **wave** - Filter by wave (cohort)
3. **sort_by** - Sort field
4. **sort_dir** - Sort direction (asc/desc)

### Code Reference:
```go
// Lines 546-557 in dashboard_handler.go
if region := c.Query("region"); region != "" {
    filters["region"] = region
}
if wave := c.Query("wave"); wave != "" {
    filters["wave"] = wave
}
if sortBy := c.Query("sort_by"); sortBy != "" {
    filters["sort_by"] = sortBy
}
if sortDir := c.Query("sort_dir"); sortDir != "" {
    filters["sort_dir"] = sortDir
}
```

## Missing Filters in Credit Health Overview

The following filters from Agent Performance are **NOT** currently in Credit Health Overview:

1. **branch** - Filter by branch name
2. **channel** - Filter by channel
3. **user_type** - Filter by user type

## Repository Layer Analysis

### GetOfficers Repository Method
**Location**: `backend/internal/repository/dashboard_repository.go` - `GetOfficers()` method

Applies filters:
- branch
- region
- channel
- wave
- user_type
- sort_by
- sort_dir
- page
- limit

### GetBranches Repository Method
**Location**: `backend/internal/repository/dashboard_repository.go` - `GetBranches()` method (lines 1068-1146)

Current query structure:
```sql
SELECT
    l.branch,
    l.region,
    COALESCE(SUM(l.principal_outstanding), 0) as portfolio_total,
    COALESCE(SUM(CASE WHEN l.current_dpd >= 15 THEN l.principal_outstanding ELSE 0 END), 0) as overdue_15d,
    ...
FROM loans l
WHERE 1=1
```

Currently applies filters:
- region
- wave
- sort_by
- sort_dir

## Implementation Plan

### Step 1: Update GetBranches Handler
Add filter parsing for:
- branch
- channel
- user_type

### Step 2: Update GetBranches Repository Query
Modify the SQL query to:
- Filter by branch (if provided)
- Filter by channel (if provided)
- Filter by user_type (if provided)

### Step 3: Ensure Consistency
- Use same parameter names as GetOfficers
- Use same filter logic and WHERE clause construction
- Maintain same sorting behavior

## Expected Outcome

After implementation, the Credit Health Overview endpoint will support:

```
GET /api/v1/branches?region=Lagos&branch=Lekki&channel=AGENT&wave=Wave1&user_type=AGENT&sort_by=portfolio_total&sort_dir=desc
```

This will provide feature parity with the Agent Performance endpoint while maintaining the branch-level aggregation.

