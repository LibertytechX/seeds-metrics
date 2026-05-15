-- Migration 048: Make repayment sync/recalculation resilient to late loan availability
--
-- Context: repayment incremental sync can see a Django repayment before its loan has
-- been loaded into Seeds Metrics. Those repayments must be retried after the loan
-- appears instead of being lost when the sync watermark advances.

BEGIN;

-- Queue repayments that could not be inserted because their loan row was not yet
-- present in Seeds Metrics. Sync jobs drain this queue before processing new work.
CREATE TABLE IF NOT EXISTS repayment_sync_retry_queue (
    repayment_id VARCHAR(50) PRIMARY KEY,
    loan_id VARCHAR(50) NOT NULL,
    django_updated_at TIMESTAMPTZ,
    first_seen_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    last_attempt_at TIMESTAMPTZ,
    attempts INTEGER NOT NULL DEFAULT 0,
    last_error TEXT,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_repayment_sync_retry_queue_loan_id
    ON repayment_sync_retry_queue (loan_id);
CREATE INDEX IF NOT EXISTS idx_repayment_sync_retry_queue_first_seen
    ON repayment_sync_retry_queue (first_seen_at);

COMMENT ON TABLE repayment_sync_retry_queue IS
'Repayments that could not be synced yet because the referenced loan was not present; drained by repayment sync jobs.';

-- Patch the repayment-trigger function in-place so this migration stays small and
-- preserves the currently deployed function body from prior migrations.
DO $$
DECLARE
    funcdef TEXT;
BEGIN
    SELECT pg_get_functiondef('update_loan_computed_fields()'::regprocedure)
    INTO funcdef;

    -- Keep repayment_delay_rate compatible with the loans.repayment_delay_rate
    -- column precision and avoid numeric overflow during trigger execution.
    funcdef := REPLACE(funcdef,
        'v_repayment_delay_rate DECIMAL(5, 2);',
        'v_repayment_delay_rate DECIMAL(8, 2);'
    );
    funcdef := REPLACE(funcdef,
        'v_repayment_delay_rate DECIMAL(5,2);',
        'v_repayment_delay_rate DECIMAL(8, 2);'
    );

    -- AFTER DELETE triggers have OLD but not NEW. Use whichever record exists.
    funcdef := REPLACE(funcdef,
        'v_loan_id := NEW.loan_id;',
        'v_loan_id := COALESCE(NEW.loan_id, OLD.loan_id);'
    );

    -- Return OLD for DELETE for completeness. AFTER-trigger return values are
    -- ignored, but this makes the function correct for all row-level operations.
    funcdef := REPLACE(funcdef,
        'RETURN NEW;',
        E'IF TG_OP = ''DELETE'' THEN\n        RETURN OLD;\n    END IF;\n\n    RETURN NEW;'
    );

    EXECUTE funcdef;
END $$;

-- Ensure the canonical computed-fields trigger includes DELETE events.
DROP TRIGGER IF EXISTS trg_update_loan_computed_fields ON repayments;
CREATE TRIGGER trg_update_loan_computed_fields
AFTER INSERT OR UPDATE OR DELETE ON repayments
FOR EACH ROW EXECUTE FUNCTION update_loan_computed_fields();

COMMIT;
