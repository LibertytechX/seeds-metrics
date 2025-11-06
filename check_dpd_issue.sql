-- Investigation: Current DPD Issue

-- 1. Check loans with NULL first_payment_due_date
SELECT 
    COUNT(*) as total_loans,
    COUNT(CASE WHEN first_payment_due_date IS NULL THEN 1 END) as null_first_due_date,
    COUNT(CASE WHEN first_payment_due_date IS NOT NULL THEN 1 END) as has_first_due_date
FROM loans;

-- 2. Check loans with NULL first_payment_due_date and their current_dpd
SELECT 
    loan_id,
    customer_name,
    status,
    disbursement_date,
    first_payment_due_date,
    current_dpd,
    days_since_due
FROM loans
WHERE first_payment_due_date IS NULL
LIMIT 20;

-- 3. Check if there are loans with NULL current_dpd but have first_payment_due_date
SELECT 
    COUNT(*) as loans_with_null_dpd_but_due_date
FROM loans
WHERE current_dpd IS NULL
AND first_payment_due_date IS NOT NULL;

-- 4. Check the issue: current_dpd calculation when first_payment_due_date is NULL
-- According to trigger: v_current_dpd := GREATEST(0, (CURRENT_DATE - v_first_due_date)::INTEGER)
-- If v_first_due_date is NULL, this will result in NULL
SELECT 
    COUNT(*) as loans_with_null_dpd_and_null_due_date
FROM loans
WHERE current_dpd IS NULL
AND first_payment_due_date IS NULL;

-- 5. Check loans with NULL current_dpd overall
SELECT 
    COUNT(*) as total_null_dpd
FROM loans
WHERE current_dpd IS NULL;

-- 6. Check the distribution of current_dpd values
SELECT 
    CASE 
        WHEN current_dpd IS NULL THEN 'NULL'
        WHEN current_dpd = 0 THEN '0 (Current)'
        WHEN current_dpd BETWEEN 1 AND 6 THEN '1-6 (Early Indicator)'
        WHEN current_dpd BETWEEN 7 AND 15 THEN '7-15 (Overdue)'
        WHEN current_dpd > 15 THEN '>15 (Severely Overdue)'
    END as dpd_category,
    COUNT(*) as count,
    ROUND(100.0 * COUNT(*) / (SELECT COUNT(*) FROM loans), 2) as percentage
FROM loans
GROUP BY dpd_category
ORDER BY count DESC;

-- 7. Sample loans with NULL current_dpd
SELECT 
    loan_id,
    customer_name,
    status,
    disbursement_date,
    first_payment_due_date,
    first_payment_received_date,
    current_dpd,
    days_since_due,
    total_outstanding
FROM loans
WHERE current_dpd IS NULL
LIMIT 10;

-- 8. Check if the issue is related to loan status
SELECT 
    status,
    COUNT(*) as total,
    COUNT(CASE WHEN current_dpd IS NULL THEN 1 END) as null_dpd,
    COUNT(CASE WHEN current_dpd IS NOT NULL THEN 1 END) as has_dpd
FROM loans
GROUP BY status;

-- 9. Check if there's a pattern with disbursement_date
SELECT 
    COUNT(*) as total_loans,
    COUNT(CASE WHEN disbursement_date IS NULL THEN 1 END) as null_disbursement,
    COUNT(CASE WHEN disbursement_date IS NOT NULL THEN 1 END) as has_disbursement
FROM loans;

-- 10. Check the trigger function to see if it's being called
SELECT 
    COUNT(*) as total_repayments
FROM repayments;

