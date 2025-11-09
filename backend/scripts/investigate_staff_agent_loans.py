#!/usr/bin/env python3
"""
Script to investigate and fix data quality issues with STAFF_AGENT loans.

This script:
1. Counts loans with user_type = 'STAFF_AGENT' and NULL officer data
2. Investigates the root cause by checking Django source database
3. Syncs missing officer data from Django to Seeds Metrics
4. Provides before/after verification
"""

import psycopg2
from psycopg2 import OperationalError
import sys
from datetime import datetime

# Database connection configurations
SEEDS_METRICS_DB = {
    'host': 'private-generaldb-do-user-9489371-0.k.db.ondigitalocean.com',
    'port': '25060',
    'dbname': 'seedsmetrics',
    'user': 'seedsuser',
    'password': '@seedsuser2020',
    'sslmode': 'require'
}

DJANGO_DB = {
    'host': '164.90.155.2',
    'port': '5432',
    'dbname': 'savings',
    'user': 'metricsuser',
    'password': 'EiRXo6IfeHQuM3wcbZ67$LzwmVKCXhpUhWg',
    'sslmode': 'require'
}

def create_connection(db_config, db_name):
    """Create database connection."""
    try:
        conn = psycopg2.connect(**db_config)
        print(f"✅ Connected to {db_name} database")
        return conn
    except OperationalError as e:
        print(f"❌ Failed to connect to {db_name} database: {e}")
        sys.exit(1)

def run_query(conn, query, params=None, fetch=True):
    """Execute a query and return results."""
    try:
        cursor = conn.cursor()
        cursor.execute(query, params)
        if fetch:
            results = cursor.fetchall()
            cursor.close()
            return results
        else:
            conn.commit()
            rowcount = cursor.rowcount
            cursor.close()
            return rowcount
    except Exception as e:
        print(f"❌ Query error: {e}")
        print(f"Query: {query}")
        conn.rollback()
        raise

def print_section(title):
    """Print a formatted section header."""
    print(f"\n{'='*80}")
    print(f"  {title}")
    print(f"{'='*80}\n")

