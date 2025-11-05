# Loan Distribution by User Type - Summary Report

**Report Date**: 2025-11-05  
**Database**: SeedsMetrics (Analytics Database)  
**Total Loans Analyzed**: 17,419  
**Total Officers**: 6,778  

---

## Executive Summary

This report analyzes the distribution of loans across different user types in the Seeds Metrics system. The analysis reveals that **STAFF_AGENT** and officers with **unspecified user types** account for nearly 100% of all loans in the portfolio.

### Key Findings

1. **STAFF_AGENT** dominates the portfolio with **54.20%** of all loans (9,441 loans)
2. **Not Specified** (officers without user_type) accounts for **45.63%** (7,948 loans)
3. Only **7 user types** have active loans; **7 user types** have zero loans
4. **829 MERCHANT** officers exist but only **1 loan** is associated with this user type
5. **421 LOTTO_AGENT** officers exist but have **zero loans**

---

## Detailed Analysis

### Table 1: Loan Distribution by User Type (All User Types)

| User Type       | Total Officers | Total Loans | Active Loans | Avg Principal | Total Portfolio (₦) | % of Total |
|-----------------|----------------|-------------|--------------|---------------|---------------------|------------|
| STAFF_AGENT     | 350            | 9,441       | 6,402        | 79,688.19     | 752,336,225.80      | 54.20%     |
| Not Specified   | 2,744          | 7,948       | 5,455        | 88,072.93     | 700,003,627.26      | 45.63%     |
| PERSONAL        | 275            | 23          | 9            | 7,169.57      | 164,900.00          | 0.13%      |
| AJO_AGENT       | 67             | 3           | 1            | 0.00          | 0.00                | 0.02%      |
| AGENT           | 251            | 2           | 1            | 147,500.00    | 295,000.00          | 0.01%      |
| MERCHANT        | 829            | 1           | 1            | 0.00          | 0.00                | 0.01%      |
| lite            | 1,788          | 1           | 1            | 533,416.25    | 533,416.25          | 0.01%      |
| LIBERTY_RETAIL  | 14             | 0           | 0            | -             | -                   | 0.00%      |
| PHARMACIST      | 9              | 0           | 0            | -             | -                   | 0.00%      |
| MERCHANT_AGENT  | 9              | 0           | 0            | -             | -                   | 0.00%      |
| MICRO_SAVER     | 1              | 0           | 0            | -             | -                   | 0.00%      |
| DMO_AGENT       | 16             | 0           | 0            | -             | -                   | 0.00%      |
| LOTTO_AGENT     | 421            | 0           | 0            | -             | -                   | 0.00%      |
| PROSPER_AGENT   | 4              | 0           | 0            | -             | -                   | 0.00%      |

**Total**: 6,778 officers | 17,419 loans | 11,870 active loans | ₦1,453,333,169.31 total portfolio

---

## Key Insights

### 1. User Type Distribution

```
STAFF_AGENT     ████████████████████████████████████████████████████ 54.20%
Not Specified   ██████████████████████████████████████████████       45.63%
PERSONAL        ▌ 0.13%
Others          ▌ 0.04%
```

### 2. Portfolio Concentration

- **Top 2 user types** (STAFF_AGENT + Not Specified) control **99.83%** of all loans
- **Remaining 12 user types** share only **0.17%** of loans (30 loans total)

### 3. Officer Utilization

**High Officer Count, Low Loan Activity:**
- **MERCHANT**: 829 officers → 1 loan (0.12% utilization)
- **LOTTO_AGENT**: 421 officers → 0 loans (0% utilization)
- **lite**: 1,788 officers → 1 loan (0.06% utilization)
- **AGENT**: 251 officers → 2 loans (0.80% utilization)

**High Officer Count, High Loan Activity:**
- **STAFF_AGENT**: 350 officers → 9,441 loans (27 loans per officer avg)
- **Not Specified**: 2,744 officers → 7,948 loans (2.9 loans per officer avg)

### 4. Active vs Total Loans

