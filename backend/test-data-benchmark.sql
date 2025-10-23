-- ============================================================================
-- PREDICTABLE BENCHMARK TEST DATA
-- ============================================================================
-- This script creates a curated, predictable dataset for benchmarking
-- the frontend dashboard against known metrics.
--
-- EXPECTED BENCHMARKS:
-- - Total Loans: 10
-- - FIMR Loans: 4 (40% FIMR rate)
-- - Early Indicator Loans (DPD 1-30): 3
-- - Performing Loans: 3
-- - Total Officers: 3
-- - Total Customers: 10
-- - Total Portfolio: ₦50,000,000
-- ============================================================================

BEGIN;

-- Clean up existing test data
DELETE FROM repayments WHERE loan_id LIKE 'BENCH%';
DELETE FROM loan_schedule WHERE loan_id LIKE 'BENCH%';
DELETE FROM loans WHERE loan_id LIKE 'BENCH%';
DELETE FROM customers WHERE customer_id LIKE 'BENCH%';
DELETE FROM officers WHERE officer_id LIKE 'BENCH%';

-- ============================================================================
-- STEP 1: Create Officers
-- ============================================================================

INSERT INTO officers (officer_id, officer_name, officer_phone, officer_email, region, branch, employment_status, hire_date, created_at)
VALUES
    ('BENCH_OFF001', 'Alice Okonkwo', '+234-801-111-1111', 'alice.okonkwo@company.com', 'South West', 'Lagos Main', 'Active', '2023-01-15', CURRENT_TIMESTAMP),
    ('BENCH_OFF002', 'Bob Adeyemi', '+234-802-222-2222', 'bob.adeyemi@company.com', 'South West', 'Ikeja Branch', 'Active', '2023-03-20', CURRENT_TIMESTAMP),
    ('BENCH_OFF003', 'Carol Nwosu', '+234-803-333-3333', 'carol.nwosu@company.com', 'South East', 'Enugu Branch', 'Active', '2023-06-10', CURRENT_TIMESTAMP);

-- ============================================================================
-- STEP 2: Create Customers
-- ============================================================================

INSERT INTO customers (customer_id, customer_name, customer_phone, state, kyc_status, created_at)
VALUES
    ('BENCH_CUST001', 'John Adekunle', '+234-901-111-1111', 'Lagos', 'Verified', CURRENT_TIMESTAMP),
    ('BENCH_CUST002', 'Mary Okafor', '+234-901-222-2222', 'Lagos', 'Verified', CURRENT_TIMESTAMP),
    ('BENCH_CUST003', 'Peter Chukwu', '+234-901-333-3333', 'Lagos', 'Verified', CURRENT_TIMESTAMP),
    ('BENCH_CUST004', 'Grace Bello', '+234-901-444-4444', 'Lagos', 'Verified', CURRENT_TIMESTAMP),
    ('BENCH_CUST005', 'David Obi', '+234-901-555-5555', 'Enugu', 'Verified', CURRENT_TIMESTAMP),
    ('BENCH_CUST006', 'Sarah Musa', '+234-901-666-6666', 'Lagos', 'Verified', CURRENT_TIMESTAMP),
    ('BENCH_CUST007', 'James Eze', '+234-901-777-7777', 'Enugu', 'Verified', CURRENT_TIMESTAMP),
    ('BENCH_CUST008', 'Ruth Akinola', '+234-901-888-8888', 'Lagos', 'Verified', CURRENT_TIMESTAMP),
    ('BENCH_CUST009', 'Samuel Okoro', '+234-901-999-9999', 'Lagos', 'Verified', CURRENT_TIMESTAMP),
    ('BENCH_CUST010', 'Esther Uche', '+234-901-000-0000', 'Enugu', 'Verified', CURRENT_TIMESTAMP);

-- ============================================================================
-- STEP 3: Create Loans with Predictable Patterns
-- ============================================================================

-- CATEGORY 1: FIMR LOANS (4 loans - missed first payment)
-- ============================================================================

