BEGIN;

DROP TABLE IF EXISTS officer_verticals_temp;

CREATE TABLE officer_verticals_temp (
    loan_officer_id VARCHAR(50),
    loan_officer_email VARCHAR(255),
    loan_officer_name VARCHAR(255),
    loan_officer_phone VARCHAR(50),
    branch_name VARCHAR(255),
    branch_location VARCHAR(255),
    branch_sub_location VARCHAR(255),
    branch_state VARCHAR(255),
    branch_supervisor_email VARCHAR(255),
    branch_supervisor_name VARCHAR(255),
    region_name VARCHAR(255),
    vertical_lead_email VARCHAR(255),
    vertical_lead_name VARCHAR(255)
);

\copy officer_verticals_temp FROM '/tmp/verticals.tsv' WITH (FORMAT csv, DELIMITER E'\t', HEADER true);

-- Quick sanity checks on imported data
SELECT
    COUNT(*) AS total_rows,
    COUNT(DISTINCT loan_officer_id) AS distinct_officers,
    COUNT(DISTINCT region_name) AS distinct_regions
FROM officer_verticals_temp;

-- Update officers with region and branch from verticals
UPDATE officers o
SET
    region = COALESCE(NULLIF(v.region_name, ''), o.region),
    branch = COALESCE(NULLIF(v.branch_sub_location, ''), o.branch)
FROM officer_verticals_temp v
WHERE o.officer_id = v.loan_officer_id;

-- Coverage of officers mapped from verticals
SELECT
    COUNT(*) AS officers_total,
    COUNT(*) FILTER (WHERE v.loan_officer_id IS NOT NULL) AS officers_with_mapping
FROM officers o
LEFT JOIN officer_verticals_temp v ON o.officer_id = v.loan_officer_id;

-- Propagate updated region & branch from officers to loans
UPDATE loans l
SET
    region = o.region,
    branch = o.branch
FROM officers o
WHERE l.officer_id = o.officer_id;

-- Snapshot of loan regions after update
SELECT region, COUNT(*) AS loan_count
FROM loans
GROUP BY region
ORDER BY loan_count DESC
LIMIT 20;

-- Snapshot of loan branches after update
SELECT branch, COUNT(*) AS loan_count
FROM loans
GROUP BY branch
ORDER BY loan_count DESC
LIMIT 20;

COMMIT;

