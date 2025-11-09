# AYR (Annualized Yield Rate) Calculation Documentation

## Executive Summary

**AYR (Annualized Yield Rate)** is a key performance metric in the Seeds Metrics platform that measures the yield (interest + fees collected) relative to the portfolio at risk. It is calculated at the **officer level** and aggregated to the **portfolio level**.

**Current Portfolio Metrics:**
- **Average AYR**: 0.1968 (19.68%)
- **Top Officer AYR**: 123.6961 (12,369.61%)
- **Officers with AYR > 0**: 465 out of 4,546 (10.2%)
- **Median AYR**: 0.0000 (most officers have zero AYR due to zero principal outstanding)

---

## 1. AYR Formula

### Mathematical Formula

```
AYR = (Interest Collected + Fees Collected) / PAR15 Mid-Month
```

Where:
- **Numerator**: Total yield collected (interest + fees) from repayments
- **Denominator**: PAR15 Mid-Month (Portfolio at Risk 15+ days at mid-month)

### Current Implementation

**Note**: In the current implementation, `PAR15 Mid-Month` is **NOT** a snapshot taken on the 15th of each month. Instead, it is calculated as:

```sql
par15_mid_month = SUM(principal_outstanding)
```

This means the denominator is the **total principal outstanding** across all loans for the officer, regardless of their DPD (Days Past Due) status.

---

## 2. Data Sources and SQL Query

### Location in Codebase

**File**: `backend/internal/repository/dashboard_repository.go`  
**Method**: `GetOfficers()` (lines 186-335)  
**Service**: `backend/internal/services/metrics_service.go`  
**Method**: `CalculateOfficerMetrics()` (lines 41-44)

### SQL Query for Raw Metrics

<augment_code_snippet path="backend/internal/repository/dashboard_repository.go" mode="EXCERPT">
````sql
WITH loan_repayments AS (
    SELECT
        l.loan_id,
        l.officer_id,
        l.loan_amount,
        l.interest_rate,
        l.fee_amount,
        SUM(r.payment_amount) as total_repayments
    FROM loans l
    LEFT JOIN repayments r ON l.loan_id = r.loan_id AND r.is_reversed = false
    GROUP BY l.loan_id, l.officer_id, l.loan_amount, l.interest_rate, l.fee_amount
)
SELECT
    -- Calculate fees collected from repayments (proportional allocation)
    COALESCE(SUM(
        CASE
            WHEN lr.loan_amount * (1 + lr.interest_rate) + lr.fee_amount > 0 THEN
                lr.total_repayments * lr.fee_amount / (lr.loan_amount * (1 + lr.interest_rate) + lr.fee_amount)
            ELSE 0
        END
    ), 0) as fees_collected,
    
    -- Calculate interest collected from repayments (proportional allocation)
    COALESCE(SUM(
        CASE
            WHEN lr.loan_amount * (1 + lr.interest_rate) + lr.fee_amount > 0 THEN
                lr.total_repayments * (lr.loan_amount * lr.interest_rate) / (lr.loan_amount * (1 + lr.interest_rate) + lr.fee_amount)
            ELSE 0
        END
    ), 0) as interest_collected,
    
    -- PAR15 mid-month (currently = total principal outstanding)
    COALESCE(SUM(l.principal_outstanding), 0) as par15_mid_month
FROM officers o
LEFT JOIN loans l ON o.officer_id = l.officer_id
LEFT JOIN loan_repayments lr ON l.loan_id = lr.loan_id
WHERE (o.user_type IN ('AGENT', 'AJO_AGENT', 'DMO_AGENT', 'MERCHANT', 'MERCHANT_AGENT', 'MICRO_SAVER', 'PERSONAL', 'PROSPER_AGENT', 'STAFF_AGENT') OR o.user_type IS NULL)
GROUP BY o.officer_id
````
</augment_code_snippet>

### Go Code for AYR Calculation

<augment_code_snippet path="backend/internal/services/metrics_service.go" mode="EXCERPT">
````go
// AYR = (interestCollected + feesCollected) / par15MidMonth
if raw.Par15MidMonth > 0 {
    calculated.AYR = (raw.InterestCollected + raw.FeesCollected) / raw.Par15MidMonth
}
````
</augment_code_snippet>

---

## 3. Detailed Breakdown of Components

### 3.1 Interest Collected (Proportional Allocation)

Interest collected is calculated using **proportional allocation** based on the total repayments made on each loan:

```
Interest Collected (per loan) = Total Repayments × (Loan Amount × Interest Rate) / Total Expected
```

