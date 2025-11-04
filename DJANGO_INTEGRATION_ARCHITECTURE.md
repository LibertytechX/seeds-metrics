# Django Database Integration Architecture

## Executive Summary

**Recommended Approach:** **Modified Hybrid Architecture (Option 2+)**

This document provides a comprehensive architectural recommendation for integrating the Seeds Metrics application with the main Django backend database to eliminate ETL synchronization gaps and ensure real-time data consistency.

---

## Table of Contents

1. [Current State Analysis](#current-state-analysis)
2. [Architectural Options Evaluation](#architectural-options-evaluation)
3. [Recommended Architecture](#recommended-architecture)
4. [Schema Mapping Strategy](#schema-mapping-strategy)
5. [Migration Plan](#migration-plan)
6. [Performance Considerations](#performance-considerations)
7. [Risk Assessment & Mitigation](#risk-assessment--mitigation)
8. [Implementation Roadmap](#implementation-roadmap)

---

## 1. Current State Analysis

### Current Architecture

**Seeds Metrics Database (seedsmetrics):**
- **Location:** DigitalOcean Managed PostgreSQL
- **Host:** `private-generaldb-do-user-9489371-0.k.db.ondigitalocean.com:25060`
- **Database:** `seedsmetrics`
- **User:** `metricsuser`
- **Purpose:** Analytics and metrics dashboard

**Main Django Database:**
- **Purpose:** Source of truth for all business data
- **Tables:** `accounts_customuser`, `ajo_ajouser`, `loans_ajoloan`, `loans_ajoloanrepayment`, `loans_ajoloanschedule`

### Current Data Flow

```
Django Database (Source of Truth)
    ↓
ETL Process (Django Management Command)
    ↓
Seeds Metrics Database (Analytics)
    ↓
Go Backend API
    ↓
React Dashboard
```

### Problems with Current Approach

1. **Data Synchronization Gaps:**
   - ETL runs on schedule (not real-time)
   - Potential for missed updates between sync intervals
   - Manual intervention required for failed syncs

2. **Data Duplication:**
   - Same core business data stored in two databases
   - Increased storage costs
   - Potential for data inconsistency

3. **Maintenance Overhead:**
   - ETL code must be maintained
   - Schema changes require updates in both systems
   - Error handling and retry logic complexity

4. **Computed Fields Dependency:**
   - Seeds Metrics database has 14 computed fields calculated via triggers
   - These computations reduce load on main Django server
   - Critical for dashboard performance

---

## 2. Architectural Options Evaluation

### Option 1: Direct Read-Only Connection (Pure Direct Access)

**Description:** Connect Go backend directly to Django database, eliminate seedsmetrics database entirely.

**Pros:**
- ✅ Single source of truth
- ✅ Real-time data (no sync lag)
- ✅ No ETL maintenance
- ✅ Reduced storage costs

**Cons:**
- ❌ **CRITICAL:** Computed fields must be calculated on-the-fly (performance impact)
- ❌ Complex queries burden main Django database
- ❌ No database triggers available (read-only)
- ❌ Dashboard queries could impact production performance
- ❌ Schema differences require extensive query rewriting
- ❌ No local caching or aggregation tables

**Verdict:** ❌ **NOT RECOMMENDED** - Performance and production stability risks outweigh benefits.

---

### Option 2: Hybrid Approach (Original)

**Description:** Keep seedsmetrics for computed fields, read core data from Django database.

**Pros:**
- ✅ Real-time core business data
- ✅ Computed fields remain performant
- ✅ Reduced ETL complexity
- ✅ Dashboard queries don't impact production

**Cons:**
- ❌ Still requires some ETL for initial data population
- ❌ Two database connections to manage
- ❌ Partial data duplication

**Verdict:** ⚠️ **GOOD BUT CAN BE IMPROVED**

---

### Option 3: Modified Hybrid (RECOMMENDED)

**Description:** Read-only connection to Django for core tables + seedsmetrics for computed/aggregated data + minimal ETL for supporting tables.

**Architecture:**
```
┌─────────────────────────────────────────────────────────────┐
│                    Go Backend (Gin)                         │
├─────────────────────────────────────────────────────────────┤
│  ┌──────────────────────┐  ┌──────────────────────────────┐│
│  │ Django DB Connection │  │ SeedsMetrics DB Connection   ││
│  │   (Read-Only)        │  │   (Read-Write)               ││
│  └──────────────────────┘  └──────────────────────────────┘│
└─────────────────────────────────────────────────────────────┘
         │                              │
         ▼                              ▼
┌──────────────────────┐    ┌──────────────────────────────┐
│  Django Database     │    │  SeedsMetrics Database       │
│  (Source of Truth)   │    │  (Analytics & Computed)      │
├──────────────────────┤    ├──────────────────────────────┤
│ READ DIRECTLY:       │    │ KEEP LOCAL:                  │
│ • CustomUser         │    │ • loans (with computed)      │
│ • AjoUser            │    │ • repayments (copy)          │
│ • AjoLoan            │    │ • loan_schedule (copy)       │
│ • AjoLoanRepayment   │    │ • officer_metrics_daily      │
│ • AjoLoanSchedule    │    │ • branch_metrics_daily       │
│                      │    │ • dpd_transitions            │
│                      │    │ • par15_snapshots            │
│                      │    │ • audit_tracking             │
│                      │    │ • team_members               │
└──────────────────────┘    └──────────────────────────────┘
```

**Pros:**
- ✅ Real-time access to source data
- ✅ Computed fields remain performant (triggers in seedsmetrics)
- ✅ Aggregation tables for dashboard performance
- ✅ No impact on production Django database
- ✅ Minimal ETL (only for repayments/schedule sync)
- ✅ Best of both worlds

**Cons:**
- ⚠️ Two database connections (manageable)
- ⚠️ Partial data duplication (repayments, schedule)
- ⚠️ Requires careful schema mapping

**Verdict:** ✅ **RECOMMENDED** - Optimal balance of performance, consistency, and maintainability.

---

## 3. Recommended Architecture

### Core Principle: **Read Source, Compute Local**

**Strategy:**
1. **Read directly from Django DB:** Core business entities (officers, customers, loans)
2. **Sync to seedsmetrics:** Transaction data (repayments, schedules) for trigger-based computation
3. **Compute locally:** All derived metrics, aggregations, and computed fields
4. **Cache locally:** Pre-calculated daily metrics for performance

### Data Source Matrix

| Table | Source | Reason |
|-------|--------|--------|
| **Officers** | Django DB (direct) | Master data, low volume, real-time needed |
| **Customers** | Django DB (direct) | Master data, low volume, real-time needed |
| **Loans (base)** | Django DB (direct) | Core loan data, real-time status needed |
| **Repayments** | SeedsMetrics (synced) | High volume, triggers computed fields |
| **Loan Schedule** | SeedsMetrics (synced) | High volume, triggers DPD calculation |
| **Loans (computed)** | SeedsMetrics (local) | 14 computed fields via triggers |
| **Officer Metrics** | SeedsMetrics (local) | Pre-calculated aggregations |
| **Branch Metrics** | SeedsMetrics (local) | Pre-calculated aggregations |
| **DPD Transitions** | SeedsMetrics (local) | Derived from repayments |
| **PAR15 Snapshots** | SeedsMetrics (local) | Point-in-time snapshots |
| **Audit Tracking** | SeedsMetrics (local) | Dashboard-specific |
| **Team Members** | SeedsMetrics (local) | Dashboard-specific |

---

## 4. Schema Mapping Strategy

### 4.1 Officers Mapping

**Django Table:** `accounts_customuser`  
**SeedsMetrics Table:** `officers` (VIEW or materialized)

```sql
-- Create VIEW in SeedsMetrics that reads from Django DB via Foreign Data Wrapper
CREATE VIEW officers_live AS
SELECT
    id::VARCHAR(50) as officer_id,
    COALESCE(username, email) as officer_name,
    user_phone as officer_phone,
    email as officer_email,
    user_branch as branch,
    'Nigeria' as region,  -- Default or derive from branch mapping
    CASE WHEN performance_status = 'Active' THEN 'Active' ELSE 'Inactive' END as employment_status,
    date_joined::DATE as hire_date,
    NULL::DATE as termination_date,
    'Direct' as primary_channel,  -- Default or derive from user_type
    date_joined as created_at,
    CURRENT_TIMESTAMP as updated_at
FROM django_db.accounts_customuser
WHERE user_type IN ('AGENT', 'STAFF_AGENT', 'PROSPER_AGENT', 'DMO_AGENT', 'AJO_AGENT', 'RECOVERY_AGENT');
```

### 4.2 Customers Mapping

**Django Table:** `ajo_ajouser`  
**SeedsMetrics Table:** `customers` (VIEW)

```sql
CREATE VIEW customers_live AS
SELECT
    id::VARCHAR(50) as customer_id,
    COALESCE(first_name || ' ' || last_name, phone_number) as customer_name,
    phone_number as customer_phone,
    NULL as customer_email,  -- Not in Django schema
    dob as date_of_birth,
    gender,
    state,
    lga,
    address,
    CASE WHEN bvn_verified THEN 'Verified' ELSE 'Pending' END as kyc_status,
    date_created as created_at,
    date_modified as updated_at
FROM django_db.ajo_ajouser
WHERE onboarding_complete = TRUE;
```

### 4.3 Loans Mapping (Base Fields Only)

**Django Table:** `loans_ajoloan`  
**SeedsMetrics Table:** `loans` (HYBRID: base from Django, computed local)

```sql
-- Strategy: Join Django loan data with local computed fields
SELECT
    -- FROM DJANGO (via FDW)
    dl.id::VARCHAR(50) as loan_id,
    dl.borrower_id::VARCHAR(50) as customer_id,
    dl.borrower_full_name as customer_name,
    dl.borrower_phone_number as customer_phone,
    dl.agent_id::VARCHAR(50) as officer_id,
    o.officer_name,
    o.officer_phone,
    o.region,
    o.branch,
    c.state,
    dl.amount_disbursed as loan_amount,
    dl.repayment_amount,
    dl.date_disbursed::DATE as disbursement_date,
    dl.expected_end_date as maturity_date,
    dl.tenor_in_days as loan_term_days,
    dl.interest_rate / 100.0 as interest_rate,  -- Convert percentage to decimal
    dl.processing_fee + dl.nem_fee as fee_amount,
    'Direct' as channel,  -- Derive from loan_type or default
    NULL as channel_partner,
    dl.status,
    dl.date_completed::DATE as closed_date,
    
    -- FROM LOCAL COMPUTED (seedsmetrics.loans table)
    lc.current_dpd,
    lc.max_dpd_ever,
    lc.first_payment_missed,
    lc.first_payment_due_date,
    lc.first_payment_received_date,
    lc.principal_outstanding,
    lc.interest_outstanding,
    lc.fees_outstanding,
    lc.total_outstanding,
    lc.total_principal_paid,
    lc.total_interest_paid,
    lc.total_fees_paid,
    lc.fimr_tagged,
    lc.early_indicator_tagged
    
FROM django_db.loans_ajoloan dl
LEFT JOIN officers_live o ON dl.agent_id = o.officer_id
LEFT JOIN customers_live c ON dl.borrower_id = c.customer_id
LEFT JOIN seedsmetrics.loans_computed lc ON dl.id::VARCHAR(50) = lc.loan_id
WHERE dl.is_disbursed = TRUE;
```

### 4.4 Repayments Mapping (Synced to Local)

**Django Table:** `loans_ajoloanrepayment`  
**SeedsMetrics Table:** `repayments` (COPY via ETL)

**ETL Strategy:** Incremental sync every 5-15 minutes

```python
# Simplified ETL for repayments only
def sync_repayments_incremental():
    last_sync = get_last_sync_timestamp('repayment')
    
    new_repayments = AjoLoanRepayment.objects.filter(
        date_created__gte=last_sync
    ).select_related('ajo_loan')
    
    for repayment in new_repayments:
        upsert_repayment_to_seedsmetrics(repayment)
```

---

## 5. Migration Plan

### Phase 1: Setup Foreign Data Wrapper (Week 1)

**Objective:** Enable SeedsMetrics database to read from Django database

**Steps:**
1. Install `postgres_fdw` extension in SeedsMetrics database
2. Create foreign server connection to Django database
3. Create user mapping with read-only credentials
4. Import foreign schema for core tables
5. Test connectivity and query performance

**SQL Commands:**
```sql
-- On SeedsMetrics database
CREATE EXTENSION IF NOT EXISTS postgres_fdw;

CREATE SERVER django_db
FOREIGN DATA WRAPPER postgres_fdw
OPTIONS (
    host 'django-db-host',
    port '5432',
    dbname 'django_production',
    fetch_size '10000'
);

CREATE USER MAPPING FOR metricsuser
SERVER django_db
OPTIONS (
    user 'readonly_user',
    password 'readonly_password'
);

IMPORT FOREIGN SCHEMA public
LIMIT TO (accounts_customuser, ajo_ajouser, loans_ajoloan)
FROM SERVER django_db
INTO django_db;
```

### Phase 2: Create Views and Mappings (Week 1-2)

**Objective:** Create views that map Django schema to SeedsMetrics schema

**Steps:**
1. Create `officers_live` view
2. Create `customers_live` view
3. Create `loans_base` view
4. Test view performance
5. Add indexes on foreign tables if needed

### Phase 3: Refactor Repository Layer (Week 2-3)

**Objective:** Update Go backend to use hybrid data sources

**Files to Modify:**
- `backend/internal/repository/officer_repository.go`
- `backend/internal/repository/customer_repository.go`
- `backend/internal/repository/loan_repository.go`
- `backend/internal/repository/dashboard_repository.go`

**Example Changes:**
```go
// officer_repository.go - BEFORE
func (r *OfficerRepository) GetAll(ctx context.Context) ([]*models.Officer, error) {
    query := `SELECT * FROM officers`
    // ...
}

// officer_repository.go - AFTER
func (r *OfficerRepository) GetAll(ctx context.Context) ([]*models.Officer, error) {
    query := `SELECT * FROM officers_live`  // Now reads from Django via FDW
    // ...
}
```

### Phase 4: Simplify ETL (Week 3)

**Objective:** Reduce ETL to only sync repayments and schedules

**Steps:**
1. Remove officer sync from ETL
2. Remove customer sync from ETL
3. Remove loan base field sync from ETL
4. Keep only repayment and schedule sync
5. Increase sync frequency to 5 minutes

### Phase 5: Testing & Validation (Week 4)

**Objective:** Ensure data consistency and performance

**Test Cases:**
1. Officer data matches between Django and dashboard
2. Customer data matches between Django and dashboard
3. Loan base fields match between Django and dashboard
4. Computed fields calculate correctly after repayment sync
5. Dashboard performance meets SLA (<2s page load)
6. No production database performance impact

### Phase 6: Production Deployment (Week 5)

**Objective:** Deploy to production with rollback plan

**Steps:**
1. Deploy FDW setup to production SeedsMetrics DB
2. Deploy updated Go backend
3. Monitor performance for 24 hours
4. Gradually reduce ETL frequency
5. Full cutover after validation

---

## 6. Performance Considerations

### 6.1 Foreign Data Wrapper Performance

**Optimization Strategies:**

1. **Fetch Size Tuning:**
   ```sql
   ALTER SERVER django_db OPTIONS (SET fetch_size '10000');
   ```

2. **Selective Column Fetching:**
   ```sql
   -- Only fetch needed columns
   SELECT id, name, branch FROM officers_live
   -- NOT: SELECT * FROM officers_live
   ```

3. **Local Caching:**
   ```sql
   -- Create materialized view for frequently accessed data
   CREATE MATERIALIZED VIEW officers_cached AS
   SELECT * FROM officers_live;
   
   -- Refresh every hour
   REFRESH MATERIALIZED VIEW officers_cached;
   ```

4. **Connection Pooling:**
   - Configure `max_connections` on Django DB
   - Use connection pooling in Go backend
   - Monitor connection count

### 6.2 Query Performance Benchmarks

**Expected Performance:**

| Query Type | Current (ETL) | Hybrid (FDW) | Target |
|------------|---------------|--------------|--------|
| Get All Officers | 50ms | 150ms | <200ms |
| Get Officer by ID | 10ms | 30ms | <50ms |
| Get All Loans | 500ms | 600ms | <1000ms |
| Get Loan by ID | 20ms | 40ms | <100ms |
| Dashboard Load | 1.5s | 2.0s | <3s |

### 6.3 Network Latency Considerations

**Mitigation:**
- Use same cloud provider/region for both databases
- Enable compression on FDW connection
- Implement application-level caching (Redis)
- Use materialized views for heavy queries

---

## 7. Risk Assessment & Mitigation

### Risk 1: Production Database Performance Impact

**Likelihood:** Medium  
**Impact:** High  
**Mitigation:**
- Use read-only credentials (prevents accidental writes)
- Set connection limits on Django DB
- Monitor query performance with pg_stat_statements
- Implement circuit breaker pattern in Go backend
- Use materialized views for heavy aggregations

### Risk 2: Network Connectivity Issues

**Likelihood:** Low  
**Impact:** High  
**Mitigation:**
- Implement retry logic with exponential backoff
- Cache frequently accessed data locally
- Fallback to last known good data
- Alert on connection failures
- Keep minimal ETL as backup

### Risk 3: Schema Drift

**Likelihood:** Medium  
**Impact:** Medium  
**Mitigation:**
- Document schema mapping in code comments
- Automated tests for schema compatibility
- Version control for view definitions
- Change notification process for Django schema changes

### Risk 4: Data Consistency During Migration

**Likelihood:** Low  
**Impact:** Medium  
**Mitigation:**
- Parallel run period (both ETL and FDW active)
- Automated data validation scripts
- Rollback plan with ETL reactivation
- Gradual cutover by table/feature

---

## 8. Implementation Roadmap

### Timeline: 5 Weeks

**Week 1: Setup & Infrastructure**
- [ ] Obtain read-only credentials for Django database
- [ ] Install and configure postgres_fdw
- [ ] Create foreign server and user mapping
- [ ] Import foreign schema
- [ ] Test connectivity

**Week 2: Schema Mapping**
- [ ] Create officers_live view
- [ ] Create customers_live view
- [ ] Create loans_base view
- [ ] Performance testing
- [ ] Documentation

**Week 3: Backend Refactoring**
- [ ] Update officer_repository.go
- [ ] Update customer_repository.go
- [ ] Update loan_repository.go
- [ ] Update dashboard_repository.go
- [ ] Unit tests

**Week 4: ETL Simplification & Testing**
- [ ] Simplify ETL to repayments/schedules only
- [ ] Integration testing
- [ ] Performance testing
- [ ] Data validation
- [ ] Load testing

**Week 5: Production Deployment**
- [ ] Deploy to staging
- [ ] Staging validation
- [ ] Deploy to production
- [ ] Monitor for 48 hours
- [ ] Full cutover
- [ ] Decommission old ETL

---

## 9. Code Changes Required

### 9.1 Configuration Changes

**File:** `backend/internal/config/config.go`

```go
type Config struct {
    Server   ServerConfig
    Database DatabaseConfig
    DjangoDatabase DatabaseConfig  // NEW: Django DB connection
    Redis    RedisConfig
    // ...
}

type DatabaseConfig struct {
    Host               string
    Port               string
    User               string
    Password           string
    DBName             string
    SSLMode            string
    MaxConnections     int
    MaxIdleConnections int
    ConnMaxLifetime    time.Duration
    ReadOnly           bool  // NEW: Flag for read-only connections
}
```

**File:** `backend/.env.production`

```bash
# Existing SeedsMetrics DB
DB_HOST=generaldb-do-user-9489371-0.k.db.ondigitalocean.com
DB_PORT=25060
DB_USER=metricsuser
DB_PASSWORD=<password>
DB_NAME=seedsmetrics

# NEW: Django DB (Read-Only)
DJANGO_DB_HOST=<django-db-host>
DJANGO_DB_PORT=5432
DJANGO_DB_USER=readonly_user
DJANGO_DB_PASSWORD=<readonly-password>
DJANGO_DB_NAME=django_production
DJANGO_DB_SSLMODE=require
DJANGO_DB_MAX_CONNECTIONS=10
```

### 9.2 Repository Layer Changes

**File:** `backend/internal/repository/officer_repository.go`

```go
// Change FROM:
query := `SELECT * FROM officers WHERE officer_id = $1`

// Change TO:
query := `SELECT * FROM officers_live WHERE officer_id = $1`
```

**File:** `backend/internal/repository/loan_repository.go`

```go
// For loan listing (base fields from Django, computed from local)
query := `
    SELECT
        l.loan_id,
        l.customer_name,
        l.officer_name,
        l.loan_amount,
        l.disbursement_date,
        l.status,
        -- Computed fields from local table
        lc.current_dpd,
        lc.total_outstanding,
        lc.fimr_tagged
    FROM loans_base l  -- View to Django
    LEFT JOIN loans_computed lc ON l.loan_id = lc.loan_id  -- Local computed
    WHERE l.status = 'Active'
`
```

---

## 10. Success Metrics

### Key Performance Indicators

1. **Data Freshness:**
   - Target: <5 minutes lag for repayments
   - Target: Real-time for officers/customers/loans

2. **Dashboard Performance:**
   - Target: <3s page load time
   - Target: <500ms API response time

3. **System Reliability:**
   - Target: 99.9% uptime
   - Target: <0.1% data sync errors

4. **Production Impact:**
   - Target: <5% increase in Django DB load
   - Target: Zero production incidents

---

## 11. Rollback Plan

### If Issues Arise

1. **Immediate Rollback (< 1 hour):**
   - Revert Go backend to previous version
   - Reactivate full ETL process
   - Switch DNS/load balancer if needed

2. **Data Validation:**
   - Compare data between Django and SeedsMetrics
   - Identify and fix discrepancies
   - Re-sync if necessary

3. **Root Cause Analysis:**
   - Review logs and metrics
   - Identify performance bottlenecks
   - Document lessons learned

---

## 12. Conclusion

The **Modified Hybrid Architecture** provides the optimal balance between:
- ✅ Real-time data consistency
- ✅ High performance (computed fields via triggers)
- ✅ Production stability (no impact on Django DB)
- ✅ Maintainability (reduced ETL complexity)

**Next Steps:**
1. Obtain read-only credentials for Django database
2. Begin Phase 1 implementation
3. Schedule weekly progress reviews

---

**Document Version:** 1.0  
**Last Updated:** 2025-11-04  
**Author:** Seeds Metrics Team  
**Status:** Pending Approval