| User Type     | Total Loans | Active Loans | Active % |
|---------------|-------------|--------------|----------|
| STAFF_AGENT   | 9,441       | 6,402        | 67.8%    |
| Not Specified | 7,948       | 5,455        | 68.6%    |
| PERSONAL      | 23          | 9            | 39.1%    |
| AJO_AGENT     | 3           | 1            | 33.3%    |
| AGENT         | 2           | 1            | 50.0%    |
| MERCHANT      | 1           | 1            | 100.0%   |
| lite          | 1           | 1            | 100.0%   |

**Overall Active Rate**: 68.2% (11,870 active out of 17,419 total)

---

## Recommendations

### 1. Data Quality Improvement

**Issue**: 45.63% of loans are associated with officers who have no user_type specified.

**Recommendation**: 
- Implement data validation to require user_type during officer creation
- Run a data cleanup script to assign user_type to the 2,744 officers currently marked as "Not Specified"
- Review the sync process from Django database to ensure user_type is properly populated

### 2. User Type Utilization Analysis

**Issue**: Several user types have many officers but zero or very few loans.

**Recommendation**:
- **LOTTO_AGENT** (421 officers, 0 loans): Investigate why these officers have no loans
- **MERCHANT** (829 officers, 1 loan): Determine if these officers are inactive or if there's a data sync issue
- **lite** (1,788 officers, 1 loan): Review if this user type is still relevant or if officers should be reclassified

### 3. Portfolio Monitoring

**Issue**: Portfolio is heavily concentrated in STAFF_AGENT user type.

**Recommendation**:
- Monitor STAFF_AGENT portfolio health closely as it represents 54% of total portfolio value
- Diversify loan distribution across user types to reduce concentration risk
- Set up alerts for significant changes in STAFF_AGENT portfolio metrics

### 4. Officer Productivity Analysis

**Observation**: STAFF_AGENT officers are significantly more productive (27 loans/officer) compared to other types.

**Recommendation**:
- Analyze what makes STAFF_AGENT officers more productive
- Consider training programs to improve productivity of other user types
- Review compensation/incentive structures across user types

---

## Technical Details

### SQL Query Used

```sql
SELECT 
    COALESCE(NULLIF(o.user_type, ''), 'Not Specified') as user_type,
    COUNT(DISTINCT o.officer_id) as total_officers,
    COUNT(l.loan_id) as total_loans,
    COUNT(CASE WHEN l.status = 'Active' THEN 1 END) as active_loans,
    ROUND(AVG(l.principal_outstanding), 2) as avg_principal_outstanding,
    ROUND(SUM(l.principal_outstanding), 2) as total_principal_outstanding,
    ROUND(COUNT(l.loan_id) * 100.0 / SUM(COUNT(l.loan_id)) OVER (), 2) as percentage_of_total
FROM officers o
LEFT JOIN loans l ON o.officer_id = l.officer_id
GROUP BY o.user_type
ORDER BY total_loans DESC;
```

### Database Connection

- **Database**: SeedsMetrics (Analytics)
- **Host**: generaldb-do-user-9489371-0.k.db.ondigitalocean.com
- **Port**: 25060
- **Tables**: `officers`, `loans`

### Data Freshness

This report is based on data as of **2025-11-05**. For the most current data, re-run the queries in `reports/loan_distribution_by_user_type.sql`.

---

## Appendix: User Type Definitions

| User Type       | Description                                      |
|-----------------|--------------------------------------------------|
| STAFF_AGENT     | Internal staff members acting as loan officers   |
| MERCHANT        | Merchant partners                                |
| AGENT           | External agents                                  |
| LOTTO_AGENT     | Lottery agents                                   |
| AJO_AGENT       | Ajo (savings group) agents                       |
| PERSONAL        | Personal/individual officers                     |
| lite            | Lite version users                               |
| LIBERTY_RETAIL  | Liberty retail partners                          |
| PHARMACIST      | Pharmacy partners                                |
| MERCHANT_AGENT  | Merchant agents (hybrid)                         |
| MICRO_SAVER     | Micro savings officers                           |
| DMO_AGENT       | DMO agents                                       |
| PROSPER_AGENT   | Prosper program agents                           |

---

**Report Generated By**: Seeds Metrics Analytics System  
**For Questions**: Contact system administrator  
**Related Files**: `reports/loan_distribution_by_user_type.sql`

