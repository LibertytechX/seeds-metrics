#!/usr/bin/env python3
"""
Analyze the relationship between verticals.tsv and the Seeds Metrics database.
This script compares officer IDs, names, emails, and identifies new organizational data.
"""

import os
import sys
import csv
import psycopg2
from psycopg2.extras import RealDictCursor
from collections import defaultdict
from dotenv import load_dotenv

# Load environment variables
load_dotenv()

# Database connection parameters for Seeds Metrics
SEEDS_DB_CONFIG = {
    'host': os.getenv('DB_HOST'),
    'port': int(os.getenv('DB_PORT', 25060)),
    'database': os.getenv('DB_NAME'),
    'user': os.getenv('DB_USER'),
    'password': os.getenv('DB_PASSWORD'),
    'sslmode': 'require'
}

def connect_to_seedsmetrics():
    """Connect to Seeds Metrics database"""
    try:
        conn = psycopg2.connect(**SEEDS_DB_CONFIG)
        return conn
    except Exception as e:
        print(f"Error connecting to Seeds Metrics database: {e}")
        sys.exit(1)

def load_verticals_tsv(filepath):
    """Load and parse the verticals.tsv file"""
    officers_tsv = {}

    with open(filepath, 'r', encoding='utf-8') as f:
        reader = csv.DictReader(f, delimiter='\t')
        for row in reader:
            officer_id = (row.get('loan_officer_id') or '').strip()
            if not officer_id:
                continue

            officers_tsv[officer_id] = {
                'officer_id': officer_id,
                'email': (row.get('loan_officer_email') or '').strip(),
                'name': (row.get('loan_officer_name') or '').strip(),
                'phone': (row.get('loan_officer_phone') or '').strip(),
                'branch_name': (row.get('branch_name') or '').strip(),
                'branch_location': (row.get('branch_location') or '').strip(),
                'branch_sub_location': (row.get('branch_sub_location') or '').strip(),
                'branch_state': (row.get('branch_state') or '').strip(),
                'branch_supervisor_email': (row.get('branch_supervisor_email') or '').strip(),
                'branch_supervisor_name': (row.get('branch_supervisor_name') or '').strip(),
                'region_name': (row.get('region_name') or '').strip(),
                'vertical_lead_email': (row.get('vertical_lead_email') or '').strip(),
                'vertical_lead_name': (row.get('vertical_lead_name') or '').strip(),
            }

    return officers_tsv

def get_database_officers(conn):
    """Get all officers from the Seeds Metrics database"""
    query = """
        SELECT
            officer_id,
            officer_name,
            officer_email,
            officer_phone,
            branch,
            region,
            hire_date,
            user_type
        FROM officers
        ORDER BY officer_id
    """

    with conn.cursor(cursor_factory=RealDictCursor) as cur:
        cur.execute(query)
        officers_db = {str(row['officer_id']): dict(row) for row in cur.fetchall()}

    return officers_db

def analyze_officer_matching(officers_tsv, officers_db):
    """Analyze the matching between TSV and database officers"""

    tsv_ids = set(officers_tsv.keys())
    db_ids = set(officers_db.keys())

    # Find matches and mismatches
    matched_ids = tsv_ids & db_ids
    tsv_only_ids = tsv_ids - db_ids
    db_only_ids = db_ids - tsv_ids

    # Calculate match rate
    total_tsv = len(tsv_ids)
    total_db = len(db_ids)
    total_matched = len(matched_ids)
    match_rate = (total_matched / total_tsv * 100) if total_tsv > 0 else 0

    return {
        'total_tsv': total_tsv,
        'total_db': total_db,
        'total_matched': total_matched,
        'match_rate': match_rate,
        'matched_ids': matched_ids,
        'tsv_only_ids': tsv_only_ids,
        'db_only_ids': db_only_ids
    }

def compare_officer_data(officers_tsv, officers_db, matched_ids):
    """Compare officer data for matched IDs"""

    discrepancies = []

    for officer_id in matched_ids:
        tsv_officer = officers_tsv[officer_id]
        db_officer = officers_db[officer_id]

        issues = []

        # Compare names (normalize for comparison)
        tsv_name = tsv_officer['name'].upper().strip()
        db_name = (db_officer['officer_name'] or '').upper().strip()
        if tsv_name != db_name and db_name != '':
            issues.append(f"Name mismatch: TSV='{tsv_officer['name']}' vs DB='{db_officer['officer_name']}'")

        # Compare emails (normalize for comparison)
        tsv_email = tsv_officer['email'].lower().strip()
        db_email = (db_officer['officer_email'] or '').lower().strip()
        if tsv_email != db_email and db_email != '':
            issues.append(f"Email mismatch: TSV='{tsv_officer['email']}' vs DB='{db_officer['officer_email']}'")

        # Compare branch
        tsv_branch = tsv_officer['branch_name'].upper().strip()
        db_branch = (db_officer['branch'] or '').upper().strip()
        if tsv_branch != db_branch and db_branch != '':
            issues.append(f"Branch mismatch: TSV='{tsv_officer['branch_name']}' vs DB='{db_officer['branch']}'")

        if issues:
            discrepancies.append({
                'officer_id': officer_id,
                'issues': issues
            })

    return discrepancies

