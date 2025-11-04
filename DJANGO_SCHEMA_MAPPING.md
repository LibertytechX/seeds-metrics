# Django to SeedsMetrics Schema Mapping Reference

## Overview

This document provides detailed field-level mapping between the Django database schema (source of truth) and the SeedsMetrics database schema (analytics platform).

---

## Table of Contents

1. [Officers Mapping](#1-officers-mapping)
2. [Customers Mapping](#2-customers-mapping)
3. [Loans Mapping](#3-loans-mapping)
4. [Repayments Mapping](#4-repayments-mapping)
5. [Loan Schedule Mapping](#5-loan-schedule-mapping)
6. [Data Type Conversions](#6-data-type-conversions)
7. [Business Logic Mappings](#7-business-logic-mappings)

---

## 1. Officers Mapping

### Django Table: `accounts_customuser`
### SeedsMetrics Table: `officers`

| SeedsMetrics Field | Django Field | Transformation | Notes |
|-------------------|--------------|----------------|-------|
| `officer_id` | `id` | `id::VARCHAR(50)` | Cast integer to string |
| `officer_name` | `username` or `email` | `COALESCE(username, email)` | Fallback to email if username null |
| `officer_phone` | `user_phone` | Direct | |
| `officer_email` | `email` | Direct | |
| `region` | N/A | Derive from `user_branch` | Requires branch-to-region mapping table |
| `branch` | `user_branch` | Direct | |
| `employment_status` | `performance_status` | Map to 'Active'/'Inactive' | |
| `hire_date` | `date_joined` | `date_joined::DATE` | Cast to DATE |
| `termination_date` | N/A | `NULL` | Not tracked in Django |
| `primary_channel` | `user_type` | Derive from user_type | See channel mapping below |
| `created_at` | `date_joined` | Direct | |
| `updated_at` | `last_login` or `CURRENT_TIMESTAMP` | Use current time | |

### Filter Criteria

**Django Query:**
```sql
SELECT * FROM accounts_customuser
WHERE user_type IN (
    'AGENT',
    'STAFF_AGENT',
    'PROSPER_AGENT',
    'DMO_AGENT',
    'AJO_AGENT',
    'RECOVERY_AGENT'
)
AND is_active = TRUE;
```

### Channel Mapping

| Django `user_type` | SeedsMetrics `primary_channel` |
|-------------------|-------------------------------|
| AGENT | Direct |
| STAFF_AGENT | Direct |
| PROSPER_AGENT | Partner |
| DMO_AGENT | Partner |
| AJO_AGENT | Direct |
| RECOVERY_AGENT | Direct |

### Region Mapping (Example)

| Branch | Region |
|--------|--------|
| Lagos Central | Lagos |
| Lagos Island | Lagos |
| Abuja Main | FCT |
| Kano Branch | Kano |
| *Default* | Nigeria |

---

## 2. Customers Mapping

### Django Table: `ajo_ajouser`
### SeedsMetrics Table: `customers`

| SeedsMetrics Field | Django Field | Transformation | Notes |
|-------------------|--------------|----------------|-------|
| `customer_id` | `id` | `id::VARCHAR(50)` | Cast integer to string |
| `customer_name` | `first_name`, `last_name` | `COALESCE(first_name \|\| ' ' \|\| last_name, phone_number)` | Fallback to phone if names null |
| `customer_phone` | `phone_number` | Direct | Primary phone |
| `customer_email` | N/A | `NULL` | Not tracked in Django |
| `date_of_birth` | `dob` | Direct | |
| `gender` | `gender` | Direct | |
| `state` | `state` | Direct | |
| `lga` | `lga` | Direct | Local Government Area |
| `address` | `address` | Direct | |
| `kyc_status` | `bvn_verified`, `onboarding_verified` | Derive from verification flags | See KYC mapping below |
| `kyc_verified_date` | N/A | `NULL` | Not tracked in Django |
| `created_at` | `date_created` | Direct | |
| `updated_at` | `date_modified` | Direct | |

### Filter Criteria

**Django Query:**
```sql
SELECT * FROM ajo_ajouser
WHERE onboarding_complete = TRUE;
```

### KYC Status Mapping

| Django Conditions | SeedsMetrics `kyc_status` |
|------------------|--------------------------|
| `bvn_verified = TRUE AND onboarding_verified = TRUE` | Verified |
| `bvn_verified = TRUE AND onboarding_verified = FALSE` | Partial |
| `bvn_verified = FALSE` | Pending |

---

## 3. Loans Mapping

### Django Table: `loans_ajoloan`
### SeedsMetrics Table: `loans`

**IMPORTANT:** Loans table is HYBRID - base fields from Django, computed fields local.

### Base Fields (FROM DJANGO)

| SeedsMetrics Field | Django Field | Transformation | Notes |
|-------------------|--------------|----------------|-------|
| `loan_id` | `id` | `id::VARCHAR(50)` | Cast integer to string |
| `customer_id` | `borrower_id` | `borrower_id::VARCHAR(50)` | Foreign key to AjoUser |
| `customer_name` | `borrower_full_name` | Direct | Cached in loan record |
| `customer_phone` | `borrower_phone_number` | Direct | Cached in loan record |
| `officer_id` | `agent_id` | `agent_id::VARCHAR(50)` | Foreign key to CustomUser |
| `officer_name` | N/A | Join to `accounts_customuser.username` | |
| `officer_phone` | N/A | Join to `accounts_customuser.user_phone` | |
| `region` | N/A | Derive from officer's branch | |
| `branch` | N/A | Join to `accounts_customuser.user_branch` | |
| `state` | N/A | Join to `ajo_ajouser.state` | |
| `loan_amount` | `amount_disbursed` | Direct | Actual disbursed amount |
| `repayment_amount` | `repayment_amount` | Direct | Expected total repayment |
| `disbursement_date` | `date_disbursed` | `date_disbursed::DATE` | |
| `maturity_date` | `expected_end_date` | Direct | |
| `loan_term_days` | `tenor_in_days` | Direct | |
| `interest_rate` | `interest_rate` | `interest_rate / 100.0` | Convert percentage to decimal |
| `fee_amount` | `processing_fee`, `nem_fee` | `processing_fee + nem_fee` | Sum of all fees |
| `channel` | `loan_type` | Map loan_type to channel | See channel mapping |
| `channel_partner` | N/A | `NULL` or derive | |
| `status` | `status` | Direct | PENDING, APPROVED, DISBURSED, COMPLETED, DEFAULTED |
| `closed_date` | `date_completed` | `date_completed::DATE` | |
| `wave` | N/A | Default to 'Wave 2' | Not in Django schema |
| `created_at` | `date_created` | Direct | |
| `updated_at` | `date_modified` | Direct | |

### Computed Fields (LOCAL IN SEEDSMETRICS)

These fields are calculated by triggers in SeedsMetrics database:

| Field | Calculation Source | Trigger |
|-------|-------------------|---------|
| `current_dpd` | `loan_schedule` table | `update_loan_computed_fields()` |
| `max_dpd_ever` | Historical max of `current_dpd` | `update_loan_computed_fields()` |
| `first_payment_missed` | Compare first payment vs due date | `update_loan_computed_fields()` |
| `first_payment_due_date` | First installment in `loan_schedule` | `update_loan_computed_fields()` |
| `first_payment_received_date` | First repayment in `repayments` | `update_loan_computed_fields()` |
| `principal_outstanding` | `loan_amount - SUM(principal_paid)` | `update_loan_computed_fields()` |
| `interest_outstanding` | `calculated_interest - SUM(interest_paid)` | `update_loan_computed_fields()` |
| `fees_outstanding` | `fee_amount - SUM(fees_paid)` | `update_loan_computed_fields()` |
| `total_outstanding` | Sum of all outstanding | `update_loan_computed_fields()` |
| `total_principal_paid` | `SUM(principal_paid)` from repayments | `update_loan_computed_fields()` |
| `total_interest_paid` | `SUM(interest_paid)` from repayments | `update_loan_computed_fields()` |
| `total_fees_paid` | `SUM(fees_paid)` from repayments | `update_loan_computed_fields()` |
| `total_repayments` | `SUM(payment_amount)` from repayments | `update_loan_computed_fields()` |
| `fimr_tagged` | First payment 4+ days late | `update_loan_computed_fields()` |
| `early_indicator_tagged` | `current_dpd BETWEEN 1 AND 6` | `update_loan_computed_fields()` |
| `actual_outstanding` | Outstanding based on schedule | `update_loan_computed_fields()` |
| `timeliness_score` | Payment timeliness metric | `update_loan_computed_fields()` |
| `repayment_health` | Repayment consistency metric | `update_loan_computed_fields()` |
| `days_since_last_repayment` | Days since last payment | `update_loan_computed_fields()` |
| `repayment_delay_rate` | Percentage of on-time payments | `update_loan_computed_fields()` |

### Filter Criteria

**Django Query:**
```sql
SELECT * FROM loans_ajoloan
WHERE is_disbursed = TRUE;
```

### Loan Type to Channel Mapping

| Django `loan_type` | SeedsMetrics `channel` |
|-------------------|----------------------|
| AJO | Direct |
| BNPL | Partner |
| PROSPER | Partner |
| DMO | Partner |
| *Default* | Direct |

### Status Mapping

| Django Status | SeedsMetrics Status | Notes |
|--------------|-------------------|-------|
| PENDING | Pending | Awaiting approval |
| APPROVED | Approved | Approved but not disbursed |
| DISBURSED | Active | Funds disbursed, loan active |
| COMPLETED | Closed | Fully repaid |
| DEFAULTED | Defaulted | In default |
| REJECTED | Rejected | Application rejected |

---

## 4. Repayments Mapping

### Django Table: `loans_ajoloanrepayment`
### SeedsMetrics Table: `repayments`

**STRATEGY:** Full copy to SeedsMetrics (triggers depend on this data)

| SeedsMetrics Field | Django Field | Transformation | Notes |
|-------------------|--------------|----------------|-------|
| `repayment_id` | `id` | `id::VARCHAR(50)` | Cast integer to string |
| `loan_id` | `ajo_loan_id` | `ajo_loan_id::VARCHAR(50)` | Foreign key |
| `payment_date` | `paid_date` | `paid_date::DATE` | |
| `payment_amount` | `repayment_amount` | Direct | Total payment amount |
| `principal_paid` | N/A | Calculate from loan amortization | Not in Django |
| `interest_paid` | N/A | Calculate from loan amortization | Not in Django |
| `fees_paid` | `health_insurance` | Use health_insurance as proxy | Approximate |
| `penalty_paid` | N/A | `0` | Not tracked in Django |
| `payment_method` | `repayment_type` | Direct | CASH, TRANSFER, CARD, USSD |
| `payment_reference` | `repayment_ref` | `repayment_ref::VARCHAR(100)` | UUID |
| `payment_channel` | `repayment_type` | Direct | Same as method |
| `dpd_at_payment` | N/A | Calculate from loan schedule | |
| `is_backdated` | `marked_as` | `marked_as = 'BACKDATED'` | Check marked_as field |
| `is_reversed` | `marked_as` | `marked_as = 'REVERSAL'` | Check marked_as field |
| `reversal_date` | N/A | `NULL` | Not tracked |
| `reversal_reason` | `comment` | Use if is_reversed | |
| `waiver_amount` | N/A | `0` | Not tracked in Django |
| `waiver_type` | N/A | `NULL` | Not tracked |
| `waiver_approved_by` | N/A | `NULL` | Not tracked |
| `created_at` | `date_created` | Direct | |
| `updated_at` | `date_modified` | Direct | |

### Payment Breakdown Calculation

**Challenge:** Django doesn't store principal/interest breakdown per payment.

**Solution:** Calculate using loan amortization schedule:

```sql
-- Calculate principal and interest paid per repayment
WITH loan_details AS (
    SELECT
        loan_id,
        loan_amount,
        interest_rate,
        loan_term_days,
        fee_amount
    FROM loans
),
payment_allocation AS (
    SELECT
        r.repayment_id,
        r.loan_id,
        r.payment_amount,
        -- Allocate payment: fees first, then interest, then principal
        CASE
            WHEN r.payment_amount <= ld.fee_amount THEN 0
            WHEN r.payment_amount <= (ld.fee_amount + (ld.loan_amount * ld.interest_rate * ld.loan_term_days / 365)) THEN 0
            ELSE r.payment_amount - ld.fee_amount - (ld.loan_amount * ld.interest_rate * ld.loan_term_days / 365)
        END as principal_paid,
        CASE
            WHEN r.payment_amount <= ld.fee_amount THEN 0
            WHEN r.payment_amount <= (ld.fee_amount + (ld.loan_amount * ld.interest_rate * ld.loan_term_days / 365)) THEN r.payment_amount - ld.fee_amount
            ELSE (ld.loan_amount * ld.interest_rate * ld.loan_term_days / 365)
        END as interest_paid,
        CASE
            WHEN r.payment_amount <= ld.fee_amount THEN r.payment_amount
            ELSE ld.fee_amount
        END as fees_paid
    FROM repayments r
    JOIN loan_details ld ON r.loan_id = ld.loan_id
)
SELECT * FROM payment_allocation;
```

---

## 5. Loan Schedule Mapping

### Django Table: `loans_ajoloanschedule`
### SeedsMetrics Table: `loan_schedule`

**STRATEGY:** Full copy to SeedsMetrics (required for DPD calculation)

| SeedsMetrics Field | Django Field | Transformation | Notes |
|-------------------|--------------|----------------|-------|
| `schedule_id` | `id` | Auto-increment in SeedsMetrics | Don't use Django ID |
| `loan_id` | `loan_id` | `loan_id::VARCHAR(50)` | Foreign key |
| `installment_number` | N/A | Calculate from `due_date` order | Not in Django |
| `due_date` | `due_date` | Direct | |
| `principal_due` | `due_amount` | Allocate from `due_amount` | Approximate |
| `interest_due` | N/A | Calculate from loan | |
| `fee_due` | N/A | Calculate from loan | |
| `total_due` | `due_amount` | Direct | |
| `payment_status` | `fully_paid`, `is_late` | Derive from flags | See status mapping |
| `amount_paid` | `paid_amount` | Direct | |
| `payment_date` | `paid_date` | Direct | |
| `created_at` | `date_created` | Direct | |
| `updated_at` | `date_modified` | Direct | |

### Payment Status Mapping

| Django Conditions | SeedsMetrics `payment_status` |
|------------------|------------------------------|
| `fully_paid = TRUE` | Paid |
| `paid_amount > 0 AND fully_paid = FALSE` | Partial |
| `due_date < CURRENT_DATE AND paid_amount = 0` | Overdue |
| `due_date >= CURRENT_DATE AND paid_amount = 0` | Pending |

---

## 6. Data Type Conversions

### Integer to String (IDs)

**Django:** `id INTEGER`  
**SeedsMetrics:** `loan_id VARCHAR(50)`

```sql
-- Conversion
id::VARCHAR(50)
```

### Percentage to Decimal (Interest Rate)

**Django:** `interest_rate FLOAT` (e.g., 15.0 for 15%)  
**SeedsMetrics:** `interest_rate DECIMAL(5,4)` (e.g., 0.15 for 15%)

```sql
-- Conversion
interest_rate / 100.0
```

### DateTime to Date

**Django:** `date_disbursed DATETIME`  
**SeedsMetrics:** `disbursement_date DATE`

```sql
-- Conversion
date_disbursed::DATE
```

### Boolean Flags

**Django:** `is_disbursed BOOLEAN`  
**SeedsMetrics:** Same

```sql
-- Direct mapping
is_disbursed
```

---

## 7. Business Logic Mappings

### FIMR Tagging Logic

**Rule:** Loan is FIMR if first payment is 4+ days late

```sql
-- In SeedsMetrics trigger
fimr_tagged = CASE
    WHEN first_payment_received_date IS NULL AND first_payment_due_date IS NOT NULL THEN
        (CURRENT_DATE - first_payment_due_date) >= 4
    WHEN first_payment_received_date IS NOT NULL AND first_payment_due_date IS NOT NULL THEN
        (first_payment_received_date - first_payment_due_date) >= 4
    ELSE
        FALSE
END
```

### Current DPD Calculation

**Rule:** Days past due based on unpaid installments

```sql
-- In SeedsMetrics trigger
current_dpd = (
    SELECT COALESCE(MAX(CURRENT_DATE - due_date), 0)
    FROM loan_schedule
    WHERE loan_id = v_loan_id
      AND payment_status IN ('Pending', 'Partial', 'Overdue')
      AND due_date < CURRENT_DATE
)
```

### Outstanding Balance Calculation

**Rule:** Original amount minus payments

```sql
-- In SeedsMetrics trigger
principal_outstanding = loan_amount - total_principal_paid
interest_outstanding = (loan_amount * interest_rate * loan_term_days / 365) - total_interest_paid
fees_outstanding = fee_amount - total_fees_paid
total_outstanding = principal_outstanding + interest_outstanding + fees_outstanding
```

---

## 8. ETL Sync Strategy

### Incremental Sync (Recommended)

**Frequency:** Every 5-15 minutes

**Tables to Sync:**
1. **Repayments:** Sync new/updated records
2. **Loan Schedule:** Sync new/updated records

**Query:**
```python
# Repayments
last_sync = get_last_sync_timestamp('repayment')
new_repayments = AjoLoanRepayment.objects.filter(
    date_created__gte=last_sync
).select_related('ajo_loan')

# Loan Schedule
new_schedules = AjoLoanSchedule.objects.filter(
    date_modified__gte=last_sync
).select_related('loan')
```

### Full Sync (Backup)

**Frequency:** Daily at 2 AM

**Purpose:** Catch any missed incremental syncs

---

## 9. Missing Fields Handling

### Fields in SeedsMetrics NOT in Django

| Field | Strategy |
|-------|----------|
| `wave` | Default to 'Wave 2' or derive from disbursement_date |
| `channel_partner` | NULL or derive from loan_type |
| `principal_paid` (repayments) | Calculate using amortization |
| `interest_paid` (repayments) | Calculate using amortization |
| `dpd_at_payment` | Calculate from loan_schedule |
| `installment_number` (schedule) | Calculate from due_date order |

### Fields in Django NOT in SeedsMetrics

| Field | Action |
|-------|--------|
| `eligibility_id` | Ignore (not needed for metrics) |
| `guarantor_id` | Ignore (not needed for metrics) |
| `escrow_amount` | Ignore (not needed for metrics) |
| `loandisk_loan_id` | Ignore (legacy system) |

---

**Document Version:** 1.0  
**Last Updated:** 2025-11-04  
**Related:** DJANGO_INTEGRATION_ARCHITECTURE.md