Where:
```
Total Expected = Loan Amount × (1 + Interest Rate) + Fee Amount
```

**Example** (Loan ID 1095, Officer 1053):
- Loan Amount: ₦100,000
- Interest Rate: 30% (0.30)
- Fee Amount: ₦2,000
- Total Repayments: ₦130,000
- Total Expected: ₦100,000 × (1 + 0.30) + ₦2,000 = ₦132,000

```
Interest Collected = ₦130,000 × (₦100,000 × 0.30) / ₦132,000
                   = ₦130,000 × ₦30,000 / ₦132,000
                   = ₦29,545.45
```

### 3.2 Fees Collected (Proportional Allocation)

Fees collected is calculated similarly:

```
Fees Collected (per loan) = Total Repayments × Fee Amount / Total Expected
```

**Example** (Loan ID 1095, Officer 1053):
```
Fees Collected = ₦130,000 × ₦2,000 / ₦132,000
               = ₦1,969.70
```

### 3.3 PAR15 Mid-Month (Current Implementation)

**Current Implementation**:
```sql
par15_mid_month = SUM(principal_outstanding)
```

This is the **total principal outstanding** across all loans for the officer.

**Intended Implementation** (per documentation):
```
PAR15 Mid-Month = Portfolio at Risk (>15 days DPD) measured at mid-month (15th)
```

**Discrepancy**: The current implementation does NOT filter by DPD >= 15, and does NOT take a snapshot on the 15th of the month. It simply sums all principal outstanding.

---

## 4. Filters Applied

### 4.1 User Type Filter

**Included**:
- Officers with `user_type` IN ('AGENT', 'AJO_AGENT', 'DMO_AGENT', 'MERCHANT', 'MERCHANT_AGENT', 'MICRO_SAVER', 'PERSONAL', 'PROSPER_AGENT', 'STAFF_AGENT')
- Officers with `user_type IS NULL` (terminated officers)

**Excluded**:
- Officers with `user_type = 'lite'` (1 loan excluded)

### 4.2 Loan Status Filter

**Included**:
- **ACTIVE** loans (11,869 loans)
- **CLOSED** loans (5,549 loans)

**Note**: The AYR calculation includes **both ACTIVE and CLOSED loans** because:
- The SQL query does NOT filter by loan status
- CLOSED loans have `principal_outstanding = 0` (or negative in some cases)
- CLOSED loans contribute to the numerator (interest + fees collected) but NOT to the denominator (principal outstanding)

### 4.3 Repayment Filter

**Included**:
- All repayments where `is_reversed = false`

**Excluded**:
- Reversed repayments (`is_reversed = true`)

---

## 5. Example Calculation

### Officer 1053 (Top AYR Officer)

**Officer Details**:
- Officer ID: 1053
- Officer Name: obamo2012@gmail.com
- AYR: 123.6961 (12,369.61%)

**Raw Metrics**:
- Fees Collected: ₦181,482.34
- Interest Collected: ₦2,027,730.03
- PAR15 Mid-Month (Principal Outstanding): ₦17,860.00

**AYR Calculation**:
```
AYR = (₦181,482.34 + ₦2,027,730.03) / ₦17,860.00
    = ₦2,209,212.37 / ₦17,860.00
    = 123.6961
```

**Interpretation**: This officer has collected ₦2.2M in interest and fees, but only has ₦17,860 in principal outstanding. This results in an extremely high AYR of 12,369.61%.

**Why is this so high?**
- The officer likely has many **CLOSED loans** that contributed to the numerator (interest + fees collected)
- But very few **ACTIVE loans** with principal outstanding (denominator)
- This creates a very high ratio

---

## 6. Portfolio-Level AYR

### Calculation Method

<augment_code_snippet path="backend/internal/services/metrics_service.go" mode="EXCERPT">
````go
// CalculatePortfolioMetrics calculates portfolio-level metrics from officer metrics
func (s *MetricsService) CalculatePortfolioMetrics(officers []*models.DashboardOfficerMetrics) *models.PortfolioMetrics {
    var totalAYR float64
    
    for _, officer := range officers {
        if officer.CalculatedMetrics != nil {
            totalAYR += officer.CalculatedMetrics.AYR
        }
    }
    
    portfolio.AvgAYR = totalAYR / float64(len(officers))
    
    return portfolio
}
````
</augment_code_snippet>

**Formula**:
```
Average AYR = SUM(Officer AYR) / Total Officers
```

