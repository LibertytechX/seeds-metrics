#!/usr/bin/env python3
"""
Sync Officer Emails from Django to Seeds Metrics

This script syncs officer email addresses from the Django database to the Seeds Metrics
officers table. The issue is that all officers in Seeds Metrics have NULL or empty email
addresses, but the Django database has the correct email information.

Author: Seeds Metrics Team
Date: 2025-11-06
"""

import psycopg2
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

def connect_db(config):
    """Connect to a PostgreSQL database."""
    try:
        conn = psycopg2.connect(
            host=config['host'],
            port=config['port'],
            dbname=config['dbname'],
            user=config['user'],
            password=config['password'],
            sslmode=config['sslmode']
        )
        return conn
    except Exception as e:
        print(f"❌ Database connection error: {e}")
        raise

def run_query(conn, query, params=None):
    """Execute a query and return results."""
    try:
        cursor = conn.cursor()
        cursor.execute(query, params)
        if cursor.description:  # SELECT query
            result = cursor.fetchall()
            cursor.close()
            return result
        else:  # INSERT/UPDATE/DELETE query
            conn.commit()
            cursor.close()
            return None
    except Exception as e:
        print(f"❌ Query error: {e}")
        print(f"Query: {query}")
        raise

def print_section(title):
    """Print a formatted section header."""
    print(f"\n{'='*80}")
    print(f"  {title}")
    print(f"{'='*80}\n")

def step1_count_officers_without_email(seeds_conn):
    """Step 1: Count officers with NULL or empty email in Seeds Metrics."""
    print_section("STEP 1: Count Officers Without Email")

    # Query 1: Total officers
    query1 = """
        SELECT COUNT(*)
        FROM officers
    """
    result1 = run_query(seeds_conn, query1)
    total_officers = result1[0][0]
    print(f"📊 Total officers in database: {total_officers:,}")

    # Query 2: Officers with NULL or empty email
    query2 = """
        SELECT COUNT(*)
        FROM officers
        WHERE officer_email IS NULL OR officer_email = ''
    """
    result2 = run_query(seeds_conn, query2)
    null_email_count = result2[0][0]
    print(f"📊 Officers with NULL or empty email: {null_email_count:,}")

    # Query 3: Officers with STAFF_AGENT user_type
    query3 = """
        SELECT COUNT(*)
        FROM officers
        WHERE user_type = 'STAFF_AGENT'
    """
    result3 = run_query(seeds_conn, query3)
    staff_agent_count = result3[0][0]
    print(f"📊 Officers with user_type = 'STAFF_AGENT': {staff_agent_count:,}")

    # Query 4: STAFF_AGENT officers with NULL email
    query4 = """
        SELECT COUNT(*)
        FROM officers
        WHERE user_type = 'STAFF_AGENT'
        AND (officer_email IS NULL OR officer_email = '')
    """
    result4 = run_query(seeds_conn, query4)
    staff_agent_null_email = result4[0][0]
    print(f"📊 STAFF_AGENT officers with NULL email: {staff_agent_null_email:,}")

    # Query 5: Sample of officers without email
    query5 = """
        SELECT officer_id, officer_name, officer_phone, branch, user_type
        FROM officers
        WHERE officer_email IS NULL OR officer_email = ''
        LIMIT 10
    """
    result5 = run_query(seeds_conn, query5)
    print(f"\n📋 Sample of officers without email (first 10):")
    print(f"{'Officer ID':<12} {'Officer Name':<30} {'Phone':<15} {'Branch':<20} {'User Type':<15}")
    print("-" * 100)
    for row in result5:
        officer_id, officer_name, officer_phone, branch, user_type = row
        print(f"{officer_id:<12} {str(officer_name)[:29]:<30} {str(officer_phone)[:14]:<15} {str(branch)[:19]:<20} {str(user_type)[:14]:<15}")

    return {
        'total_officers': total_officers,
        'null_email_count': null_email_count,
        'staff_agent_count': staff_agent_count,
        'staff_agent_null_email': staff_agent_null_email
    }

def step2_fetch_emails_from_django(django_conn):
    """Step 2: Fetch officer emails from Django database."""
    print_section("STEP 2: Fetch Officer Emails from Django")

    query = """
        SELECT
            u.id::VARCHAR(50) as officer_id,
            COALESCE(u.username, u.email) as officer_name,
            u.email as officer_email,
            u.user_phone as officer_phone
        FROM accounts_customuser u
        WHERE u.email IS NOT NULL
        AND u.email != ''
    """
    result = run_query(django_conn, query)
    print(f"✅ Fetched {len(result):,} officers with email addresses from Django")

    # Show sample
    print(f"\n📋 Sample of officers from Django (first 10):")
    print(f"{'Officer ID':<12} {'Officer Name':<30} {'Email':<35} {'Phone':<15}")
    print("-" * 100)
    for i, row in enumerate(result[:10]):
        officer_id, officer_name, officer_email, officer_phone = row
        print(f"{officer_id:<12} {str(officer_name)[:29]:<30} {str(officer_email)[:34]:<35} {str(officer_phone)[:14]:<15}")

    return result

