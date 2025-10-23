# 📊 Benchmark Test Data Specification

## Overview

This document describes the **predictable, curated test dataset** designed for benchmarking the frontend dashboard against known metrics.

---

## 🎯 Expected Benchmarks (Summary)

| Metric | Expected Value |
|--------|----------------|
| **Total Loans** | 10 |
| **FIMR Loans** | 4 (40% FIMR rate) |
| **Early Indicator Loans (DPD 1-30, non-FIMR)** | 3 |
| **Performing Loans (DPD 0)** | 3 |
| **Total Officers** | 3 |
| **Total Customers** | 10 |
| **Total Portfolio** | ₦50,000,000 |
| **Total Repayments** | 286 payments |

---

## 👥 Officers (3)

| Officer ID | Name | Region | Branch | Loans Assigned |
|------------|------|--------|--------|----------------|
| BENCH_OFF001 | Alice Okonkwo | South West | Lagos Main | 4 loans |
| BENCH_OFF002 | Bob Adeyemi | South West | Ikeja Branch | 3 loans |
| BENCH_OFF003 | Carol Nwosu | South East | Enugu Branch | 3 loans |

---

## 💰 Loan Portfolio Breakdown

### **Category 1: FIMR Loans (4 loans - 40%)**

These loans **missed their first payment** (first installment was not paid on time).

#### **LOAN 1: John Adekunle - FIMR (Missed 5 days)**
- **Loan ID:** BENCH_LN001
- **Customer:** John Adekunle (BENCH_CUST001)
- **Officer:** Alice Okonkwo (BENCH_OFF001)
- **Amount:** ₦5,000,000
- **Disbursement:** 90 days ago
- **Term:** 180 days
- **Interest:** 10% (₦500,000)
- **Fees:** ₦250,000
- **Total Due:** ₦5,750,000
- **Daily Installment:** ₦31,944.44
- **Channel:** Direct
- **FIMR Pattern:**
  - Days 1-5: **MISSED** (FIMR!)
  - Days 6-80: Paid (75 payments)
  - Days 81+: Pending
- **Expected Metrics:**
  - FIMR Tagged: ✅ TRUE
  - First Payment Missed: ✅ TRUE
  - Current DPD: ~10 days
  - Payments Made: 75

#### **LOAN 2: Mary Okafor - FIMR (Missed 10 days)**
- **Loan ID:** BENCH_LN002
- **Customer:** Mary Okafor (BENCH_CUST002)
- **Officer:** Alice Okonkwo (BENCH_OFF001)
- **Amount:** ₦3,000,000
- **Disbursement:** 75 days ago
- **Term:** 180 days
- **Interest:** 10% (₦300,000)
- **Fees:** ₦150,000
- **Total Due:** ₦3,450,000
- **Daily Installment:** ₦19,166.67
- **Channel:** Agent
- **FIMR Pattern:**
  - Days 1-10: **MISSED** (FIMR!)
  - Days 11-30: Paid (20 payments)
  - Days 31-39: Missed
  - Days 40-50: Paid (11 payments)
  - Days 51+: Pending
- **Expected Metrics:**
  - FIMR Tagged: ✅ TRUE
  - First Payment Missed: ✅ TRUE
  - Current DPD: ~25 days
  - Payments Made: 31

#### **LOAN 3: Peter Chukwu - FIMR (Missed 7 days)**
- **Loan ID:** BENCH_LN003
- **Customer:** Peter Chukwu (BENCH_CUST003)
- **Officer:** Bob Adeyemi (BENCH_OFF002)
- **Amount:** ₦2,000,000
- **Disbursement:** 60 days ago
- **Term:** 180 days
- **Interest:** 10% (₦200,000)
- **Fees:** ₦100,000
- **Total Due:** ₦2,300,000
- **Daily Installment:** ₦12,777.78
- **Channel:** Direct
- **FIMR Pattern:**
  - Days 1-7: **MISSED** (FIMR!)
  - Days 8-20: Paid (13 payments)
  - Days 21+: Pending
- **Expected Metrics:**
  - FIMR Tagged: ✅ TRUE
  - First Payment Missed: ✅ TRUE
  - Current DPD: ~40 days
  - Payments Made: 13