def step1_count_staff_agent_loans(seeds_conn):
    """Step 1: Count loans with NULL or empty officer data."""
    print_section("STEP 1: Count Loans with NULL Officer Data")

    # Query 1: Count total loans
    query1 = """
        SELECT COUNT(*)
        FROM loans
    """
    result1 = run_query(seeds_conn, query1)
    total_loans = result1[0][0]
    print(f"📊 Total loans in database: {total_loans:,}")

    # Query 2: Count loans with NULL or empty officer_id
    query2 = """
        SELECT COUNT(*)
        FROM loans
        WHERE officer_id IS NULL OR officer_id = ''
    """
    result2 = run_query(seeds_conn, query2)
    null_officer_id_count = result2[0][0]
    print(f"📊 Loans with officer_id = NULL or empty: {null_officer_id_count:,}")

    # Query 3: Count loans with NULL or empty officer_name
    query3 = """
        SELECT COUNT(*)
        FROM loans
        WHERE officer_name IS NULL OR officer_name = ''
    """
    result3 = run_query(seeds_conn, query3)
    null_officer_name_count = result3[0][0]
    print(f"📊 Loans with officer_name = NULL or empty: {null_officer_name_count:,}")

    # Query 4: Count loans with NULL or empty officer_email (note: officer_email column doesn't exist in loans table)
    # We'll check this in the officers table instead
    query4 = """
        SELECT COUNT(DISTINCT l.loan_id)
        FROM loans l
        LEFT JOIN officers o ON l.officer_id = o.officer_id
        WHERE o.officer_email IS NULL OR o.officer_email = ''
    """
    result4 = run_query(seeds_conn, query4)
    null_officer_email_count = result4[0][0]
    print(f"📊 Loans where officer has NULL or empty email: {null_officer_email_count:,}")

    # Query 5: Count loans with STAFF_AGENT officers
    query5 = """
        SELECT COUNT(DISTINCT l.loan_id)
        FROM loans l
        LEFT JOIN officers o ON l.officer_id = o.officer_id
        WHERE o.user_type = 'STAFF_AGENT'
    """
    result5 = run_query(seeds_conn, query5)
    staff_agent_count = result5[0][0]
    print(f"📊 Loans with STAFF_AGENT officers: {staff_agent_count:,}")

    # Query 6: Count loans with NULL officer data (any of the fields)
    query6 = """
        SELECT COUNT(*)
        FROM loans
        WHERE officer_id IS NULL OR officer_id = ''
           OR officer_name IS NULL OR officer_name = ''
    """
    result6 = run_query(seeds_conn, query6)
    loans_with_null_officer = result6[0][0]
    print(f"📊 Loans with ANY NULL officer data: {loans_with_null_officer:,}")

    # Query 7: Sample of loans with NULL officer data
    query7 = """
        SELECT loan_id, customer_name, loan_amount, disbursement_date,
               officer_id, officer_name, officer_phone
        FROM loans
        WHERE officer_id IS NULL OR officer_id = ''
           OR officer_name IS NULL OR officer_name = ''
        LIMIT 10
    """
    result7 = run_query(seeds_conn, query7)
    print(f"\n📋 Sample of loans with NULL officer data (first 10):")
    print(f"{'Loan ID':<15} {'Customer':<25} {'Amount':<12} {'Disbursement':<12} {'Officer ID':<12} {'Officer Name':<20} {'Officer Phone':<15}")
    print("-" * 130)
    for row in result7:
        loan_id, customer_name, loan_amount, disb_date, officer_id, officer_name, officer_phone = row
        print(f"{loan_id:<15} {customer_name[:24]:<25} {loan_amount:<12.2f} {str(disb_date):<12} {str(officer_id):<12} {str(officer_name)[:19]:<20} {str(officer_phone)[:14]:<15}")

    return {
        'total_loans': total_loans,
        'null_officer_id_count': null_officer_id_count,
        'null_officer_name_count': null_officer_name_count,
        'null_officer_email_count': null_officer_email_count,
        'staff_agent_count': staff_agent_count,
        'loans_with_null_officer': loans_with_null_officer
    }

def step2_investigate_django_source(django_conn, seeds_conn):
    """Step 2: Check if officer data exists in Django database."""
    print_section("STEP 2: Investigate Django Source Database")

    # Get sample loan IDs from Seeds Metrics with NULL officer data
    query_sample = """
        SELECT loan_id
        FROM loans
        WHERE officer_id IS NULL OR officer_id = ''
           OR officer_name IS NULL OR officer_name = ''
        LIMIT 5
    """
    sample_loans = run_query(seeds_conn, query_sample)
    sample_loan_ids = [row[0] for row in sample_loans]

    print(f"🔍 Checking {len(sample_loan_ids)} sample loan IDs in Django database:")
    print(f"Sample Loan IDs: {', '.join(sample_loan_ids)}")

    # Check Django database for these loans
    for loan_id in sample_loan_ids:
        query_django = """
            SELECT
                l.id::VARCHAR(50) as loan_id,
                l.agent_id::VARCHAR(50) as officer_id,
                COALESCE(u.username, u.email) as officer_name,
                u.email as officer_email,
                u.user_phone as officer_phone,
                COALESCE(u.user_branch, 'Unknown') as branch
            FROM loans_ajoloan l
            LEFT JOIN accounts_customuser u ON l.agent_id = u.id
            WHERE l.id::VARCHAR(50) = %s
        """
        result = run_query(django_conn, query_django, (loan_id,))

        if result:
            loan_id_dj, officer_id, officer_name, officer_email, officer_phone, branch = result[0]
            print(f"\n  Loan ID: {loan_id_dj}")
            print(f"    Officer ID: {officer_id}")
            print(f"    Officer Name: {officer_name}")
            print(f"    Officer Email: {officer_email}")
            print(f"    Officer Phone: {officer_phone}")
            print(f"    Branch: {branch}")
        else:
            print(f"\n  Loan ID: {loan_id} - NOT FOUND in Django database")

    # Count total loans in Django with officer data
    query_django_count = """
        SELECT COUNT(*)
        FROM loans_ajoloan l
        LEFT JOIN accounts_customuser u ON l.agent_id = u.id
        WHERE l.is_disbursed = TRUE
        AND l.agent_id IS NOT NULL
        AND u.id IS NOT NULL
    """
    result_count = run_query(django_conn, query_django_count)
    django_loans_with_officer = result_count[0][0]
    print(f"\n📊 Total loans in Django with officer data: {django_loans_with_officer:,}")

