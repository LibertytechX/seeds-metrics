# Alternative Django Integration Architecture
## (Without Foreign Data Wrapper)

## Problem

The original plan to use PostgreSQL Foreign Data Wrapper (FDW) cannot be implemented because:
1. DigitalOcean Managed PostgreSQL restricts FDW creation to superuser/doadmin roles
2. We only have `seedsuser` credentials which lack the necessary permissions
3. Granting FDW permissions requires database admin access

## Alternative Solution: Dual Database Connection in Go Backend

Instead of using FDW at the database level, we'll implement dual database connections in the Go backend application layer.

---

## Architecture Overview

```
┌─────────────────────────────────────────────────────────────┐
│                    Go Backend (Gin)                         │
├─────────────────────────────────────────────────────────────┤
│  ┌──────────────────────┐  ┌──────────────────────────────┐│
│  │ Django DB Connection │  │ SeedsMetrics DB Connection   ││
│  │   (Read-Only)        │  │   (Read-Write)               ││
│  │   Direct SQL Queries │  │   Computed Fields & Metrics  ││
│  └──────────────────────┘  └──────────────────────────────┘│
└─────────────────────────────────────────────────────────────┘
         │                              │
         ▼                              ▼
┌──────────────────────┐    ┌──────────────────────────────┐
│  Django Database     │    │  SeedsMetrics Database       │
│  164.90.155.2:5432   │    │  (DigitalOcean Managed)      │
│  Database: savings   │    │  Database: seedsmetrics      │
├──────────────────────┤    ├──────────────────────────────┤
│ READ DIRECTLY:       │    │ KEEP LOCAL:                  │
│ • accounts_customuser│    │ • loans (with computed)      │
│ • ajo_ajouser        │    │ • repayments (synced)        │
│ • loans_ajoloan      │    │ • loan_schedule (synced)     │
│ • loans_ajoloanrepay │    │ • officer_metrics_daily      │
│ • loans_ajoloanschedu│    │ • branch_metrics_daily       │
│                      │    │ • dpd_transitions            │
│                      │    │ • par15_snapshots            │
└──────────────────────┘    └──────────────────────────────┘
```

---

## Implementation Strategy

### Phase 1: Add Django Database Connection to Go Backend

**File:** `backend/internal/config/config.go`

```go
type Config struct {
    Server         ServerConfig
    Database       DatabaseConfig  // SeedsMetrics DB
    DjangoDatabase DatabaseConfig  // NEW: Django DB (read-only)
    Redis          RedisConfig
}
```

**File:** `backend/.env`

```bash
# Existing SeedsMetrics DB
DB_HOST=generaldb-do-user-9489371-0.k.db.ondigitalocean.com
DB_PORT=25060
DB_USER=seedsuser
DB_PASSWORD=@seedsuser2020
DB_NAME=seedsmetrics
DB_SSLMODE=require

# NEW: Django DB (Read-Only)
DJANGO_DB_HOST=164.90.155.2
DJANGO_DB_PORT=5432
DJANGO_DB_USER=metricsuser
DJANGO_DB_PASSWORD=EiRXo6IfeHQuM3wcbZ67$LzwmVKCXhpUhWg
DJANGO_DB_NAME=savings
DJANGO_DB_SSLMODE=require
DJANGO_DB_MAX_CONNECTIONS=10
```

### Phase 2: Create Django Repository Layer

**File:** `backend/internal/repository/django_repository.go`

```go
package repository

import (
    "context"
    "database/sql"
)

type DjangoRepository struct {
    db *sql.DB
}

func NewDjangoRepository(db *sql.DB) *DjangoRepository {
    return &DjangoRepository{db: db}
}

// GetOfficers reads officers directly from Django database
func (r *DjangoRepository) GetOfficers(ctx context.Context) ([]*models.Officer, error) {
    query := `
        SELECT
            id::VARCHAR(50) as officer_id,
            COALESCE(username, email) as officer_name,
            user_phone as officer_phone,
            email as officer_email,
            user_branch as branch,
            CASE
                WHEN user_branch LIKE '%Lagos%' THEN 'Lagos'
                WHEN user_branch LIKE '%Abuja%' THEN 'FCT'
                ELSE 'Nigeria'
            END as region,
            CASE
                WHEN performance_status = 'Active' THEN 'Active'
                ELSE 'Inactive'
            END as employment_status,
            date_joined::DATE as hire_date,
            CASE
                WHEN user_type IN ('PROSPER_AGENT', 'DMO_AGENT') THEN 'Partner'
                ELSE 'Direct'
            END as primary_channel
        FROM accounts_customuser
        WHERE user_type IN ('AGENT', 'STAFF_AGENT', 'PROSPER_AGENT', 'DMO_AGENT', 'AJO_AGENT', 'RECOVERY_AGENT')
        AND is_active = TRUE
    `
    
    rows, err := r.db.QueryContext(ctx, query)
    if err != nil {
        return nil, err
    }
    defer rows.Close()
    
    var officers []*models.Officer
    for rows.Next() {
        var officer models.Officer
        err := rows.Scan(
            &officer.OfficerID,
            &officer.OfficerName,
            &officer.OfficerPhone,
            &officer.OfficerEmail,
            &officer.Branch,
            &officer.Region,
            &officer.EmploymentStatus,
            &officer.HireDate,
            &officer.PrimaryChannel,
        )
        if err != nil {
            return nil, err
        }
        officers = append(officers, &officer)
    }
    
    return officers, nil
}

// GetCustomers reads customers directly from Django database
func (r *DjangoRepository) GetCustomers(ctx context.Context) ([]*models.Customer, error) {
    query := `
        SELECT
            id::VARCHAR(50) as customer_id,
            COALESCE(TRIM(first_name || ' ' || last_name), phone_number) as customer_name,
            phone_number as customer_phone,
            dob as date_of_birth,
            gender,
            state,
            lga,
            address,
            CASE
                WHEN bvn_verified = TRUE AND onboarding_verified = TRUE THEN 'Verified'
                WHEN bvn_verified = TRUE THEN 'Partial'
                ELSE 'Pending'
            END as kyc_status
        FROM ajo_ajouser
        WHERE onboarding_complete = TRUE
    `
    
    // Similar implementation...
    return customers, nil
}

// GetLoansBase reads loan base fields directly from Django database
func (r *DjangoRepository) GetLoansBase(ctx context.Context) ([]*models.LoanBase, error) {
    query := `
        SELECT
            id::VARCHAR(50) as loan_id,
            borrower_id::VARCHAR(50) as customer_id,
            borrower_full_name as customer_name,
            agent_id::VARCHAR(50) as officer_id,
            amount_disbursed as loan_amount,
            repayment_amount,
            date_disbursed::DATE as disbursement_date,
            expected_end_date as maturity_date,
            tenor_in_days as loan_term_days,
            (interest_rate / 100.0)::DECIMAL(5,4) as interest_rate,
            (COALESCE(processing_fee, 0) + COALESCE(nem_fee, 0))::DECIMAL(15,2) as fee_amount,
            status,
            date_completed::DATE as closed_date
        FROM loans_ajoloan
        WHERE is_disbursed = TRUE
    `
    
    // Similar implementation...
    return loans, nil
}
```