**Current Portfolio Metrics**:
- Total Officers: 4,546
- Officers with AYR > 0: 465 (10.2%)
- Average AYR: 0.1968 (19.68%)
- Median AYR: 0.0000 (50th percentile)
- 75th Percentile AYR: 0.0000

**Why is the average so low?**
- 90% of officers (4,081 out of 4,546) have **AYR = 0**
- This is because they have **zero principal outstanding** (denominator = 0)
- The average is heavily skewed by the 10% of officers with active portfolios

---

## 7. Edge Cases and Special Handling

### 7.1 Zero Denominator

**Condition**: `par15_mid_month = 0` (no principal outstanding)

**Handling**:
```go
if raw.Par15MidMonth > 0 {
    calculated.AYR = (raw.InterestCollected + raw.FeesCollected) / raw.Par15MidMonth
}
// Otherwise, AYR remains 0 (default value)
```

**Result**: AYR = 0

### 7.2 Negative Principal Outstanding

**Observation**: Some CLOSED loans have negative `principal_outstanding` (₦-400,680.15 total)

**Impact**: This can create negative denominators, leading to negative AYR values (though this is rare)

### 7.3 Officers with No Loans

**Condition**: Officer has no loans in the `loans` table

**Handling**: 
- `fees_collected = 0`
- `interest_collected = 0`
- `par15_mid_month = 0`
- **AYR = 0**

---

## 8. Comparison: Average vs Top Officer AYR

| Metric | Average AYR | Top Officer AYR | Difference |
|--------|-------------|-----------------|------------|
| **Value** | 0.1968 (19.68%) | 123.6961 (12,369.61%) | **628x higher** |
| **Numerator** | ₦290.29 (avg per officer) | ₦2,209,212.37 | 7,610x higher |
| **Denominator** | ₦324.60 (avg per officer) | ₦17,860.00 | 55x higher |

**Why the huge difference?**

1. **Portfolio Composition**:
   - Average officer: Mix of active and closed loans, many with zero outstanding
   - Top officer: Mostly closed loans (high collections) with minimal active portfolio

2. **Repayment Performance**:
   - Top officer has collected ₦2.2M in interest + fees
   - This is 7,610x higher than the average officer's collections

3. **Statistical Skew**:
   - 90% of officers have AYR = 0 (no active portfolio)
   - The average is pulled down by these zero values
   - The top officer represents an extreme outlier

---

## 9. Known Issues and Recommendations

### Issue 1: PAR15 Mid-Month Implementation

**Current**: `par15_mid_month = SUM(principal_outstanding)` (all loans)

**Expected**: PAR15 (Portfolio at Risk >15 days) snapshot at mid-month

**Recommendation**: Update the SQL query to:
```sql
par15_mid_month = SUM(CASE WHEN current_dpd >= 15 THEN principal_outstanding ELSE 0 END)
```

**Impact**: This would make the denominator more accurate and align with the intended business logic.

### Issue 2: CLOSED Loans Included

**Current**: CLOSED loans contribute to numerator but not denominator

**Impact**: Creates artificially high AYR values for officers with many closed loans

**Recommendation**: Consider filtering to only ACTIVE loans:
```sql
WHERE UPPER(l.status) = 'ACTIVE'
```

### Issue 3: No Time Period Filter

**Current**: AYR calculation includes ALL repayments (lifetime)

**Expected**: AYR should be calculated for a specific time period (e.g., current month)

**Recommendation**: Add date filters to the repayments query:
```sql
LEFT JOIN repayments r ON l.loan_id = r.loan_id 
    AND r.is_reversed = false
    AND r.payment_date >= DATE_TRUNC('month', CURRENT_DATE)
    AND r.payment_date < DATE_TRUNC('month', CURRENT_DATE) + INTERVAL '1 month'
```

---

## 10. Summary

**AYR Calculation**:
```
AYR = (Interest Collected + Fees Collected) / Total Principal Outstanding
```

**Key Points**:
- Calculated at **officer level**, aggregated to **portfolio level**
- Uses **proportional allocation** for interest and fees from repayments
- Includes **both ACTIVE and CLOSED loans**
- Includes **terminated officers** (NULL user_type)
- **No time period filter** (lifetime collections)
- **No DPD filter** on denominator (should be PAR15 only)

**Current Metrics**:
- Average AYR: 19.68%
- Top Officer AYR: 12,369.61%
- 90% of officers have AYR = 0 (no active portfolio)

**Recommendations**:
1. Implement proper PAR15 calculation (DPD >= 15 filter)
2. Add time period filter for monthly AYR calculation
3. Consider excluding CLOSED loans from the calculation
4. Implement mid-month snapshot for PAR15 denominator