def step3_sync_officer_data(django_conn, seeds_conn):
    """Step 3: Sync missing officer data from Django to Seeds Metrics."""
    print_section("STEP 3: Sync Missing Officer Data")

    print("🔄 Fetching officer data from Django database...")

    # Get all loans from Django with officer information
    query_django_loans = """
        SELECT
            l.id::VARCHAR(50) as loan_id,
            l.agent_id::VARCHAR(50) as officer_id,
            COALESCE(u.username, u.email) as officer_name,
            u.email as officer_email,
            u.user_phone as officer_phone,
            COALESCE(u.user_branch, 'Unknown') as branch
        FROM loans_ajoloan l
        LEFT JOIN accounts_customuser u ON l.agent_id = u.id
        WHERE l.is_disbursed = TRUE
        AND l.agent_id IS NOT NULL
        AND u.id IS NOT NULL
    """
    django_loans = run_query(django_conn, query_django_loans)
    print(f"✅ Fetched {len(django_loans):,} loans with officer data from Django")

    # Update Seeds Metrics database
    print("\n🔄 Updating Seeds Metrics database...")

    # Note: loans table doesn't have officer_email column - that's in the officers table
    # We'll update officer_id, officer_name, officer_phone, and branch in the loans table
    update_query = """
        UPDATE loans
        SET
            officer_id = %s,
            officer_name = %s,
            officer_phone = %s,
            branch = %s,
            updated_at = NOW()
        WHERE loan_id = %s
        AND (officer_id IS NULL OR officer_id = '' OR officer_name IS NULL OR officer_name = '')
    """

    updated_count = 0
    batch_size = 1000
    total_loans = len(django_loans)

    cursor = seeds_conn.cursor()

    for i, loan in enumerate(django_loans, 1):
        loan_id, officer_id, officer_name, officer_email, officer_phone, branch = loan

        try:
            cursor.execute(update_query, (officer_id, officer_name, officer_phone, branch, loan_id))
            if cursor.rowcount > 0:
                updated_count += 1

            # Commit in batches
            if i % batch_size == 0:
                seeds_conn.commit()
                print(f"  Progress: {i:,}/{total_loans:,} loans processed ({updated_count:,} updated)")
        except Exception as e:
            print(f"  ⚠️  Error updating loan {loan_id}: {e}")
            seeds_conn.rollback()
            continue

    # Final commit
    seeds_conn.commit()
    cursor.close()

    print(f"\n✅ Update complete!")
    print(f"📊 Total loans updated: {updated_count:,}")

    return updated_count