#### **LOAN 4: Grace Bello - FIMR (Missed 3 days)**
- **Loan ID:** BENCH_LN004
- **Customer:** Grace Bello (BENCH_CUST004)
- **Officer:** Bob Adeyemi (BENCH_OFF002)
- **Amount:** ₦4,000,000
- **Disbursement:** 50 days ago
- **Term:** 180 days
- **Interest:** 10% (₦400,000)
- **Fees:** ₦200,000
- **Total Due:** ₦4,600,000
- **Daily Installment:** ₦25,555.56
- **Channel:** Agent
- **FIMR Pattern:**
  - Days 1-3: **MISSED** (FIMR!)
  - Days 4-45: Paid (42 payments)
  - Days 46+: Pending
- **Expected Metrics:**
  - FIMR Tagged: ✅ TRUE
  - First Payment Missed: ✅ TRUE
  - Current DPD: ~5 days
  - Payments Made: 42

---

### **Category 2: Early Indicator Loans (3 loans - 30%)**

These loans **paid their first payment on time** but are now in early delinquency (DPD 1-30).

#### **LOAN 5: David Obi - Early Indicator (DPD 5)**
- **Loan ID:** BENCH_LN005
- **Customer:** David Obi (BENCH_CUST005)
- **Officer:** Carol Nwosu (BENCH_OFF003)
- **Amount:** ₦6,000,000
- **Disbursement:** 40 days ago
- **Term:** 180 days
- **Interest:** 10% (₦600,000)
- **Fees:** ₦300,000
- **Total Due:** ₦6,900,000
- **Daily Installment:** ₦38,333.33
- **Channel:** Direct
- **Payment Pattern:**
  - Days 1-35: Paid on time (35 payments)
  - Days 36+: Pending (now DPD 5)
- **Expected Metrics:**
  - FIMR Tagged: ❌ FALSE
  - First Payment Missed: ❌ FALSE
  - Current DPD: ~5 days
  - Payments Made: 35

#### **LOAN 6: Sarah Musa - Early Indicator (DPD 12)**
- **Loan ID:** BENCH_LN006
- **Customer:** Sarah Musa (BENCH_CUST006)
- **Officer:** Carol Nwosu (BENCH_OFF003)
- **Amount:** ₦7,000,000
- **Disbursement:** 35 days ago
- **Term:** 180 days
- **Interest:** 10% (₦700,000)
- **Fees:** ₦350,000
- **Total Due:** ₦8,050,000
- **Daily Installment:** ₦44,722.22
- **Channel:** Agent
- **Payment Pattern:**
  - Days 1-23: Paid on time (23 payments)
  - Days 24+: Pending (now DPD 12)
- **Expected Metrics:**
  - FIMR Tagged: ❌ FALSE
  - First Payment Missed: ❌ FALSE
  - Current DPD: ~12 days
  - Payments Made: 23

#### **LOAN 7: James Eze - Early Indicator (DPD 20)**
- **Loan ID:** BENCH_LN007
- **Customer:** James Eze (BENCH_CUST007)
- **Officer:** Alice Okonkwo (BENCH_OFF001)
- **Amount:** ₦8,000,000
- **Disbursement:** 30 days ago
- **Term:** 180 days
- **Interest:** 10% (₦800,000)
- **Fees:** ₦400,000
- **Total Due:** ₦9,200,000
- **Daily Installment:** ₦51,111.11
- **Channel:** Direct
- **Payment Pattern:**
  - Days 1-10: Paid on time (10 payments)
  - Days 11+: Pending (now DPD 20)
- **Expected Metrics:**
  - FIMR Tagged: ❌ FALSE
  - First Payment Missed: ❌ FALSE
  - Current DPD: ~20 days
  - Payments Made: 10

---

### **Category 3: Performing Loans (3 loans - 30%)**

These loans are **paying on time** with no delinquency (DPD 0).

#### **LOAN 8: Ruth Akinola - Performing (Perfect)**
- **Loan ID:** BENCH_LN008
- **Customer:** Ruth Akinola (BENCH_CUST008)
- **Officer:** Bob Adeyemi (BENCH_OFF002)
- **Amount:** ₦10,000,000
- **Disbursement:** 25 days ago
- **Term:** 180 days
- **Interest:** 10% (₦1,000,000)
- **Fees:** ₦500,000
- **Total Due:** ₦11,500,000
- **Daily Installment:** ₦63,888.89
- **Channel:** Direct
- **Payment Pattern:**
  - Days 1-25: Paid on time (25 payments)
  - Current (DPD 0)
- **Expected Metrics:**
  - FIMR Tagged: ❌ FALSE
  - First Payment Missed: ❌ FALSE
  - Current DPD: 0 days
  - Payments Made: 25

