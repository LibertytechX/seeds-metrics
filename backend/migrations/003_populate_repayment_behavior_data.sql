-- ============================================================================
-- Migration: Populate Sample Data for Repayment Behavior Fields
-- Description: Add realistic test data for timeliness_score and repayment_health
-- ============================================================================

-- Update loans with timeliness_score and repayment_health
-- Strategy: Create a mix of high, medium, and low performers

-- HIGH PERFORMERS (Scores 80-100) - 4 loans
-- BENCH_LN008, BENCH_LN009, BENCH_LN010 (have recent repayments)
UPDATE loans SET 
    timeliness_score = 92.50,
    repayment_health = 88.75
WHERE loan_id = 'BENCH_LN008';

UPDATE loans SET 
    timeliness_score = 95.00,
    repayment_health = 93.25
WHERE loan_id = 'BENCH_LN009';

UPDATE loans SET 
    timeliness_score = 89.50,
    repayment_health = 91.00
WHERE loan_id = 'BENCH_LN010';

UPDATE loans SET 
    timeliness_score = 87.25,
    repayment_health = 85.50
WHERE loan_id = 'BENCH_LN001';

-- MEDIUM PERFORMERS (Scores 50-79) - 4 loans
UPDATE loans SET 
    timeliness_score = 72.50,
    repayment_health = 68.00
WHERE loan_id = 'BENCH_LN002';

UPDATE loans SET 
    timeliness_score = 65.75,
    repayment_health = 71.25
WHERE loan_id = 'BENCH_LN003';

UPDATE loans SET 
    timeliness_score = 58.00,
    repayment_health = 62.50
WHERE loan_id = 'BENCH_LN007';

UPDATE loans SET 
    timeliness_score = 76.25,
    repayment_health = 73.75
WHERE loan_id = 'LN2024100001';

-- LOW PERFORMERS (Scores 0-49) - 4 loans
UPDATE loans SET 
    timeliness_score = 45.50,
    repayment_health = 42.00
WHERE loan_id = 'BENCH_LN004';

UPDATE loans SET 
    timeliness_score = 38.75,
    repayment_health = 35.25
WHERE loan_id = 'BENCH_LN005';

UPDATE loans SET 
    timeliness_score = 28.00,
    repayment_health = 31.50
WHERE loan_id = 'BENCH_LN006';

UPDATE loans SET 
    timeliness_score = 41.25,
    repayment_health = 39.75
WHERE loan_id = 'LN2024100002';

-- Add some additional repayment records with varying dates to create diversity in days_since_last_repayment
-- This will help test the repayment delay rate calculation

-- Add older repayments for some loans (to increase days_since_last_repayment)
INSERT INTO repayments (
    repayment_id, loan_id, payment_date, payment_amount, 
    principal_paid, interest_paid, fees_paid, payment_method, payment_channel
) VALUES
    -- BENCH_LN001: Last payment 19 days ago (2025-10-11)
    ('REP_TEST_001', 'BENCH_LN001', '2025-10-11', 15000.00, 12000.00, 2500.00, 500.00, 'Bank Transfer', 'Mobile'),
    
    -- BENCH_LN002: Last payment 34 days ago (2025-09-26)
    ('REP_TEST_002', 'BENCH_LN002', '2025-09-26', 18000.00, 15000.00, 2500.00, 500.00, 'Cash', 'Agent'),
    
    -- BENCH_LN003: Last payment 49 days ago (2025-09-11)
    ('REP_TEST_003', 'BENCH_LN003', '2025-09-11', 12000.00, 10000.00, 1800.00, 200.00, 'Bank Transfer', 'Mobile'),
    
    -- BENCH_LN004: Last payment 65 days ago (2025-08-26)
    ('REP_TEST_004', 'BENCH_LN004', '2025-08-26', 8000.00, 6500.00, 1200.00, 300.00, 'Cash', 'Agent'),
    
    -- BENCH_LN005: Last payment 80 days ago (2025-08-11)
    ('REP_TEST_005', 'BENCH_LN005', '2025-08-11', 10000.00, 8000.00, 1700.00, 300.00, 'Mobile Money', 'Mobile'),
    
    -- BENCH_LN006: Last payment 95 days ago (2025-07-27)
    ('REP_TEST_006', 'BENCH_LN006', '2025-07-27', 7500.00, 6000.00, 1200.00, 300.00, 'Cash', 'Agent'),
    
    -- BENCH_LN007: Last payment 15 days ago (2025-10-15)
    ('REP_TEST_007', 'BENCH_LN007', '2025-10-15', 20000.00, 16000.00, 3500.00, 500.00, 'Bank Transfer', 'Mobile'),
    
    -- LN2024100001: Last payment 52 days ago (already has this from trigger)
    -- LN2024100002: Last payment 26 days ago (already has this from trigger)
    
    -- Add a very recent payment for BENCH_LN001 to test low days_since_last_repayment
    ('REP_TEST_008', 'BENCH_LN001', '2025-10-28', 15000.00, 12000.00, 2500.00, 500.00, 'Bank Transfer', 'Mobile')
ON CONFLICT (repayment_id) DO NOTHING;

-- Verify the updates
SELECT 
    loan_id,
    timeliness_score,
    repayment_health,
    days_since_last_repayment,
    CASE 
        WHEN timeliness_score >= 80 THEN 'High Performer'
        WHEN timeliness_score >= 50 THEN 'Medium Performer'
        ELSE 'Low Performer'
    END as performance_category
FROM loans
ORDER BY timeliness_score DESC;

-- Show summary statistics
SELECT 
    COUNT(*) as total_loans,
    ROUND(AVG(timeliness_score), 2) as avg_timeliness,
    ROUND(AVG(repayment_health), 2) as avg_health,
    ROUND(AVG(days_since_last_repayment), 1) as avg_days_since_payment,
    MIN(timeliness_score) as min_timeliness,
    MAX(timeliness_score) as max_timeliness
FROM loans
WHERE timeliness_score IS NOT NULL;

-- Show officer-level aggregates (preview of what the dashboard will show)
SELECT 
    o.officer_name,
    COUNT(l.loan_id) as total_loans,
    COUNT(CASE WHEN (l.principal_outstanding + l.interest_outstanding + l.fees_outstanding) > 2000 THEN 1 END) as loans_over_2k,
    ROUND(AVG(CASE WHEN (l.principal_outstanding + l.interest_outstanding + l.fees_outstanding) > 2000 THEN l.timeliness_score END), 2) as avg_timeliness_score,
    ROUND(AVG(CASE WHEN (l.principal_outstanding + l.interest_outstanding + l.fees_outstanding) > 2000 THEN l.repayment_health END), 2) as avg_repayment_health,
    ROUND(AVG(CASE WHEN (l.principal_outstanding + l.interest_outstanding + l.fees_outstanding) > 2000 THEN l.days_since_last_repayment END), 1) as avg_days_since_last_repayment,
    ROUND(AVG(CASE WHEN (l.principal_outstanding + l.interest_outstanding + l.fees_outstanding) > 2000 THEN (CURRENT_DATE - l.disbursement_date::date) END), 1) as avg_loan_age
FROM officers o
LEFT JOIN loans l ON o.officer_id = l.officer_id
GROUP BY o.officer_id, o.officer_name
ORDER BY avg_timeliness_score DESC NULLS LAST;

COMMIT;

