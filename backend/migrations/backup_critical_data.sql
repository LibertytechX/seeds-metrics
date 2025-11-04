-- Backup Critical Data Before Truncation
-- Date: 2025-11-04
-- Purpose: Export critical computed data that cannot be easily regenerated

-- Export computed fields from loans table (these are calculated by triggers)
\copy (SELECT loan_id, current_dpd, max_dpd_ever, first_payment_missed, first_payment_due_date, first_payment_received_date, principal_outstanding, interest_outstanding, fees_outstanding, total_outstanding, total_principal_paid, total_interest_paid, total_fees_paid, total_repayments, fimr_tagged, early_indicator_tagged, days_since_last_repayment, actual_outstanding, repayment_delay_rate, timeliness_score, repayment_health FROM loans) TO '/tmp/loans_computed_backup.csv' WITH CSV HEADER;

-- Export aggregation tables
\copy officer_metrics_daily TO '/tmp/officer_metrics_daily_backup.csv' WITH CSV HEADER;
\copy branch_metrics_daily TO '/tmp/branch_metrics_daily_backup.csv' WITH CSV HEADER;
\copy dpd_transitions TO '/tmp/dpd_transitions_backup.csv' WITH CSV HEADER;
\copy par15_snapshots TO '/tmp/par15_snapshots_backup.csv' WITH CSV HEADER;

-- Export team members
\copy team_members TO '/tmp/team_members_backup.csv' WITH CSV HEADER;

-- Record counts for verification
\echo 'Backup completed. Record counts:'
SELECT 'loans_computed' as table_name, COUNT(*) as records FROM loans
UNION ALL
SELECT 'officer_metrics_daily', COUNT(*) FROM officer_metrics_daily
UNION ALL
SELECT 'branch_metrics_daily', COUNT(*) FROM branch_metrics_daily
UNION ALL
SELECT 'dpd_transitions', COUNT(*) FROM dpd_transitions
UNION ALL
SELECT 'par15_snapshots', COUNT(*) FROM par15_snapshots
UNION ALL
SELECT 'team_members', COUNT(*) FROM team_members;

