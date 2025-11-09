#!/usr/bin/env python3
"""
Find Missing Loans Between Django and Seeds Metrics

This script identifies which loans exist in the Django source database but are missing
from the Seeds Metrics database.

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

def main():
    """Main execution function."""
    print(f"\n{'='*80}")
    print(f"  Find Missing Loans Between Django and Seeds Metrics")
    print(f"  Started: {datetime.now().strftime('%Y-%m-%d %H:%M:%S')}")
    print(f"{'='*80}\n")

    try:
        # Connect to databases
        seeds_conn = connect_db(SEEDS_METRICS_DB)
        print("✅ Connected to Seeds Metrics database")

        django_conn = connect_db(DJANGO_DB)
        print("✅ Connected to Django database")

        # Get all loan IDs from Seeds Metrics
        print("\n🔍 Fetching loan IDs from Seeds Metrics...")
        seeds_cursor = seeds_conn.cursor()
        seeds_cursor.execute("SELECT loan_id FROM loans ORDER BY loan_id")
        seeds_loan_ids = set(row[0] for row in seeds_cursor.fetchall())
        seeds_cursor.close()
        print(f"✅ Found {len(seeds_loan_ids):,} loans in Seeds Metrics")

        # Get all disbursed loan IDs from Django
        print("\n🔍 Fetching disbursed loan IDs from Django...")
        django_cursor = django_conn.cursor()
        django_cursor.execute("SELECT id::VARCHAR(50) FROM loans_ajoloan WHERE is_disbursed = TRUE ORDER BY id")
        django_loan_ids = set(row[0] for row in django_cursor.fetchall())
        django_cursor.close()
        print(f"✅ Found {len(django_loan_ids):,} disbursed loans in Django")

        # Find missing loans
        missing_loan_ids = django_loan_ids - seeds_loan_ids
        print(f"\n📊 Difference: {len(missing_loan_ids):,} loans in Django but NOT in Seeds Metrics")

        if missing_loan_ids:
            # Get details of missing loans
            print(f"\n🔍 Fetching details of missing loans...")
            placeholders = ','.join(['%s'] * len(list(missing_loan_ids)[:100]))  # Limit to first 100
            query = f"""
                SELECT
                    l.id::VARCHAR(50) as loan_id,
                    l.date_disbursed,
                    l.amount,
                    l.status,
                    l.agent_id::VARCHAR(50) as officer_id,
                    COALESCE(u.username, u.email) as officer_name,
                    COALESCE(u.user_branch, 'Unknown') as branch
                FROM loans_ajoloan l
                LEFT JOIN accounts_customuser u ON l.agent_id = u.id
                WHERE l.id::VARCHAR(50) IN ({placeholders})
                ORDER BY l.date_disbursed DESC
                LIMIT 100
            """
            django_cursor = django_conn.cursor()
            django_cursor.execute(query, tuple(list(missing_loan_ids)[:100]))
            missing_loans = django_cursor.fetchall()
            django_cursor.close()

            print(f"\n📋 Sample of missing loans (showing up to 100 most recent):")
            print(f"{'Loan ID':<12} {'Disbursed':<12} {'Amount':<15} {'Status':<12} {'Officer ID':<12} {'Officer Name':<30} {'Branch':<20}")
            print("-" * 130)
            for row in missing_loans[:20]:  # Show first 20
                loan_id, date_disbursed, amount, status, officer_id, officer_name, branch = row
                print(f"{loan_id:<12} {str(date_disbursed)[:10]:<12} {amount:<15.2f} {status:<12} {str(officer_id):<12} {str(officer_name)[:29]:<30} {str(branch)[:19]:<20}")

            if len(missing_loans) > 20:
                print(f"... and {len(missing_loans) - 20} more")

            # Analyze by date
            print(f"\n📊 Analysis by disbursement date:")
            query_by_date = f"""
                SELECT
                    DATE(l.date_disbursed) as disb_date,
                    COUNT(*) as loan_count
                FROM loans_ajoloan l
                WHERE l.id::VARCHAR(50) IN ({placeholders})
                GROUP BY DATE(l.date_disbursed)
                ORDER BY disb_date DESC
                LIMIT 10
            """
            django_cursor = django_conn.cursor()
            django_cursor.execute(query_by_date, tuple(list(missing_loan_ids)[:100]))
            date_counts = django_cursor.fetchall()
            django_cursor.close()

            print(f"{'Date':<15} {'Missing Loans':<15}")
            print("-" * 30)
            for date, count in date_counts:
                print(f"{str(date):<15} {count:<15}")

        else:
            print("\n✅ No missing loans found - databases are in sync!")

        # Check for extra loans in Seeds Metrics (shouldn't happen)
        extra_loan_ids = seeds_loan_ids - django_loan_ids
        if extra_loan_ids:
            print(f"\n⚠️  WARNING: {len(extra_loan_ids):,} loans in Seeds Metrics but NOT in Django!")
            print(f"Sample IDs: {list(extra_loan_ids)[:10]}")

        # Summary
        print(f"\n{'='*80}")
        print(f"  SUMMARY")
        print(f"{'='*80}")
        print(f"Django (source):        {len(django_loan_ids):,} disbursed loans")
        print(f"Seeds Metrics (target): {len(seeds_loan_ids):,} loans")
        print(f"Missing from Seeds:     {len(missing_loan_ids):,} loans")
        print(f"Extra in Seeds:         {len(extra_loan_ids):,} loans")
        print(f"{'='*80}\n")

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