def step3_sync_emails(django_officers, seeds_conn):
    """Step 3: Sync officer emails from Django to Seeds Metrics."""
    print_section("STEP 3: Sync Officer Emails")

    print("🔄 Updating Seeds Metrics officers table...")

    update_query = """
        UPDATE officers
        SET
            officer_email = %s,
            updated_at = NOW()
        WHERE officer_id = %s
        AND (officer_email IS NULL OR officer_email = '')
    """

    updated_count = 0
    batch_size = 1000
    total_officers = len(django_officers)

    cursor = seeds_conn.cursor()

    for i, officer in enumerate(django_officers, 1):
        officer_id, officer_name, officer_email, officer_phone = officer

        try:
            cursor.execute(update_query, (officer_email, officer_id))
            if cursor.rowcount > 0:
                updated_count += 1

            # Commit in batches
            if i % batch_size == 0:
                seeds_conn.commit()
                print(f"  Progress: {i:,}/{total_officers:,} officers processed ({updated_count:,} updated)")
        except Exception as e:
            print(f"  ⚠️  Error updating officer {officer_id}: {e}")
            seeds_conn.rollback()
            continue

    # Final commit
    seeds_conn.commit()
    cursor.close()

    print(f"\n✅ Update complete!")
    print(f"📊 Total officers updated: {updated_count:,}")

    return updated_count

def step4_verify_sync(seeds_conn):
    """Step 4: Verify the sync by re-querying Seeds Metrics."""
    print_section("STEP 4: Verification - After Sync")

    # Query 1: Total officers
    query1 = """
        SELECT COUNT(*)
        FROM officers
    """
    result1 = run_query(seeds_conn, query1)
    total_officers = result1[0][0]
    print(f"📊 Total officers in database: {total_officers:,}")

    # Query 2: Officers with NULL or empty email
    query2 = """
        SELECT COUNT(*)
        FROM officers
        WHERE officer_email IS NULL OR officer_email = ''
    """
    result2 = run_query(seeds_conn, query2)
    null_email_count = result2[0][0]
    print(f"📊 Officers with NULL or empty email: {null_email_count:,}")

    # Query 3: Officers with valid email
    query3 = """
        SELECT COUNT(*)
        FROM officers
        WHERE officer_email IS NOT NULL AND officer_email != ''
    """
    result3 = run_query(seeds_conn, query3)
    valid_email_count = result3[0][0]
    print(f"📊 Officers with valid email: {valid_email_count:,}")

    # Query 4: STAFF_AGENT officers with NULL email
    query4 = """
        SELECT COUNT(*)
        FROM officers
        WHERE user_type = 'STAFF_AGENT'
        AND (officer_email IS NULL OR officer_email = '')
    """
    result4 = run_query(seeds_conn, query4)
    staff_agent_null_email = result4[0][0]
    print(f"📊 STAFF_AGENT officers with NULL email: {staff_agent_null_email:,}")

    return {
        'total_officers': total_officers,
        'null_email_count': null_email_count,
        'valid_email_count': valid_email_count,
        'staff_agent_null_email': staff_agent_null_email
    }

def main():
    """Main execution function."""
    print(f"\n{'='*80}")
    print(f"  Officer Email Sync from Django to Seeds Metrics")
    print(f"  Started: {datetime.now().strftime('%Y-%m-%d %H:%M:%S')}")
    print(f"{'='*80}\n")

    try:
        # Connect to databases
        seeds_conn = connect_db(SEEDS_METRICS_DB)
        print("✅ Connected to Seeds Metrics database")

        django_conn = connect_db(DJANGO_DB)
        print("✅ Connected to Django database")

        # Step 1: Count officers without email
        before_counts = step1_count_officers_without_email(seeds_conn)

        # Step 2: Fetch emails from Django
        django_officers = step2_fetch_emails_from_django(django_conn)

        # Step 3: Sync emails
        updated_count = step3_sync_emails(django_officers, seeds_conn)

        # Step 4: Verify sync
        after_counts = step4_verify_sync(seeds_conn)

        # Summary
        print_section("SUMMARY")
        print("BEFORE SYNC:")
        print(f"  - Total officers: {before_counts['total_officers']:,}")
        print(f"  - Officers with NULL email: {before_counts['null_email_count']:,}")
        print(f"  - STAFF_AGENT officers: {before_counts['staff_agent_count']:,}")
        print(f"  - STAFF_AGENT with NULL email: {before_counts['staff_agent_null_email']:,}")

        print("\nAFTER SYNC:")
        print(f"  - Total officers: {after_counts['total_officers']:,}")
        print(f"  - Officers with NULL email: {after_counts['null_email_count']:,}")
        print(f"  - Officers with valid email: {after_counts['valid_email_count']:,}")
        print(f"  - STAFF_AGENT with NULL email: {after_counts['staff_agent_null_email']:,}")

        print(f"\n✅ Total officers updated: {updated_count:,}")
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

