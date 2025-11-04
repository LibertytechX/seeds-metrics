-- Migration: Fix repayment_amount field for all loans
--
-- Issue: The loan sync was using daily_repayment_amount instead of repayment_amount
-- - repayment_amount: Total amount to be repaid over the life of the loan (principal + interest + fees)
-- - daily_repayment_amount: Amount to be paid per installment
--
-- This migration will be applied by re-running the sync script with the corrected field mapping
-- The sync script now uses l.repayment_amount instead of l.daily_repayment_amount

-- For manual verification, you can check loan 6:
-- Django: SELECT id, amount, repayment_amount, daily_repayment_amount FROM loans_ajoloan WHERE id = 6;
-- Expected: repayment_amount = 57500, daily_repayment_amount = 1916.67
--
-- SeedsMetrics: SELECT loan_id, loan_amount, repayment_amount FROM loans WHERE loan_id = '6';
-- After fix: repayment_amount should be 57500.00

-- This file serves as documentation of the fix
-- The actual data update will be done by re-running the sync script

