#!/usr/bin/env python3
"""
Sync loan_type and verification_status from Django to SeedsMetrics database
"""

import psycopg2
from psycopg2.extras import RealDictCursor
import sys

# Django database connection
DJANGO_DB = {
    'host': '164.90.155.2',
    'port': 5432,
    'dbname': 'savings',
    'user': 'metricsuser',
    'password': 'EiRXo6IfeHQuM3wcbZ67$LzwmVKCXhpUhWg',
    'sslmode': 'require'
}

# SeedsMetrics database connection
SEEDSMETRICS_DB = {
    'host': 'generaldb-do-user-9489371-0.k.db.ondigitalocean.com',
    'port': 25060,
    'dbname': 'seedsmetrics',
    'user': 'seedsuser',
    'password': '@seedsuser2020',
    'sslmode': 'require'
}

def main():
    print("🚀 Starting loan_type and verification_status sync...")

    # Connect to Django database
    print("📡 Connecting to Django database...")
    django_conn = psycopg2.connect(**DJANGO_DB)
    django_cur = django_conn.cursor(cursor_factory=RealDictCursor)

    # Connect to SeedsMetrics database
    print("📡 Connecting to SeedsMetrics database...")
    metrics_conn = psycopg2.connect(**SEEDSMETRICS_DB)
    metrics_cur = metrics_conn.cursor()

    try:
        # Fetch loan_type and verification_stage from Django
        print("📊 Fetching loan data from Django...")
        django_cur.execute("""
            SELECT
                id::VARCHAR(50) as loan_id,
                loan_type,
                verification_stage
            FROM loans_ajoloan
            WHERE is_disbursed = TRUE
        """)

        django_loans = django_cur.fetchall()
        print(f"✅ Fetched {len(django_loans)} loans from Django")

        # Update SeedsMetrics database using batch updates
        print("🔄 Updating SeedsMetrics database...")
        updated_count = 0
        batch_size = 1000

        for i in range(0, len(django_loans), batch_size):
            batch = django_loans[i:i+batch_size]

            # Build batch update using CASE statements
            loan_ids = [loan['loan_id'] for loan in batch]

            # Create temporary table for batch update
            metrics_cur.execute("""
                CREATE TEMP TABLE temp_loan_updates (
                    loan_id VARCHAR(50),
                    loan_type VARCHAR(255),
                    verification_status VARCHAR(255)
                ) ON COMMIT DROP
            """)

            # Insert batch data
            from psycopg2.extras import execute_values
            execute_values(
                metrics_cur,
                """
                INSERT INTO temp_loan_updates (loan_id, loan_type, verification_status)
                VALUES %s
                """,
                [(loan['loan_id'], loan['loan_type'], loan['verification_stage']) for loan in batch]
            )

            # Perform batch update
            metrics_cur.execute("""
                UPDATE loans l
                SET
                    loan_type = t.loan_type,
                    verification_status = t.verification_status,
                    updated_at = NOW()
                FROM temp_loan_updates t
                WHERE l.loan_id = t.loan_id
                    AND (
                        l.loan_type IS DISTINCT FROM t.loan_type
                        OR l.verification_status IS DISTINCT FROM t.verification_status
                    )
            """)

            updated_count += metrics_cur.rowcount

            # Commit batch
            metrics_conn.commit()

            if (i + batch_size) % 5000 == 0 or i + batch_size >= len(django_loans):
                print(f"   Processed {min(i + batch_size, len(django_loans))}/{len(django_loans)} loans...")

        print(f"✅ Updated {updated_count} loans")

        # Show statistics
        print("\n📊 Statistics:")

        # Loan types
        metrics_cur.execute("""
            SELECT DISTINCT loan_type, COUNT(*) as count
            FROM loans
            WHERE loan_type IS NOT NULL AND loan_type != ''
            GROUP BY loan_type
            ORDER BY loan_type
        """)

        loan_types = metrics_cur.fetchall()
        print(f"\nLoan Types ({len(loan_types)} types):")
        for row in loan_types:
            print(f"  - {row[0]}: {row[1]} loans")

        # Verification statuses
        metrics_cur.execute("""
            SELECT DISTINCT verification_status, COUNT(*) as count
            FROM loans
            WHERE verification_status IS NOT NULL AND verification_status != ''
            GROUP BY verification_status
            ORDER BY verification_status
        """)

        verification_statuses = metrics_cur.fetchall()
        print(f"\nVerification Statuses ({len(verification_statuses)} statuses):")
        for row in verification_statuses:
            print(f"  - {row[0]}: {row[1]} loans")

        print("\n✅ Sync completed successfully!")

    except Exception as e:
        print(f"❌ Error: {e}")
        import traceback
        traceback.print_exc()
        sys.exit(1)

    finally:
        django_cur.close()
        django_conn.close()
        metrics_cur.close()
        metrics_conn.close()

if __name__ == '__main__':
    main()

