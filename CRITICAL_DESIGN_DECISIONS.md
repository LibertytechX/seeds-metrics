# Critical Design Decisions

## 🎯 Overview

This document highlights the most important architectural decisions made for the Metrics Dashboard Backend. These decisions have significant implications for implementation and must be understood by all team members.

---

## ⚠️ DECISION #1: Computed Fields from Repayments

### **The Decision**

**All derived loan fields are computed from repayments data, NOT sent from the main business backend.**

### **What This Means**

The `loans` table has **two types of fields**:

#### **Type 1: FROM ETL SOURCE (20 fields)**
These come directly from the main backend:
- Loan identifiers (loan_id, customer_id, officer_id)
- Loan details (loan_amount, disbursement_date, maturity_date, interest_rate, fee_amount)
- Geographic info (region, branch, state)
- Channel info (channel, channel_partner)
- Status (status, closed_date)

#### **Type 2: [COMPUTED] (14 fields)**
These are calculated by the analytics service from repayments:
- **DPD tracking**: `current_dpd`, `max_dpd_ever`
- **First payment**: `first_payment_missed`, `first_payment_due_date`, `first_payment_received_date`
- **Outstanding balances**: `principal_outstanding`, `interest_outstanding`, `fees_outstanding`, `total_outstanding`
- **Collections**: `total_principal_paid`, `total_interest_paid`, `total_fees_paid`
- **Risk indicators**: `fimr_tagged`, `early_indicator_tagged`

### **Why This Decision?**

✅ **Reduces load on main business server** - No complex aggregations needed  
✅ **Centralized calculation logic** - All metrics in one place  
✅ **Real-time updates** - Triggers automatically update fields  
✅ **Audit trail** - Raw data preserved for recalculation  
✅ **Performance** - Computed fields stored and indexed  

### **How It Works**

```
Main Backend sends:
  - loan_id: "LN001"
  - loan_amount: 500000
  - disbursement_date: "2024-01-15"
  - ... (17 more ETL fields)
  - [NO DPD, NO outstanding, NO totals]

↓ ETL inserts into loans table

Main Backend sends repayments:
  - repayment_id: "REP001"
  - loan_id: "LN001"
  - principal_paid: 80000
  - interest_paid: 15000
  - fees_paid: 5000

↓ ETL inserts into repayments table

↓ Trigger fires automatically

Loans table updated:
  - total_principal_paid: 80000
  - principal_outstanding: 420000
  - total_outstanding: 445000
  - current_dpd: 0
  - fimr_tagged: FALSE
  - ... (9 more computed fields)
```

### **Implementation**

**Database Trigger:**
```sql
CREATE TRIGGER trg_update_loan_after_repayment
AFTER INSERT OR UPDATE ON repayments
FOR EACH ROW
EXECUTE FUNCTION update_loan_computed_fields();
```

**See:** `ETL_DATA_FLOW_SPECIFICATION.md` for complete details.

---

## 🏗️ DECISION #2: Separate Microservice Architecture

### **The Decision**

**Build analytics as a separate microservice, not part of the main backend.**

### **Architecture**

```
Main Business Backend (Transactional)
    ↓ (Event Stream / ETL)
Analytics Microservice (Read-Only)
    ↓ (REST API)
Metrics Dashboard (Frontend)
```

### **Why This Decision?**

✅ **Performance isolation** - Heavy analytics won't slow down core business  
✅ **Independent scaling** - Scale analytics separately from main backend  
✅ **Technology flexibility** - Use Go for analytics, Node.js for main backend  
✅ **Deployment independence** - Deploy analytics without touching main backend  

### **Trade-offs**

❌ **Additional infrastructure** - Need separate servers/containers  
❌ **Data synchronization** - Need ETL process (15-30 min delay)  
❌ **Complexity** - Two codebases to maintain  

**Verdict:** Benefits outweigh costs for this use case.

---

## 🔧 DECISION #3: Technology Stack - Go (Golang)

### **The Decision**

**Use Go (Golang) as the primary language for the analytics backend.**

### **Why Go?**

✅ **10-100x faster** than Node.js for CPU-intensive calculations  
✅ **Built-in concurrency** - Goroutines for parallel metric calculations  
✅ **Low memory footprint** - Efficient for large datasets  
✅ **Single binary deployment** - Easy to deploy  
✅ **Excellent PostgreSQL support** - `pgx` driver is production-ready  