-- LOAN 1: FIMR - Missed first 5 days, then paid
INSERT INTO loans (
    loan_id, customer_id, customer_name, customer_phone,
    officer_id, officer_name, officer_phone, region, branch, state,
    loan_amount, disbursement_date, maturity_date, loan_term_days,
    interest_rate, fee_amount, channel, status, created_at
) VALUES (
    'BENCH_LN001', 'BENCH_CUST001', 'John Adekunle', '+234-901-111-1111',
    'BENCH_OFF001', 'Alice Okonkwo', '+234-801-111-1111', 'South West', 'Lagos Main', 'Lagos',
    5000000.00, CURRENT_DATE - INTERVAL '90 days', CURRENT_DATE + INTERVAL '90 days', 180,
    0.10, 250000.00, 'Direct', 'Active', CURRENT_TIMESTAMP
);

-- LOAN 2: FIMR - Missed first 10 days, then sporadic payments
INSERT INTO loans (
    loan_id, customer_id, customer_name, customer_phone,
    officer_id, officer_name, officer_phone, region, branch, state,
    loan_amount, disbursement_date, maturity_date, loan_term_days,
    interest_rate, fee_amount, channel, status, created_at
) VALUES (
    'BENCH_LN002', 'BENCH_CUST002', 'Mary Okafor', '+234-901-222-2222',
    'BENCH_OFF001', 'Alice Okonkwo', '+234-801-111-1111', 'South West', 'Lagos Main', 'Lagos',
    3000000.00, CURRENT_DATE - INTERVAL '75 days', CURRENT_DATE + INTERVAL '105 days', 180,
    0.10, 150000.00, 'Agent', 'Active', CURRENT_TIMESTAMP
);

-- LOAN 3: FIMR - Missed first 7 days, now in default
INSERT INTO loans (
    loan_id, customer_id, customer_name, customer_phone,
    officer_id, officer_name, officer_phone, region, branch, state,
    loan_amount, disbursement_date, maturity_date, loan_term_days,
    interest_rate, fee_amount, channel, status, created_at
) VALUES (
    'BENCH_LN003', 'BENCH_CUST003', 'Peter Chukwu', '+234-901-333-3333',
    'BENCH_OFF002', 'Bob Adeyemi', '+234-802-222-2222', 'South West', 'Ikeja Branch', 'Lagos',
    2000000.00, CURRENT_DATE - INTERVAL '60 days', CURRENT_DATE + INTERVAL '120 days', 180,
    0.10, 100000.00, 'Direct', 'Active', CURRENT_TIMESTAMP
);

-- LOAN 4: FIMR - Missed first 3 days, then caught up
INSERT INTO loans (
    loan_id, customer_id, customer_name, customer_phone,
    officer_id, officer_name, officer_phone, region, branch, state,
    loan_amount, disbursement_date, maturity_date, loan_term_days,
    interest_rate, fee_amount, channel, status, created_at
) VALUES (
    'BENCH_LN004', 'BENCH_CUST004', 'Grace Bello', '+234-901-444-4444',
    'BENCH_OFF002', 'Bob Adeyemi', '+234-802-222-2222', 'South West', 'Ikeja Branch', 'Lagos',
    4000000.00, CURRENT_DATE - INTERVAL '50 days', CURRENT_DATE + INTERVAL '130 days', 180,
    0.10, 200000.00, 'Agent', 'Active', CURRENT_TIMESTAMP
);

-- CATEGORY 2: EARLY INDICATOR LOANS (3 loans - DPD 1-30, but paid first installment)
-- ============================================================================

-- LOAN 5: Early Indicator - Paid first, now DPD 5
INSERT INTO loans (
    loan_id, customer_id, customer_name, customer_phone,
    officer_id, officer_name, officer_phone, region, branch, state,
    loan_amount, disbursement_date, maturity_date, loan_term_days,
    interest_rate, fee_amount, channel, status, created_at
) VALUES (
    'BENCH_LN005', 'BENCH_CUST005', 'David Obi', '+234-901-555-5555',
    'BENCH_OFF003', 'Carol Nwosu', '+234-803-333-3333', 'South East', 'Enugu Branch', 'Enugu',
    6000000.00, CURRENT_DATE - INTERVAL '40 days', CURRENT_DATE + INTERVAL '140 days', 180,
    0.10, 300000.00, 'Direct', 'Active', CURRENT_TIMESTAMP
);