### Phase 3: Update Main Application to Use Dual Connections

**File:** `backend/cmd/api/main.go`

```go
func main() {
    cfg := config.Load()
    
    // Connect to SeedsMetrics database
    seedsDB, err := database.NewPostgresDB(&cfg.Database)
    if err != nil {
        log.Fatal("Failed to connect to SeedsMetrics database:", err)
    }
    defer seedsDB.Close()
    
    // NEW: Connect to Django database (read-only)
    djangoDB, err := database.NewPostgresDB(&cfg.DjangoDatabase)
    if err != nil {
        log.Fatal("Failed to connect to Django database:", err)
    }
    defer djangoDB.Close()
    
    // Create repositories
    djangoRepo := repository.NewDjangoRepository(djangoDB.DB)
    loanRepo := repository.NewLoanRepository(seedsDB.DB)
    repaymentRepo := repository.NewRepaymentRepository(seedsDB.DB)
    
    // Create services with both repositories
    dashboardService := service.NewDashboardService(djangoRepo, loanRepo, repaymentRepo)
    
    // ... rest of setup
}
```

### Phase 4: Modify Dashboard Service to Combine Data

**File:** `backend/internal/service/dashboard_service.go`

```go
type DashboardService struct {
    djangoRepo    *repository.DjangoRepository
    loanRepo      *repository.LoanRepository
    repaymentRepo *repository.RepaymentRepository
}

func (s *DashboardService) GetOfficers(ctx context.Context) ([]*models.Officer, error) {
    // Read directly from Django database (real-time)
    return s.djangoRepo.GetOfficers(ctx)
}

func (s *DashboardService) GetLoansWithMetrics(ctx context.Context) ([]*models.LoanWithMetrics, error) {
    // Get base loan data from Django (real-time)
    loansBase, err := s.djangoRepo.GetLoansBase(ctx)
    if err != nil {
        return nil, err
    }
    
    // Get computed fields from SeedsMetrics database
    loansComputed, err := s.loanRepo.GetComputedFields(ctx)
    if err != nil {
        return nil, err
    }
    
    // Merge base data with computed fields
    return mergeLoansData(loansBase, loansComputed), nil
}
```

---

## Advantages of This Approach

1. ✅ **No Database Permissions Required** - Works with existing user permissions
2. ✅ **Real-Time Data** - Officers, customers, and loan base data read directly from Django
3. ✅ **Computed Fields Preserved** - Triggers and computed fields remain in SeedsMetrics
4. ✅ **Flexible** - Can easily switch between Django and local data sources
5. ✅ **Testable** - Can mock Django repository for testing
6. ✅ **Gradual Migration** - Can migrate table by table

---

## Disadvantages

1. ⚠️ **Application-Level Joins** - Joining data across databases happens in Go code
2. ⚠️ **Two Connection Pools** - Need to manage two database connections
3. ⚠️ **More Complex Code** - Repository layer becomes more complex

---

## Migration Timeline

### Week 1: Setup Dual Connections
- Add Django database configuration
- Create Django repository layer
- Test connectivity and basic queries

### Week 2: Migrate Officers & Customers
- Update officer endpoints to read from Django
- Update customer endpoints to read from Django
- Test and validate data consistency

### Week 3: Migrate Loans Base Data
- Update loan endpoints to combine Django base + SeedsMetrics computed
- Test performance and data consistency
- Optimize queries

### Week 4: Simplify ETL
- Reduce ETL to only sync repayments and schedules
- Increase sync frequency to 5 minutes
- Remove officer/customer/loan base sync

### Week 5: Production Deployment
- Deploy to staging
- Performance testing
- Deploy to production
- Monitor and optimize

---

## Next Steps

1. **Approve this alternative approach**
2. **Implement Phase 1** - Add Django database connection to Go backend
3. **Test connectivity** - Verify we can query Django database from Go
4. **Proceed with implementation** - Follow the migration timeline

---

**This approach achieves the same goals as FDW but at the application layer instead of database layer.**