#### **LOAN 9: Samuel Okoro - Performing (Excellent)**
- **Loan ID:** BENCH_LN009
- **Customer:** Samuel Okoro (BENCH_CUST009)
- **Officer:** Carol Nwosu (BENCH_OFF003)
- **Amount:** ₦3,000,000
- **Disbursement:** 20 days ago
- **Term:** 180 days
- **Interest:** 10% (₦300,000)
- **Fees:** ₦150,000
- **Total Due:** ₦3,450,000
- **Daily Installment:** ₦19,166.67
- **Channel:** Agent
- **Payment Pattern:**
  - Days 1-20: Paid on time (20 payments)
  - Current (DPD 0)
- **Expected Metrics:**
  - FIMR Tagged: ❌ FALSE
  - First Payment Missed: ❌ FALSE
  - Current DPD: 0 days
  - Payments Made: 20

#### **LOAN 10: Esther Uche - Performing (Good)**
- **Loan ID:** BENCH_LN010
- **Customer:** Esther Uche (BENCH_CUST010)
- **Officer:** Alice Okonkwo (BENCH_OFF001)
- **Amount:** ₦2,000,000
- **Disbursement:** 15 days ago
- **Term:** 180 days
- **Interest:** 10% (₦200,000)
- **Fees:** ₦100,000
- **Total Due:** ₦2,300,000
- **Daily Installment:** ₦12,777.78
- **Channel:** Direct
- **Payment Pattern:**
  - Days 1-15: Paid on time (15 payments)
  - Current (DPD 0)
- **Expected Metrics:**
  - FIMR Tagged: ❌ FALSE
  - First Payment Missed: ❌ FALSE
  - Current DPD: 0 days
  - Payments Made: 15

---

## 📊 Expected Frontend Dashboard Metrics

### **1. KPI Strip (Top of Dashboard)**

| KPI | Expected Value | Calculation |
|-----|----------------|-------------|
| **Portfolio Overdue >15 Days** | ~₦19M - ₦23M | Loans 1, 2, 3, 6, 7 (DPD >15) |
| **Average DQI** | 70-85 | Calculated from all loans |
| **Average AYR** | 0.08-0.12 | Calculated from all loans |
| **Risk Score (Avg)** | 60-75 | Calculated from all loans |
| **Top Performing Officer** | Alice, Bob, or Carol | Officer with best AYR |
| **Watchlist Count** | 1-2 | Officers with risk score <80 |

### **2. FIMR Drilldown Tab**

**Expected: 4 loans visible**

| Loan ID | Customer | Officer | Amount | DPD | FIMR Tagged |
|---------|----------|---------|--------|-----|-------------|
| BENCH_LN001 | John Adekunle | Alice Okonkwo | ₦5,000,000 | ~10 | ✅ TRUE |
| BENCH_LN002 | Mary Okafor | Alice Okonkwo | ₦3,000,000 | ~25 | ✅ TRUE |
| BENCH_LN003 | Peter Chukwu | Bob Adeyemi | ₦2,000,000 | ~40 | ✅ TRUE |
| BENCH_LN004 | Grace Bello | Bob Adeyemi | ₦4,000,000 | ~5 | ✅ TRUE |

**Total FIMR Loans:** 4  
**Total FIMR Amount:** ₦14,000,000  
**FIMR Rate:** 40%

### **3. Early Indicators Drilldown Tab**

**Expected: 6-7 loans visible** (FIMR loans + Early Indicator loans with DPD 1-30)

| Loan ID | Customer | Officer | Amount | DPD | Category |
|---------|----------|---------|--------|-----|----------|
| BENCH_LN004 | Grace Bello | Bob Adeyemi | ₦4,000,000 | ~5 | FIMR + Early |
| BENCH_LN005 | David Obi | Carol Nwosu | ₦6,000,000 | ~5 | Early Indicator |
| BENCH_LN001 | John Adekunle | Alice Okonkwo | ₦5,000,000 | ~10 | FIMR + Early |
| BENCH_LN006 | Sarah Musa | Carol Nwosu | ₦7,000,000 | ~12 | Early Indicator |
| BENCH_LN007 | James Eze | Alice Okonkwo | ₦8,000,000 | ~20 | Early Indicator |
| BENCH_LN002 | Mary Okafor | Alice Okonkwo | ₦3,000,000 | ~25 | FIMR + Early |

**Note:** Loans with DPD >30 (BENCH_LN003) should NOT appear in Early Indicators.

### **4. Agent Performance Tab**

**Expected: 3 officers visible**