-- LOAN 6: Early Indicator - Paid first, now DPD 12
INSERT INTO loans (
    loan_id, customer_id, customer_name, customer_phone,
    officer_id, officer_name, officer_phone, region, branch, state,
    loan_amount, disbursement_date, maturity_date, loan_term_days,
    interest_rate, fee_amount, channel, status, created_at
) VALUES (
    'BENCH_LN006', 'BENCH_CUST006', 'Sarah Musa', '+234-901-666-6666',
    'BENCH_OFF003', 'Carol Nwosu', '+234-803-333-3333', 'South East', 'Enugu Branch', 'Lagos',
    7000000.00, CURRENT_DATE - INTERVAL '35 days', CURRENT_DATE + INTERVAL '145 days', 180,
    0.10, 350000.00, 'Agent', 'Active', CURRENT_TIMESTAMP
);

-- LOAN 7: Early Indicator - Paid first, now DPD 20
INSERT INTO loans (
    loan_id, customer_id, customer_name, customer_phone,
    officer_id, officer_name, officer_phone, region, branch, state,
    loan_amount, disbursement_date, maturity_date, loan_term_days,
    interest_rate, fee_amount, channel, status, created_at
) VALUES (
    'BENCH_LN007', 'BENCH_CUST007', 'James Eze', '+234-901-777-7777',
    'BENCH_OFF001', 'Alice Okonkwo', '+234-801-111-1111', 'South West', 'Lagos Main', 'Enugu',
    8000000.00, CURRENT_DATE - INTERVAL '30 days', CURRENT_DATE + INTERVAL '150 days', 180,
    0.10, 400000.00, 'Direct', 'Active', CURRENT_TIMESTAMP
);

-- CATEGORY 3: PERFORMING LOANS (3 loans - paying on time)
-- ============================================================================

-- LOAN 8: Performing - Perfect payment record
INSERT INTO loans (
    loan_id, customer_id, customer_name, customer_phone,
    officer_id, officer_name, officer_phone, region, branch, state,
    loan_amount, disbursement_date, maturity_date, loan_term_days,
    interest_rate, fee_amount, channel, status, created_at
) VALUES (
    'BENCH_LN008', 'BENCH_CUST008', 'Ruth Akinola', '+234-901-888-8888',
    'BENCH_OFF002', 'Bob Adeyemi', '+234-802-222-2222', 'South West', 'Ikeja Branch', 'Lagos',
    10000000.00, CURRENT_DATE - INTERVAL '25 days', CURRENT_DATE + INTERVAL '155 days', 180,
    0.10, 500000.00, 'Direct', 'Active', CURRENT_TIMESTAMP
);

-- LOAN 9: Performing - Excellent payment record
INSERT INTO loans (
    loan_id, customer_id, customer_name, customer_phone,
    officer_id, officer_name, officer_phone, region, branch, state,
    loan_amount, disbursement_date, maturity_date, loan_term_days,
    interest_rate, fee_amount, channel, status, created_at
) VALUES (
    'BENCH_LN009', 'BENCH_CUST009', 'Samuel Okoro', '+234-901-999-9999',
    'BENCH_OFF003', 'Carol Nwosu', '+234-803-333-3333', 'South East', 'Enugu Branch', 'Lagos',
    3000000.00, CURRENT_DATE - INTERVAL '20 days', CURRENT_DATE + INTERVAL '160 days', 180,
    0.10, 150000.00, 'Agent', 'Active', CURRENT_TIMESTAMP
);

-- LOAN 10: Performing - Good payment record
INSERT INTO loans (
    loan_id, customer_id, customer_name, customer_phone,
    officer_id, officer_name, officer_phone, region, branch, state,
    loan_amount, disbursement_date, maturity_date, loan_term_days,
    interest_rate, fee_amount, channel, status, created_at
) VALUES (
    'BENCH_LN010', 'BENCH_CUST010', 'Esther Uche', '+234-901-000-0000',
    'BENCH_OFF001', 'Alice Okonkwo', '+234-801-111-1111', 'South West', 'Lagos Main', 'Enugu',
    2000000.00, CURRENT_DATE - INTERVAL '15 days', CURRENT_DATE + INTERVAL '165 days', 180,
    0.10, 100000.00, 'Direct', 'Active', CURRENT_TIMESTAMP
);

-- ============================================================================
-- STEP 4: Create Loan Schedules (180 daily installments per loan)
-- ============================================================================

-- Helper function to calculate daily installment amounts
-- For each loan: Total Due = Principal + Interest + Fees
-- Daily Installment = Total Due / 180 days

