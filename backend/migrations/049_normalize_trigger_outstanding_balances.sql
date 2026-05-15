-- Migration 049: Keep trigger-computed outstanding balances normalized
--
-- Repayment-trigger activity must follow the same business rule as the full
-- recalculation path: contractual outstanding is non-negative and equals
-- repayment_amount - total_repayments. Also cap actual_outstanding so it never
-- exceeds the contractual balance after repayments are inserted/updated/deleted.

BEGIN;

DO $$
DECLARE
    funcdef TEXT;
BEGIN
    SELECT pg_get_functiondef('update_loan_computed_fields()'::regprocedure)
    INTO funcdef;

    -- Preserve the safety fixes from migration 048 if this is run on an
    -- environment where they have not already been applied.
    funcdef := REPLACE(funcdef,
        'v_repayment_delay_rate DECIMAL(5, 2);',
        'v_repayment_delay_rate DECIMAL(8, 2);'
    );
    funcdef := REPLACE(funcdef,
        'v_repayment_delay_rate DECIMAL(5,2);',
        'v_repayment_delay_rate DECIMAL(8, 2);'
    );
    funcdef := REPLACE(funcdef,
        'v_loan_id := NEW.loan_id;',
        'v_loan_id := COALESCE(NEW.loan_id, OLD.loan_id);'
    );

    -- Use the contractual repayment balance instead of component balances,
    -- which can go negative or NULL when source component fields are incomplete.
    funcdef := REPLACE(funcdef,
        'v_total_outstanding := v_principal_outstanding + v_interest_outstanding + v_fees_outstanding;',
        'v_total_outstanding := GREATEST(0, COALESCE(v_repayment_amount, 0) - COALESCE(v_total_repayments, 0));'
    );

    -- Idempotently cap actual_outstanding in the repayment trigger as well.
    IF POSITION('actual_outstanding = LEAST(COALESCE(actual_outstanding, 0), v_total_outstanding)' IN funcdef) = 0 THEN
        funcdef := REPLACE(funcdef,
            'total_outstanding = v_total_outstanding,',
            'total_outstanding = v_total_outstanding,
        actual_outstanding = LEAST(COALESCE(actual_outstanding, 0), v_total_outstanding),'
        );
    END IF;

    IF POSITION('RETURN OLD' IN funcdef) = 0 THEN
        funcdef := REPLACE(funcdef,
            'RETURN NEW;',
            E'IF TG_OP = ''DELETE'' THEN\n        RETURN OLD;\n    END IF;\n\n    RETURN NEW;'
        );
    END IF;

    EXECUTE funcdef;
END $$;

DROP TRIGGER IF EXISTS trg_update_loan_computed_fields ON repayments;
CREATE TRIGGER trg_update_loan_computed_fields
AFTER INSERT OR UPDATE OR DELETE ON repayments
FOR EACH ROW EXECUTE FUNCTION update_loan_computed_fields();

-- Normalize rows already affected by older trigger executions.
UPDATE loans
SET
    total_outstanding = GREATEST(0, COALESCE(repayment_amount, 0) - COALESCE(total_repayments, 0)),
    actual_outstanding = LEAST(
        COALESCE(actual_outstanding, 0),
        GREATEST(0, COALESCE(repayment_amount, 0) - COALESCE(total_repayments, 0))
    )
WHERE
    total_outstanding IS DISTINCT FROM GREATEST(0, COALESCE(repayment_amount, 0) - COALESCE(total_repayments, 0))
    OR COALESCE(actual_outstanding, 0) > GREATEST(0, COALESCE(repayment_amount, 0) - COALESCE(total_repayments, 0));

COMMIT;