| Officer | Region | Branch | Loans | FIMR Count | FIMR Rate |
|---------|--------|--------|-------|------------|-----------|
| Alice Okonkwo | South West | Lagos Main | 4 | 2 | 50% |
| Bob Adeyemi | South West | Ikeja Branch | 3 | 2 | 67% |
| Carol Nwosu | South East | Enugu Branch | 3 | 0 | 0% |

---

## 🔍 How to Load and Verify

### **Step 1: Load the Benchmark Data**

```bash
PGPASSWORD=19sedimat54 psql -h localhost -p 5432 -U postgres -d seedsmetrics -f backend/test-data-benchmark.sql
```

### **Step 2: Verify in Database**

```bash
# Check total loans
PGPASSWORD=19sedimat54 psql -h localhost -p 5432 -U postgres -d seedsmetrics -c "SELECT COUNT(*) FROM loans WHERE loan_id LIKE 'BENCH%';"
# Expected: 10

# Check FIMR loans
PGPASSWORD=19sedimat54 psql -h localhost -p 5432 -U postgres -d seedsmetrics -c "SELECT COUNT(*) FROM loans WHERE loan_id LIKE 'BENCH%' AND fimr_tagged = true;"
# Expected: 4

# Check total repayments
PGPASSWORD=19sedimat54 psql -h localhost -p 5432 -U postgres -d seedsmetrics -c "SELECT COUNT(*) FROM repayments WHERE repayment_id LIKE 'BENCH%';"
# Expected: 286
```

### **Step 3: Verify API Endpoints**

```bash
# FIMR loans
curl -s http://localhost:8080/api/v1/fimr/loans | jq '.data.total'
# Expected: 4

# Early indicators
curl -s http://localhost:8080/api/v1/early-indicators/loans | jq '.data.total'
# Expected: 6-7

# Portfolio metrics
curl -s http://localhost:8080/api/v1/metrics/portfolio | jq '.data.totalLoans'
# Expected: 10 (or 12 if old test data still exists)
```

### **Step 4: Verify Frontend**

1. **Refresh the frontend:** `Cmd + Shift + R`
2. **Open:** http://localhost:5173/
3. **Check FIMR Drilldown tab:** Should see 4 loans
4. **Check Early Indicators tab:** Should see 6-7 loans
5. **Check Agent Performance tab:** Should see 3 officers

---

## ✅ Benchmark Checklist

Use this checklist to verify the frontend matches the expected data:

- [ ] **Total Loans:** 10 loans visible
- [ ] **FIMR Loans:** 4 loans in FIMR Drilldown tab
- [ ] **FIMR Rate:** 40% (4 out of 10)
- [ ] **Early Indicators:** 6-7 loans in Early Indicators tab
- [ ] **Performing Loans:** 3 loans with DPD 0
- [ ] **Officers:** 3 officers in Agent Performance tab
- [ ] **Alice Okonkwo:** 4 loans, 2 FIMR (50% FIMR rate)
- [ ] **Bob Adeyemi:** 3 loans, 2 FIMR (67% FIMR rate)
- [ ] **Carol Nwosu:** 3 loans, 0 FIMR (0% FIMR rate)
- [ ] **Total Portfolio:** ₦50,000,000
- [ ] **John Adekunle (BENCH_LN001):** FIMR tagged, ~10 DPD
- [ ] **Mary Okafor (BENCH_LN002):** FIMR tagged, ~25 DPD
- [ ] **Peter Chukwu (BENCH_LN003):** FIMR tagged, ~40 DPD
- [ ] **Grace Bello (BENCH_LN004):** FIMR tagged, ~5 DPD
- [ ] **David Obi (BENCH_LN005):** NOT FIMR, ~5 DPD
- [ ] **Sarah Musa (BENCH_LN006):** NOT FIMR, ~12 DPD
- [ ] **James Eze (BENCH_LN007):** NOT FIMR, ~20 DPD
- [ ] **Ruth Akinola (BENCH_LN008):** NOT FIMR, 0 DPD
- [ ] **Samuel Okoro (BENCH_LN009):** NOT FIMR, 0 DPD
- [ ] **Esther Uche (BENCH_LN010):** NOT FIMR, 0 DPD

---

## 📁 Files

- **`backend/test-data-benchmark.sql`** - SQL script to load benchmark data
- **`backend/BENCHMARK_TEST_DATA.md`** - This documentation file

---

**This predictable dataset allows you to verify that the frontend correctly displays and calculates all metrics!** ✅