-- LOAN 1: ₦5,000,000 + ₦500,000 interest + ₦250,000 fees = ₦5,750,000 / 180 = ₦31,944.44/day
INSERT INTO loan_schedule (loan_id, installment_number, due_date, principal_due, interest_due, fee_due, total_due, payment_status)
SELECT
    'BENCH_LN001',
    day_num,
    (CURRENT_DATE - INTERVAL '90 days') + day_num * INTERVAL '1 day',
    5000000.00 / 180,
    500000.00 / 180,
    250000.00 / 180,
    5750000.00 / 180,
    CASE
        WHEN day_num BETWEEN 6 AND 80 THEN 'Paid'
        ELSE 'Pending'
    END
FROM generate_series(1, 180) AS day_num;

-- LOAN 2: ₦3,000,000 + ₦300,000 + ₦150,000 = ₦3,450,000 / 180 = ₦19,166.67/day
INSERT INTO loan_schedule (loan_id, installment_number, due_date, principal_due, interest_due, fee_due, total_due, payment_status)
SELECT
    'BENCH_LN002',
    day_num,
    (CURRENT_DATE - INTERVAL '75 days') + day_num * INTERVAL '1 day',
    3000000.00 / 180,
    300000.00 / 180,
    150000.00 / 180,
    3450000.00 / 180,
    CASE
        WHEN day_num BETWEEN 11 AND 30 THEN 'Paid'
        WHEN day_num BETWEEN 40 AND 50 THEN 'Paid'
        ELSE 'Pending'
    END
FROM generate_series(1, 180) AS day_num;

-- LOAN 3: ₦2,000,000 + ₦200,000 + ₦100,000 = ₦2,300,000 / 180 = ₦12,777.78/day
INSERT INTO loan_schedule (loan_id, installment_number, due_date, principal_due, interest_due, fee_due, total_due, payment_status)
SELECT
    'BENCH_LN003',
    day_num,
    (CURRENT_DATE - INTERVAL '60 days') + day_num * INTERVAL '1 day',
    2000000.00 / 180,
    200000.00 / 180,
    100000.00 / 180,
    2300000.00 / 180,
    CASE
        WHEN day_num BETWEEN 8 AND 20 THEN 'Paid'
        ELSE 'Pending'
    END
FROM generate_series(1, 180) AS day_num;

-- LOAN 4: ₦4,000,000 + ₦400,000 + ₦200,000 = ₦4,600,000 / 180 = ₦25,555.56/day
INSERT INTO loan_schedule (loan_id, installment_number, due_date, principal_due, interest_due, fee_due, total_due, payment_status)
SELECT
    'BENCH_LN004',
    day_num,
    (CURRENT_DATE - INTERVAL '50 days') + day_num * INTERVAL '1 day',
    4000000.00 / 180,
    400000.00 / 180,
    200000.00 / 180,
    4600000.00 / 180,
    CASE
        WHEN day_num BETWEEN 4 AND 45 THEN 'Paid'
        ELSE 'Pending'
    END
FROM generate_series(1, 180) AS day_num;

-- LOAN 5: ₦6,000,000 + ₦600,000 + ₦300,000 = ₦6,900,000 / 180 = ₦38,333.33/day
INSERT INTO loan_schedule (loan_id, installment_number, due_date, principal_due, interest_due, fee_due, total_due, payment_status)
SELECT
    'BENCH_LN005',
    day_num,
    (CURRENT_DATE - INTERVAL '40 days') + day_num * INTERVAL '1 day',
    6000000.00 / 180,
    600000.00 / 180,
    300000.00 / 180,
    6900000.00 / 180,
    CASE
        WHEN day_num BETWEEN 1 AND 35 THEN 'Paid'
        ELSE 'Pending'
    END
FROM generate_series(1, 180) AS day_num;

-- LOAN 6: ₦7,000,000 + ₦700,000 + ₦350,000 = ₦8,050,000 / 180 = ₦44,722.22/day
INSERT INTO loan_schedule (loan_id, installment_number, due_date, principal_due, interest_due, fee_due, total_due, payment_status)
SELECT
    'BENCH_LN006',
    day_num,
    (CURRENT_DATE - INTERVAL '35 days') + day_num * INTERVAL '1 day',
    7000000.00 / 180,
    700000.00 / 180,
    350000.00 / 180,
    8050000.00 / 180,
    CASE
        WHEN day_num BETWEEN 1 AND 23 THEN 'Paid'
        ELSE 'Pending'
    END
FROM generate_series(1, 180) AS day_num;

