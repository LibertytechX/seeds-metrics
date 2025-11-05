-- ============================================================================
-- LOAN DISTRIBUTION BY USER TYPE REPORT
-- ============================================================================
-- Description: Comprehensive analysis of loan distribution across user types
-- Database: SeedsMetrics (Analytics Database)
-- Last Updated: 2025-11-05
-- ============================================================================

-- ----------------------------------------------------------------------------
-- QUERY 1: Basic Loan Distribution by User Type
-- ----------------------------------------------------------------------------
-- Shows total loans and percentage for each user type
-- Includes officers with NULL or empty user_type as "Not Specified"

SELECT 
    COALESCE(NULLIF(o.user_type, ''), 'Not Specified') as user_type,
    COUNT(l.loan_id) as total_loans,
    ROUND(COUNT(l.loan_id) * 100.0 / SUM(COUNT(l.loan_id)) OVER (), 2) as percentage
FROM officers o
LEFT JOIN loans l ON o.officer_id = l.officer_id
GROUP BY o.user_type
ORDER BY total_loans DESC;


-- ----------------------------------------------------------------------------
-- QUERY 2: Comprehensive User Type Analysis
-- ----------------------------------------------------------------------------
-- Includes officer count, loan counts, portfolio values, and percentages

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


-- ----------------------------------------------------------------------------
-- QUERY 3: User Types with Loans Only (Exclude Zero Loans)
-- ----------------------------------------------------------------------------
-- Shows only user types that have at least one loan

SELECT 
    COALESCE(NULLIF(o.user_type, ''), 'Not Specified') as user_type,
    COUNT(DISTINCT o.officer_id) as total_officers,
    COUNT(l.loan_id) as total_loans,
    COUNT(CASE WHEN l.status = 'Active' THEN 1 END) as active_loans,
    ROUND(AVG(l.principal_outstanding), 2) as avg_principal_outstanding,
    ROUND(SUM(l.principal_outstanding), 2) as total_principal_outstanding,
    ROUND(COUNT(l.loan_id) * 100.0 / SUM(COUNT(l.loan_id)) OVER (), 2) as percentage_of_total
FROM officers o
INNER JOIN loans l ON o.officer_id = l.officer_id
GROUP BY o.user_type
ORDER BY total_loans DESC;


-- ----------------------------------------------------------------------------
-- QUERY 4: User Type Performance Metrics
-- ----------------------------------------------------------------------------
-- Includes average DPD, FIMR rate, and portfolio quality metrics

SELECT 
    COALESCE(NULLIF(o.user_type, ''), 'Not Specified') as user_type,
    COUNT(DISTINCT o.officer_id) as total_officers,
    COUNT(l.loan_id) as total_loans,
    ROUND(AVG(l.current_dpd), 2) as avg_dpd,
    COUNT(CASE WHEN l.fimr_tagged THEN 1 END) as fimr_loans,
    ROUND(COUNT(CASE WHEN l.fimr_tagged THEN 1 END) * 100.0 / NULLIF(COUNT(l.loan_id), 0), 2) as fimr_rate,
    ROUND(SUM(l.principal_outstanding), 2) as total_portfolio,
    ROUND(SUM(CASE WHEN l.current_dpd >= 15 THEN l.principal_outstanding ELSE 0 END), 2) as par15_portfolio
FROM officers o
LEFT JOIN loans l ON o.officer_id = l.officer_id
GROUP BY o.user_type
ORDER BY total_loans DESC;


-- ----------------------------------------------------------------------------
-- QUERY 5: Officers Without Loans by User Type
-- ----------------------------------------------------------------------------
-- Shows user types where officers exist but have no loans

SELECT 
    COALESCE(NULLIF(o.user_type, ''), 'Not Specified') as user_type,
    COUNT(DISTINCT o.officer_id) as total_officers,
    COUNT(l.loan_id) as total_loans
FROM officers o
LEFT JOIN loans l ON o.officer_id = l.officer_id
GROUP BY o.user_type
HAVING COUNT(l.loan_id) = 0
ORDER BY total_officers DESC;


-- ============================================================================
-- USAGE INSTRUCTIONS
-- ============================================================================
--
-- To run these queries on the production database:
--
-- 1. SSH to production server:
--    ssh root@143.198.146.44
--
-- 2. Connect to SeedsMetrics database:
--    psql "postgresql://seedsuser:@seedsuser2020@generaldb-do-user-9489371-0.k.db.ondigitalocean.com:25060/seedsmetrics?sslmode=require"
--
-- 3. Run desired query by copying and pasting from above
--
-- Alternatively, run from command line:
--    ssh root@143.198.146.44 'psql "postgresql://seedsuser:%40seedsuser2020@generaldb-do-user-9489371-0.k.db.ondigitalocean.com:25060/seedsmetrics?sslmode=require" -f /path/to/this/file.sql'
--
-- ============================================================================

