BEGIN;

-- Script: scripts/update_officers_and_loans_from_roster.sql
-- Purpose: Use agents_roster_new.tsv (export from agents roaster new.xlsx)
--          to update officers.branch and officers.region, then propagate
--          to loans.branch and loans.region.
-- Dry run default: UPDATE statements are commented out; run as-is to see
--                  summary only. When satisfied, uncomment UPDATE blocks
--                  and re-run to apply changes.

DROP TABLE IF EXISTS officer_roster_temp;

CREATE TABLE officer_roster_temp (
    officer_id   VARCHAR(50),
    sheet_branch VARCHAR(255),
    sheet_region VARCHAR(255)
);

-- Load roster data exported as a TSV with header:
-- officer_id   sheet_branch    sheet_region
-- Copy the TSV to the server as /tmp/agents_roster_new.tsv before running.
\copy officer_roster_temp (officer_id, sheet_branch, sheet_region) FROM '/tmp/agents_roster_new.tsv' WITH (FORMAT csv, DELIMITER E'\t', HEADER true);

-- Sanity checks on imported data
SELECT
    COUNT(*)                          AS total_rows,
    COUNT(DISTINCT officer_id)        AS distinct_officers,
    COUNT(DISTINCT sheet_branch)      AS distinct_branches,
    COUNT(DISTINCT sheet_region)      AS distinct_regions
FROM officer_roster_temp;

-- Officers in DB that have roster data with non-empty branch/region
SELECT
    COUNT(*) AS officers_in_db,
    COUNT(*) FILTER (
        WHERE r.officer_id IS NOT NULL
          AND (
              (r.sheet_branch IS NOT NULL AND r.sheet_branch <> '')
           OR (r.sheet_region IS NOT NULL AND r.sheet_region <> '')
          )
    ) AS officers_with_roster_branch_or_region
FROM officers o
LEFT JOIN officer_roster_temp r
    ON o.officer_id::text = r.officer_id::text;

-- Preview of officers where branch/region would change
SELECT
    o.officer_id,
    o.officer_name,
    o.branch        AS current_branch,
    o.region        AS current_region,
    r.sheet_branch  AS new_branch,
    r.sheet_region  AS new_region
FROM officers o
JOIN officer_roster_temp r
    ON o.officer_id::text = r.officer_id::text
WHERE
    (r.sheet_branch IS NOT NULL AND r.sheet_branch <> '' AND COALESCE(o.branch, '') <> r.sheet_branch)
 OR (r.sheet_region IS NOT NULL AND r.sheet_region <> '' AND COALESCE(o.region, '') <> r.sheet_region)
ORDER BY o.officer_id
LIMIT 50;

-- Preview of affected loans per officer (no updates yet)
WITH changed_officers AS (
    SELECT o.officer_id
    FROM officers o
    JOIN officer_roster_temp r
        ON o.officer_id::text = r.officer_id::text
    WHERE
        (r.sheet_branch IS NOT NULL AND r.sheet_branch <> '' AND COALESCE(o.branch, '') <> r.sheet_branch)
     OR (r.sheet_region IS NOT NULL AND r.sheet_region <> '' AND COALESCE(o.region, '') <> r.sheet_region)
)
SELECT
    l.officer_id,
    COUNT(*) AS loan_count
FROM loans l
JOIN changed_officers c
    ON l.officer_id = c.officer_id
GROUP BY l.officer_id
ORDER BY loan_count DESC
LIMIT 20;

-- =============================
-- ACTUAL UPDATE (COMMENTED OUT)
-- =============================
-- When you are satisfied with the preview above, remove the leading "--"
-- from the UPDATE statements below and re-run this script to apply changes.

UPDATE officers o
SET
    branch = COALESCE(NULLIF(r.sheet_branch, ''), o.branch),
    region = COALESCE(NULLIF(r.sheet_region, ''), o.region),
    updated_at = CURRENT_TIMESTAMP
FROM officer_roster_temp r
WHERE o.officer_id::text = r.officer_id::text
  AND (
      (r.sheet_branch IS NOT NULL AND r.sheet_branch <> '' AND COALESCE(o.branch, '') <> r.sheet_branch)
   OR (r.sheet_region IS NOT NULL AND r.sheet_region <> '' AND COALESCE(o.region, '') <> r.sheet_region)
  );

-- Propagate updated branch/region from officers to loans
UPDATE loans l
SET
    branch = o.branch,
    region = o.region
FROM officers o
WHERE l.officer_id = o.officer_id;

COMMIT;