-- LOAN 7: ₦8,000,000 + ₦800,000 + ₦400,000 = ₦9,200,000 / 180 = ₦51,111.11/day
INSERT INTO loan_schedule (loan_id, installment_number, due_date, principal_due, interest_due, fee_due, total_due, payment_status)
SELECT
    'BENCH_LN007',
    day_num,
    (CURRENT_DATE - INTERVAL '30 days') + day_num * INTERVAL '1 day',
    8000000.00 / 180,
    800000.00 / 180,
    400000.00 / 180,
    9200000.00 / 180,
    CASE
        WHEN day_num BETWEEN 1 AND 10 THEN 'Paid'
        ELSE 'Pending'
    END
FROM generate_series(1, 180) AS day_num;

-- LOAN 8: ₦10,000,000 + ₦1,000,000 + ₦500,000 = ₦11,500,000 / 180 = ₦63,888.89/day
INSERT INTO loan_schedule (loan_id, installment_number, due_date, principal_due, interest_due, fee_due, total_due, payment_status)
SELECT
    'BENCH_LN008',
    day_num,
    (CURRENT_DATE - INTERVAL '25 days') + day_num * INTERVAL '1 day',
    10000000.00 / 180,
    1000000.00 / 180,
    500000.00 / 180,
    11500000.00 / 180,
    CASE
        WHEN day_num BETWEEN 1 AND 25 THEN 'Paid'
        ELSE 'Pending'
    END
FROM generate_series(1, 180) AS day_num;

-- LOAN 9: ₦3,000,000 + ₦300,000 + ₦150,000 = ₦3,450,000 / 180 = ₦19,166.67/day
INSERT INTO loan_schedule (loan_id, installment_number, due_date, principal_due, interest_due, fee_due, total_due, payment_status)
SELECT
    'BENCH_LN009',
    day_num,
    (CURRENT_DATE - INTERVAL '20 days') + day_num * INTERVAL '1 day',
    3000000.00 / 180,
    300000.00 / 180,
    150000.00 / 180,
    3450000.00 / 180,
    CASE
        WHEN day_num BETWEEN 1 AND 20 THEN 'Paid'
        ELSE 'Pending'
    END
FROM generate_series(1, 180) AS day_num;

-- LOAN 10: ₦2,000,000 + ₦200,000 + ₦100,000 = ₦2,300,000 / 180 = ₦12,777.78/day
INSERT INTO loan_schedule (loan_id, installment_number, due_date, principal_due, interest_due, fee_due, total_due, payment_status)
SELECT
    'BENCH_LN010',
    day_num,
    (CURRENT_DATE - INTERVAL '15 days') + day_num * INTERVAL '1 day',
    2000000.00 / 180,
    200000.00 / 180,
    100000.00 / 180,
    2300000.00 / 180,
    CASE
        WHEN day_num BETWEEN 1 AND 15 THEN 'Paid'
        ELSE 'Pending'
    END
FROM generate_series(1, 180) AS day_num;

-- ============================================================================
-- STEP 5: Create Repayments
-- ============================================================================

-- LOAN 1: FIMR - Missed days 1-5, paid days 6-80 (75 payments)
INSERT INTO repayments (repayment_id, loan_id, payment_date, payment_amount, principal_paid, interest_paid, fees_paid, penalty_paid, payment_method, is_backdated, is_reversed, waiver_amount)
SELECT
    'BENCH_REP001_' || LPAD(day_num::TEXT, 3, '0'),
    'BENCH_LN001',
    (CURRENT_DATE - INTERVAL '90 days') + (day_num - 1) * INTERVAL '1 day',
    31944.44,
    27777.78,
    2777.78,
    1388.89,
    0,
    'Bank Transfer',
    false,
    false,
    0
FROM generate_series(6, 80) AS day_num;

-- LOAN 2: FIMR - Missed days 1-10, paid days 11-30 and 40-50 (31 payments)
INSERT INTO repayments (repayment_id, loan_id, payment_date, payment_amount, principal_paid, interest_paid, fees_paid, penalty_paid, payment_method, is_backdated, is_reversed, waiver_amount)
SELECT
    'BENCH_REP002_' || LPAD(day_num::TEXT, 3, '0'),
    'BENCH_LN002',
    (CURRENT_DATE - INTERVAL '75 days') + (day_num - 1) * INTERVAL '1 day',
    19166.67,
    16666.67,
    1666.67,
    833.33,
    0,
    'Cash',
    false,
    false,
    0