### **Alternative: Node.js (TypeScript)**

**When to use Node.js instead:**
- Rapid prototyping needed
- Team already knows JavaScript/TypeScript
- Dataset is small (< 100K loans)
- Time-to-market is critical

### **Comparison**

| Aspect | Go | Node.js |
|--------|----|----|
| **Performance** | ⭐⭐⭐⭐⭐ | ⭐⭐⭐ |
| **Concurrency** | ⭐⭐⭐⭐⭐ (goroutines) | ⭐⭐⭐ (async/await) |
| **Development Speed** | ⭐⭐⭐ | ⭐⭐⭐⭐⭐ |
| **Team Familiarity** | ⭐⭐ | ⭐⭐⭐⭐⭐ |
| **Ecosystem** | ⭐⭐⭐⭐ | ⭐⭐⭐⭐⭐ |
| **Memory Usage** | ⭐⭐⭐⭐⭐ | ⭐⭐⭐ |

**Recommendation:** Go for production, Node.js for prototyping.

---

## 📊 DECISION #4: Pre-Aggregation Strategy

### **The Decision**

**Pre-calculate and store officer/branch metrics, compute drilldowns on-demand.**

### **What Gets Pre-Aggregated?**

**Stored in Database:**
- `officer_metrics_daily` - All officer metrics (FIMR, AYR, DQI, Risk Score, etc.)
- `branch_metrics_daily` - All branch metrics
- **Frequency:** Every 15-30 minutes

**Computed On-Demand:**
- Loan-level drilldowns (FIMR drilldown, Early Indicators drilldown)
- Filtered queries (by date range, region, branch, officer)
- **Frequency:** When user requests

### **Why This Decision?**

✅ **Fast API responses** - Pre-aggregated data returns in < 200ms  
✅ **Reduced database load** - Don't recalculate metrics on every request  
✅ **Flexible drilldowns** - Users can filter loan-level data dynamically  
✅ **Historical tracking** - Daily snapshots for trend analysis  

### **Trade-offs**

❌ **Storage overhead** - Need tables for daily metrics  
❌ **Data freshness** - 15-30 minute delay  
❌ **Complexity** - Need background workers  

**Verdict:** Essential for performance at scale.

---

## 💾 DECISION #5: Caching Strategy

### **The Decision**

**Use Redis with 15-minute TTL for frequently accessed metrics.**

### **What Gets Cached?**

- Officer metrics (all fields)
- Branch metrics (all fields)
- Filter options (regions, branches, officers)

### **What Doesn't Get Cached?**

- Loan-level drilldowns (too many combinations)
- Real-time audit updates

### **Cache Invalidation**

**Time-based:**
- TTL: 15 minutes (aligned with batch processing)

**Event-based:**
- Invalidate officer cache when audit status changes
- Invalidate branch cache when officer reassigned

### **Why This Decision?**

✅ **Sub-millisecond responses** - Redis is extremely fast  
✅ **Reduced database load** - 80%+ cache hit rate target  
✅ **Simple invalidation** - TTL handles most cases  

---

## 🔄 DECISION #6: Data Synchronization - Hybrid Approach

### **The Decision**

**Use batch processing (primary) + event-driven updates (secondary).**

### **Batch Processing (Every 15-30 minutes)**

**What:**
- Sync all loans, repayments, schedules from main backend
- Calculate all officer/branch metrics
- Update cache

**Why:**
- Reliable and predictable
- Handles bulk updates efficiently
- Simple to implement

### **Event-Driven (Real-time)**

**What:**
- Critical events only (loan disbursed, loan closed, audit status changed)
- Trigger immediate cache invalidation
- Optional: Trigger immediate recalculation

**Why:**
- Faster updates for critical events
- Better user experience
- Complements batch processing

### **On-Demand (User-triggered)**

**What:**
- Loan-level drilldowns with filters
- Calculated when user requests

**Why:**
- Too many combinations to pre-calculate
- Users need flexible filtering

---

## 🗄️ DECISION #7: Database Design - Supporting Tables

### **The Decision**

**Create supporting tables for accurate metric calculations.**

### **Supporting Tables**

1. **`loan_schedule`** - Required for D0-6 Slippage calculation
2. **`dpd_transitions`** - Required for Roll calculation
3. **`par15_snapshots`** - Required for AYR calculation (mid-month PAR15)