def analyze_new_data_fields(officers_tsv):
    """Analyze the new organizational hierarchy data in TSV"""

    # Count unique values for each organizational field
    regions = set()
    supervisors = set()
    vertical_leads = set()
    branch_locations = set()
    branch_states = set()

    for officer in officers_tsv.values():
        if officer['region_name']:
            regions.add(officer['region_name'])
        if officer['branch_supervisor_email']:
            supervisors.add(f"{officer['branch_supervisor_name']} ({officer['branch_supervisor_email']})")
        if officer['vertical_lead_email']:
            vertical_leads.add(f"{officer['vertical_lead_name']} ({officer['vertical_lead_email']})")
        if officer['branch_location']:
            branch_locations.add(officer['branch_location'])
        if officer['branch_state']:
            branch_states.add(officer['branch_state'])

    return {
        'regions': sorted(regions),
        'supervisors': sorted(supervisors),
        'vertical_leads': sorted(vertical_leads),
        'branch_locations': sorted(branch_locations),
        'branch_states': sorted(branch_states)
    }

def main():
    print("=" * 80)
    print("VERTICALS.TSV vs SEEDS METRICS DATABASE ANALYSIS")
    print("=" * 80)
    print()

    # Load TSV file
    tsv_filepath = 'verticals.tsv'
    if not os.path.exists(tsv_filepath):
        print(f"Error: {tsv_filepath} not found!")
        sys.exit(1)

    print(f"Loading {tsv_filepath}...")
    officers_tsv = load_verticals_tsv(tsv_filepath)
    print(f"✓ Loaded {len(officers_tsv)} officers from TSV file")
    print()

    # Connect to database
    print("Connecting to Seeds Metrics database...")
    conn = connect_to_seedsmetrics()
    print("✓ Connected to Seeds Metrics database")
    print()

    # Get database officers
    print("Fetching officers from database...")
    officers_db = get_database_officers(conn)
    print(f"✓ Loaded {len(officers_db)} officers from database")
    print()

    # Analyze matching
    print("=" * 80)
    print("1. OFFICER ID MATCHING ANALYSIS")
    print("=" * 80)
    print()

    matching_results = analyze_officer_matching(officers_tsv, officers_db)

    print(f"Total officers in verticals.tsv:     {matching_results['total_tsv']}")
    print(f"Total officers in database:          {matching_results['total_db']}")
    print(f"Officers found in both sources:      {matching_results['total_matched']}")
    print(f"Match rate:                          {matching_results['match_rate']:.2f}%")
    print()
    print(f"Officers in TSV but NOT in database: {len(matching_results['tsv_only_ids'])}")
    print(f"Officers in database but NOT in TSV: {len(matching_results['db_only_ids'])}")
    print()

    # Show officers in TSV but not in database
    if matching_results['tsv_only_ids']:
        print("Officers in TSV but NOT in database (first 20):")
        for i, officer_id in enumerate(sorted(matching_results['tsv_only_ids'])[:20], 1):
            officer = officers_tsv[officer_id]
            print(f"  {i}. ID: {officer_id}, Name: {officer['name']}, Email: {officer['email']}")
        if len(matching_results['tsv_only_ids']) > 20:
            print(f"  ... and {len(matching_results['tsv_only_ids']) - 20} more")
        print()

    # Show officers in database but not in TSV
    if matching_results['db_only_ids']:
        print(f"Officers in database but NOT in TSV (first 20):")
        for i, officer_id in enumerate(sorted(matching_results['db_only_ids'])[:20], 1):
            officer = officers_db[officer_id]
            print(f"  {i}. ID: {officer_id}, Name: {officer['officer_name']}, Email: {officer['officer_email']}")
        if len(matching_results['db_only_ids']) > 20:
            print(f"  ... and {len(matching_results['db_only_ids']) - 20} more")
        print()

    # Compare data for matched officers
    print("=" * 80)
    print("2. DATA CONSISTENCY ANALYSIS (for matched officers)")
    print("=" * 80)
    print()

    discrepancies = compare_officer_data(officers_tsv, officers_db, matching_results['matched_ids'])

    if discrepancies:
        print(f"Found {len(discrepancies)} officers with data discrepancies (first 10):")
        print()
        for i, disc in enumerate(discrepancies[:10], 1):
            print(f"{i}. Officer ID: {disc['officer_id']}")
            for issue in disc['issues']:
                print(f"   - {issue}")
            print()
        if len(discrepancies) > 10:
            print(f"... and {len(discrepancies) - 10} more officers with discrepancies")
    else:
        print("✓ No data discrepancies found for matched officers!")
    print()

    # Analyze new organizational data
    print("=" * 80)
    print("3. NEW ORGANIZATIONAL HIERARCHY DATA IN VERTICALS.TSV")
    print("=" * 80)
    print()

    new_data = analyze_new_data_fields(officers_tsv)

    print(f"Unique Regions/Verticals ({len(new_data['regions'])}):")
    for region in new_data['regions']:
        print(f"  - {region}")
    print()

    print(f"Unique Branch Supervisors ({len(new_data['supervisors'])}):")
    for supervisor in new_data['supervisors'][:15]:
        print(f"  - {supervisor}")
    if len(new_data['supervisors']) > 15:
        print(f"  ... and {len(new_data['supervisors']) - 15} more")
    print()

    print(f"Unique Vertical Leads ({len(new_data['vertical_leads'])}):")
    for lead in new_data['vertical_leads']:
        print(f"  - {lead}")
    print()

    print(f"Unique Branch Locations ({len(new_data['branch_locations'])}):")
    for location in sorted(new_data['branch_locations'])[:15]:
        print(f"  - {location}")
    if len(new_data['branch_locations']) > 15:
        print(f"  ... and {len(new_data['branch_locations']) - 15} more")
    print()

    print(f"Unique Branch States ({len(new_data['branch_states'])}):")
    for state in new_data['branch_states']:
        print(f"  - {state}")
    print()

    conn.close()

    print("=" * 80)
    print("ANALYSIS COMPLETE")
    print("=" * 80)

if __name__ == '__main__':
    main()