FROM generate_series(11, 30) AS day_num
UNION ALL
SELECT
    'BENCH_REP002_' || LPAD(day_num::TEXT, 3, '0'),
    'BENCH_LN002',
    (CURRENT_DATE - INTERVAL '75 days') + (day_num - 1) * INTERVAL '1 day',
    19166.67,
    16666.67,
    1666.67,
    833.33,
    0,
    'Cash',
    false,
    false,
    0
FROM generate_series(40, 50) AS day_num;

-- LOAN 3: FIMR - Missed days 1-7, paid days 8-20 (13 payments)
INSERT INTO repayments (repayment_id, loan_id, payment_date, payment_amount, principal_paid, interest_paid, fees_paid, penalty_paid, payment_method, is_backdated, is_reversed, waiver_amount)
SELECT
    'BENCH_REP003_' || LPAD(day_num::TEXT, 3, '0'),
    'BENCH_LN003',
    (CURRENT_DATE - INTERVAL '60 days') + (day_num - 1) * INTERVAL '1 day',
    12777.78,
    11111.11,
    1111.11,
    555.56,
    0,
    'Bank Transfer',
    false,
    false,
    0
FROM generate_series(8, 20) AS day_num;

-- LOAN 4: FIMR - Missed days 1-3, paid days 4-45 (42 payments)
INSERT INTO repayments (repayment_id, loan_id, payment_date, payment_amount, principal_paid, interest_paid, fees_paid, penalty_paid, payment_method, is_backdated, is_reversed, waiver_amount)
SELECT
    'BENCH_REP004_' || LPAD(day_num::TEXT, 3, '0'),
    'BENCH_LN004',
    (CURRENT_DATE - INTERVAL '50 days') + (day_num - 1) * INTERVAL '1 day',
    25555.56,
    22222.22,
    2222.22,
    1111.11,
    0,
    'Mobile Money',
    false,
    false,
    0
FROM generate_series(4, 45) AS day_num;

-- LOAN 5: Early Indicator - Paid days 1-35 (35 payments, now DPD 5)
INSERT INTO repayments (repayment_id, loan_id, payment_date, payment_amount, principal_paid, interest_paid, fees_paid, penalty_paid, payment_method, is_backdated, is_reversed, waiver_amount)
SELECT
    'BENCH_REP005_' || LPAD(day_num::TEXT, 3, '0'),
    'BENCH_LN005',
    (CURRENT_DATE - INTERVAL '40 days') + day_num * INTERVAL '1 day',
    38333.33,
    33333.33,
    3333.33,
    1666.67,
    0,
    'Bank Transfer',
    false,
    false,
    0
FROM generate_series(1, 35) AS day_num;

-- LOAN 6: Early Indicator - Paid days 1-23 (23 payments, now DPD 12)
INSERT INTO repayments (repayment_id, loan_id, payment_date, payment_amount, principal_paid, interest_paid, fees_paid, penalty_paid, payment_method, is_backdated, is_reversed, waiver_amount)
SELECT
    'BENCH_REP006_' || LPAD(day_num::TEXT, 3, '0'),
    'BENCH_LN006',
    (CURRENT_DATE - INTERVAL '35 days') + day_num * INTERVAL '1 day',
    44722.22,
    38888.89,
    3888.89,
    1944.44,
    0,
    'Cash',
    false,
    false,
    0
FROM generate_series(1, 23) AS day_num;

-- LOAN 7: Early Indicator - Paid days 1-10 (10 payments, now DPD 20)
INSERT INTO repayments (repayment_id, loan_id, payment_date, payment_amount, principal_paid, interest_paid, fees_paid, penalty_paid, payment_method, is_backdated, is_reversed, waiver_amount)
SELECT
    'BENCH_REP007_' || LPAD(day_num::TEXT, 3, '0'),
    'BENCH_LN007',
    (CURRENT_DATE - INTERVAL '30 days') + day_num * INTERVAL '1 day',
    51111.11,
    44444.44,
    4444.44,
    2222.22,
    0,
    'Bank Transfer',
    false,
    false,
    0
FROM generate_series(1, 10) AS day_num;