def step4_verify_sync(seeds_conn):
    """Step 4: Verify the sync by re-querying Seeds Metrics."""
    print_section("STEP 4: Verification - After Sync")

    # Re-run the same queries from Step 1
    query1 = """
        SELECT COUNT(*)
        FROM loans
    """
    result1 = run_query(seeds_conn, query1)
    total_loans = result1[0][0]
    print(f"📊 Total loans in database: {total_loans:,}")

    query2 = """
        SELECT COUNT(*)
        FROM loans
        WHERE officer_id IS NULL OR officer_id = ''
    """
    result2 = run_query(seeds_conn, query2)
    null_officer_id_count = result2[0][0]
    print(f"📊 Loans with officer_id = NULL or empty: {null_officer_id_count:,}")

    query3 = """
        SELECT COUNT(*)
        FROM loans
        WHERE officer_name IS NULL OR officer_name = ''
    """
    result3 = run_query(seeds_conn, query3)
    null_officer_name_count = result3[0][0]
    print(f"📊 Loans with officer_name = NULL or empty: {null_officer_name_count:,}")

    query4 = """
        SELECT COUNT(DISTINCT l.loan_id)
        FROM loans l
        LEFT JOIN officers o ON l.officer_id = o.officer_id
        WHERE o.officer_email IS NULL OR o.officer_email = ''
    """
    result4 = run_query(seeds_conn, query4)
    null_officer_email_count = result4[0][0]
    print(f"📊 Loans where officer has NULL or empty email: {null_officer_email_count:,}")

    query5 = """
        SELECT COUNT(*)
        FROM loans
        WHERE officer_id IS NULL OR officer_id = ''
           OR officer_name IS NULL OR officer_name = ''
    """
    result5 = run_query(seeds_conn, query5)
    loans_with_null_officer = result5[0][0]
    print(f"📊 Loans with ANY NULL officer data: {loans_with_null_officer:,}")

    return {
        'total_loans': total_loans,
        'null_officer_id_count': null_officer_id_count,
        'null_officer_name_count': null_officer_name_count,
        'null_officer_email_count': null_officer_email_count,
        'loans_with_null_officer': loans_with_null_officer
    }

def main():
    """Main execution function."""
    print(f"\n{'='*80}")
    print(f"  STAFF_AGENT Loans Data Quality Investigation & Fix")
    print(f"  Started: {datetime.now().strftime('%Y-%m-%d %H:%M:%S')}")
    print(f"{'='*80}\n")

    # Connect to databases
    seeds_conn = create_connection(SEEDS_METRICS_DB, "Seeds Metrics")
    django_conn = create_connection(DJANGO_DB, "Django")

    try:
        # Step 1: Count and analyze
        before_counts = step1_count_staff_agent_loans(seeds_conn)

        # Step 2: Investigate Django source
        step2_investigate_django_source(django_conn, seeds_conn)

        # Step 3: Sync officer data
        updated_count = step3_sync_officer_data(django_conn, seeds_conn)

        # Step 4: Verify sync
        after_counts = step4_verify_sync(seeds_conn)

        # Summary
        print_section("SUMMARY")
        print("BEFORE SYNC:")
        print(f"  - Total loans: {before_counts['total_loans']:,}")
        print(f"  - STAFF_AGENT loans: {before_counts['staff_agent_count']:,}")
        print(f"  - Loans with NULL officer_id: {before_counts['null_officer_id_count']:,}")
        print(f"  - Loans with NULL officer_name: {before_counts['null_officer_name_count']:,}")
        print(f"  - Loans with NULL officer_email: {before_counts['null_officer_email_count']:,}")
        print(f"  - Loans with ANY NULL officer data: {before_counts['loans_with_null_officer']:,}")

        print("\nAFTER SYNC:")
        print(f"  - Total loans: {after_counts['total_loans']:,}")
        print(f"  - Loans with NULL officer_id: {after_counts['null_officer_id_count']:,}")
        print(f"  - Loans with NULL officer_name: {after_counts['null_officer_name_count']:,}")
        print(f"  - Loans with NULL officer_email: {after_counts['null_officer_email_count']:,}")
        print(f"  - Loans with ANY NULL officer data: {after_counts['loans_with_null_officer']:,}")

        print(f"\n✅ Total loans updated in loans table: {updated_count:,}")
        print(f"✅ Completed: {datetime.now().strftime('%Y-%m-%d %H:%M:%S')}")

    except Exception as e:
        print(f"\n❌ Error: {e}")
        import traceback
        traceback.print_exc()
    finally:
        # Close connections
        seeds_conn.close()
        django_conn.close()
        print("\n🔒 Database connections closed")

if __name__ == "__main__":
    main()