### **Why These Tables?**

**Without `loan_schedule`:**
- Can't calculate `amount_due_7d` (needed for D0-6 Slippage)
- Can't track which installments are overdue

**Without `dpd_transitions`:**
- Can't calculate Roll (movement from DPD 1-6 to 7-30)
- Can't track DPD bucket changes over time

**Without `par15_snapshots`:**
- Can't calculate AYR (needs PAR15 at mid-month, not current)
- Can't track historical PAR15 trends

### **Trade-offs**

❌ **More tables to maintain** - 12 total instead of 4  
❌ **More ETL complexity** - Need to sync schedules  
❌ **More storage** - Especially `loan_schedule` (10M-100M rows)  

**Verdict:** Essential for accurate metric calculations.

---

## 📈 DECISION #8: Performance Targets

### **The Targets**

| Metric | Target | Strategy |
|--------|--------|----------|
| **API Response (p95)** | < 200ms | Pre-aggregation + Redis |
| **Drilldown (p95)** | < 1s | Optimized indexes |
| **Cache Hit Rate** | > 80% | 15-min TTL |
| **Calculation Time** | < 5 min | Parallel processing |
| **Data Freshness** | < 30 min | Batch every 15-30 min |
| **Concurrent Users** | 100+ | Horizontal scaling |

### **Why These Targets?**

- **< 200ms API**: Industry standard for good UX
- **< 1s drilldown**: Acceptable for filtered queries
- **> 80% cache hit**: Reduces database load significantly
- **< 5 min calculation**: Ensures timely updates
- **< 30 min freshness**: Balances freshness with system load
- **100+ users**: Expected dashboard usage

---

## 🔒 DECISION #9: Security Model

### **The Decision**

**JWT-based authentication with role-based access control (RBAC).**

### **Roles**

1. **Admin** - Full access (read + write)
2. **Auditor** - Read + audit management
3. **Viewer** - Read-only

### **Why This Decision?**

✅ **Stateless** - No session storage needed  
✅ **Scalable** - Works with multiple API instances  
✅ **Standard** - Industry best practice  

---

## 📋 Summary of Critical Decisions

| # | Decision | Impact | Priority |
|---|----------|--------|----------|
| 1 | Computed fields from repayments | **HIGH** - Affects ETL design | ⚠️ CRITICAL |
| 2 | Separate microservice | **HIGH** - Affects architecture | ⚠️ CRITICAL |
| 3 | Go (Golang) | **MEDIUM** - Affects development | Important |
| 4 | Pre-aggregation | **HIGH** - Affects performance | ⚠️ CRITICAL |
| 5 | Redis caching | **MEDIUM** - Affects performance | Important |
| 6 | Hybrid sync | **HIGH** - Affects data freshness | ⚠️ CRITICAL |
| 7 | Supporting tables | **HIGH** - Affects accuracy | ⚠️ CRITICAL |
| 8 | Performance targets | **MEDIUM** - Affects UX | Important |
| 9 | JWT + RBAC | **LOW** - Standard practice | Standard |

---

## 🚨 What You MUST Understand

### **For Backend Developers:**
1. ⚠️ **Never send computed fields from main backend** - They will be overwritten
2. ⚠️ **Repayments are the source of truth** - All calculations derive from this
3. ⚠️ **Triggers run automatically** - No manual intervention needed

### **For ETL Developers:**
1. ⚠️ **Only send 20 fields for loans** - See `ETL_DATA_FLOW_SPECIFICATION.md`
2. ⚠️ **Send all 19 fields for repayments** - Complete data required
3. ⚠️ **Send loan schedules** - Required for DPD calculation

### **For DevOps:**
1. ⚠️ **Deploy as separate service** - Don't bundle with main backend
2. ⚠️ **Set up Redis** - Required for caching
3. ⚠️ **Schedule batch jobs** - Every 15-30 minutes

---

## 📚 Related Documentation

- **ETL_DATA_FLOW_SPECIFICATION.md** - Complete ETL specification
- **BACKEND_ARCHITECTURE.md** - Full technical details
- **SQL_MIGRATION_SCRIPTS.sql** - Database setup with triggers

---

**Last Updated:** 2024-10-18  
**Status:** ✅ Approved and Documented