-- LOAN 8: Performing - Paid days 1-25 (25 payments, current)
INSERT INTO repayments (repayment_id, loan_id, payment_date, payment_amount, principal_paid, interest_paid, fees_paid, penalty_paid, payment_method, is_backdated, is_reversed, waiver_amount)
SELECT
    'BENCH_REP008_' || LPAD(day_num::TEXT, 3, '0'),
    'BENCH_LN008',
    (CURRENT_DATE - INTERVAL '25 days') + day_num * INTERVAL '1 day',
    63888.89,
    55555.56,
    5555.56,
    2777.78,
    0,
    'Bank Transfer',
    false,
    false,
    0
FROM generate_series(1, 25) AS day_num;

-- LOAN 9: Performing - Paid days 1-20 (20 payments, current)
INSERT INTO repayments (repayment_id, loan_id, payment_date, payment_amount, principal_paid, interest_paid, fees_paid, penalty_paid, payment_method, is_backdated, is_reversed, waiver_amount)
SELECT
    'BENCH_REP009_' || LPAD(day_num::TEXT, 3, '0'),
    'BENCH_LN009',
    (CURRENT_DATE - INTERVAL '20 days') + day_num * INTERVAL '1 day',
    19166.67,
    16666.67,
    1666.67,
    833.33,
    0,
    'Mobile Money',
    false,
    false,
    0
FROM generate_series(1, 20) AS day_num;

-- LOAN 10: Performing - Paid days 1-15 (15 payments, current)
INSERT INTO repayments (repayment_id, loan_id, payment_date, payment_amount, principal_paid, interest_paid, fees_paid, penalty_paid, payment_method, is_backdated, is_reversed, waiver_amount)
SELECT
    'BENCH_REP010_' || LPAD(day_num::TEXT, 3, '0'),
    'BENCH_LN010',
    (CURRENT_DATE - INTERVAL '15 days') + day_num * INTERVAL '1 day',
    12777.78,
    11111.11,
    1111.11,
    555.56,
    0,
    'Cash',
    false,
    false,
    0
FROM generate_series(1, 15) AS day_num;

COMMIT;

-- ============================================================================
-- VERIFICATION QUERIES
-- ============================================================================

-- Show summary of all loans
SELECT
    loan_id,
    customer_name,
    officer_name,
    loan_amount,
    disbursement_date,
    first_payment_missed,
    fimr_tagged,
    current_dpd,
    (SELECT COUNT(*) FROM repayments WHERE repayments.loan_id = loans.loan_id) as payment_count,
    (SELECT MIN(payment_date) FROM repayments WHERE repayments.loan_id = loans.loan_id) as first_payment_date,
    (SELECT MIN(due_date) FROM loan_schedule WHERE loan_schedule.loan_id = loans.loan_id) as first_due_date
FROM loans
WHERE loan_id LIKE 'BENCH%'
ORDER BY loan_id;

-- Show FIMR summary
SELECT
    'FIMR Loans' as category,
    COUNT(*) as count,
    SUM(loan_amount) as total_amount
FROM loans
WHERE loan_id LIKE 'BENCH%' AND fimr_tagged = true
UNION ALL
SELECT
    'Non-FIMR Loans' as category,
    COUNT(*) as count,
    SUM(loan_amount) as total_amount
FROM loans
WHERE loan_id LIKE 'BENCH%' AND fimr_tagged = false
UNION ALL
SELECT
    'Total Loans' as category,
    COUNT(*) as count,
    SUM(loan_amount) as total_amount
FROM loans
WHERE loan_id LIKE 'BENCH%';

-- Show DPD distribution
SELECT
    CASE
        WHEN current_dpd = 0 THEN 'Current (DPD 0)'
        WHEN current_dpd BETWEEN 1 AND 7 THEN 'Early (DPD 1-7)'
        WHEN current_dpd BETWEEN 8 AND 30 THEN 'Moderate (DPD 8-30)'
        WHEN current_dpd > 30 THEN 'Severe (DPD >30)'
    END as dpd_category,
    COUNT(*) as loan_count,
    SUM(loan_amount) as total_amount
FROM loans
WHERE loan_id LIKE 'BENCH%'
GROUP BY
    CASE
        WHEN current_dpd = 0 THEN 'Current (DPD 0)'
        WHEN current_dpd BETWEEN 1 AND 7 THEN 'Early (DPD 1-7)'
        WHEN current_dpd BETWEEN 8 AND 30 THEN 'Moderate (DPD 8-30)'
        WHEN current_dpd > 30 THEN 'Severe (DPD >30)'
    END
ORDER BY dpd_category;


