#!/usr/bin/env python3
"""
Loan Metrics Validation Test Suite

This test suite validates that all computed loan metrics in the SeedsMetrics database
are correctly calculated and match the source data from the Django database.

Usage:
    python backend/tests/test_loan_metrics_validation.py

Environment Variables Required:
    - DJANGO_DB_HOST (default: 164.90.155.2)
    - DJANGO_DB_PORT (default: 5432)
    - DJANGO_DB_NAME (default: savings)
    - DJANGO_DB_USER (default: metricsuser)
    - DJANGO_DB_PASSWORD
    - SEEDSMETRICS_DB_HOST (default: generaldb-do-user-9489371-0.k.db.ondigitalocean.com)
    - SEEDSMETRICS_DB_PORT (default: 25060)
    - SEEDSMETRICS_DB_NAME (default: seedsmetrics)
    - SEEDSMETRICS_DB_USER (default: seedsuser)
    - SEEDSMETRICS_DB_PASSWORD
"""

import os
import sys
from datetime import datetime, date, timedelta
from decimal import Decimal
from typing import Dict, List, Tuple, Any
import psycopg2
from psycopg2.extras import RealDictCursor

# Test configuration
TEST_LOAN_IDS = ['8', '3475', '11883', '2118', '7150', '12129', '107', '3523', '16192', '18691']

# Color codes for terminal output
class Colors:
    GREEN = '\033[92m'
    RED = '\033[91m'
    YELLOW = '\033[93m'
    BLUE = '\033[94m'
    BOLD = '\033[1m'
    END = '\033[0m'


class DatabaseConnection:
    """Manages database connections"""

    def __init__(self, host: str, port: int, dbname: str, user: str, password: str):
        self.conn_params = {
            'host': host,
            'port': port,
            'dbname': dbname,
            'user': user,
            'password': password,
            'sslmode': 'require'
        }
        self.conn = None

    def connect(self):
        """Establish database connection"""
        self.conn = psycopg2.connect(**self.conn_params)
        return self.conn

    def close(self):
        """Close database connection"""
        if self.conn:
            self.conn.close()

    def execute_query(self, query: str, params: tuple = None) -> List[Dict]:
        """Execute query and return results as list of dicts"""
        with self.conn.cursor(cursor_factory=RealDictCursor) as cur:
            cur.execute(query, params)
            return cur.fetchall()


class MetricsValidator:
    """Validates loan metrics between Django and SeedsMetrics databases"""

    def __init__(self, django_conn: DatabaseConnection, metrics_conn: DatabaseConnection):
        self.django_conn = django_conn
        self.metrics_conn = metrics_conn
        self.test_results = []
        self.total_tests = 0
        self.passed_tests = 0
        self.failed_tests = 0

    def count_business_days(self, start_date: date, end_date: date) -> int:
        """Count business days between two dates (excluding weekends)"""
        if start_date is None or end_date is None:
            return 0

        days = 0
        current = start_date
        while current <= end_date:
            # 0 = Monday, 6 = Sunday
            if current.weekday() < 5:  # Monday to Friday
                days += 1
            current += timedelta(days=1)
        return days

    def get_django_loan_data(self, loan_id: str) -> Dict:
        """Fetch loan data from Django database"""
        query = """
            SELECT
                id as loan_id,
                customer_id,
                loan_amount,
                date_disbursed as disbursement_date,
                maturity_date,
                loan_term_days,
                interest_rate,
                fee_amount,
                start_date as first_payment_due_date,
                status
            FROM loans_ajoloan
            WHERE id = %s
        """
        results = self.django_conn.execute_query(query, (loan_id,))
        return results[0] if results else None

    def get_django_repayments(self, loan_id: str) -> List[Dict]:
        """Fetch repayments from Django database"""
        query = """
            SELECT
                payment_date,
                principal_paid,
                interest_paid,
                fees_paid,
                payment_amount,
                is_reversed
            FROM loans_ajorepayment
            WHERE loan_id = %s AND (is_reversed IS NULL OR is_reversed = FALSE)
            ORDER BY payment_date
        """
        return self.django_conn.execute_query(query, (loan_id,))

    def get_metrics_loan_data(self, loan_id: str) -> Dict:
        """Fetch loan data from SeedsMetrics database"""
        query = """
            SELECT
                loan_id,
                loan_amount,
                disbursement_date,
                maturity_date,
                loan_term_days,
                interest_rate,
                fee_amount,
                first_payment_due_date,
                first_payment_received_date,
                status,
                -- Computed fields
                total_principal_paid,
                total_interest_paid,
                total_fees_paid,
                total_repayments,
                principal_outstanding,
                interest_outstanding,
                fees_outstanding,
                total_outstanding,
                actual_outstanding,
                repayment_amount,
                -- DPD and timing fields
                current_dpd,
                max_dpd_ever,
                loan_age,
                days_since_last_repayment,
                days_since_due,
                -- Business days fields
                daily_repayment_amount,
                real_loan_tenure_days,
                repayment_days_paid,
                repayment_days_due_today,
                business_days_since_disbursement,
                -- Risk indicators
                fimr_tagged,
                early_indicator_tagged,
                first_payment_missed,
                -- Health metrics
                repayment_delay_rate,
                repayment_health,
                timeliness_score,
                -- Metadata
                created_at,
                updated_at
            FROM loans
            WHERE loan_id = %s
        """
        results = self.metrics_conn.execute_query(query, (loan_id,))
        return results[0] if results else None

    def get_metrics_repayments(self, loan_id: str) -> List[Dict]:
        """Fetch repayments from SeedsMetrics database"""
        query = """
            SELECT
                payment_date,
                principal_paid,
                interest_paid,
                fees_paid,
                payment_amount,
                is_reversed,
                dpd_at_payment
            FROM repayments
            WHERE loan_id = %s AND (is_reversed IS NULL OR is_reversed = FALSE)
            ORDER BY payment_date
        """
        return self.metrics_conn.execute_query(query, (loan_id,))

    def assert_equal(self, test_name: str, expected: Any, actual: Any, tolerance: Decimal = None) -> bool:
        """Assert two values are equal (with optional tolerance for decimals)"""
        self.total_tests += 1

        if tolerance and isinstance(expected, (Decimal, float)) and isinstance(actual, (Decimal, float)):
            passed = abs(Decimal(str(expected)) - Decimal(str(actual))) <= tolerance
        else:
            passed = expected == actual

        if passed:
            self.passed_tests += 1
            print(f"{Colors.GREEN}✓{Colors.END} {test_name}: {actual}")
        else:
            self.failed_tests += 1
            print(f"{Colors.RED}✗{Colors.END} {test_name}")
            print(f"  Expected: {expected}")
            print(f"  Actual:   {actual}")

        self.test_results.append({
            'test': test_name,
            'passed': passed,
            'expected': expected,
            'actual': actual
        })

        return passed

    def validate_basic_fields(self, loan_id: str, django_data: Dict, metrics_data: Dict):
        """Validate basic loan fields match between databases"""
        print(f"\n{Colors.BOLD}Basic Fields Validation{Colors.END}")

        self.assert_equal(
            "Loan Amount",
            Decimal(str(django_data['loan_amount'])),
            metrics_data['loan_amount'],
            tolerance=Decimal('0.01')
        )

        self.assert_equal(
            "Disbursement Date",
            django_data['disbursement_date'],
            metrics_data['disbursement_date']
        )

        self.assert_equal(
            "Maturity Date",
            django_data['maturity_date'],
            metrics_data['maturity_date']
        )

        self.assert_equal(
            "Loan Term Days",
            django_data['loan_term_days'],
            metrics_data['loan_term_days']
        )

        self.assert_equal(
            "Interest Rate",
            Decimal(str(django_data['interest_rate'])),
            metrics_data['interest_rate'],
            tolerance=Decimal('0.0001')
        )

        self.assert_equal(
            "First Payment Due Date",
            django_data['first_payment_due_date'],
            metrics_data['first_payment_due_date']
        )

    def validate_repayment_totals(self, loan_id: str, django_repayments: List[Dict], metrics_data: Dict):
        """Validate repayment totals are correctly calculated"""
        print(f"\n{Colors.BOLD}Repayment Totals Validation{Colors.END}")

        # Calculate expected totals from Django repayments
        expected_principal = sum(Decimal(str(r['principal_paid'])) for r in django_repayments)
        expected_interest = sum(Decimal(str(r['interest_paid'])) for r in django_repayments)
        expected_fees = sum(Decimal(str(r['fees_paid'])) for r in django_repayments)
        expected_total = sum(Decimal(str(r['payment_amount'])) for r in django_repayments)

        self.assert_equal(
            "Total Principal Paid",
            expected_principal,
            metrics_data['total_principal_paid'],
            tolerance=Decimal('0.01')
        )

        self.assert_equal(
            "Total Interest Paid",
            expected_interest,
            metrics_data['total_interest_paid'],
            tolerance=Decimal('0.01')
        )

        self.assert_equal(
            "Total Fees Paid",
            expected_fees,
            metrics_data['total_fees_paid'],
            tolerance=Decimal('0.01')
        )

        self.assert_equal(
            "Total Repayments",
            expected_total,
            metrics_data['total_repayments'],
            tolerance=Decimal('0.01')
        )

    def validate_outstanding_balances(self, loan_id: str, django_data: Dict, metrics_data: Dict):
        """Validate outstanding balance calculations"""
        print(f"\n{Colors.BOLD}Outstanding Balances Validation{Colors.END}")

        loan_amount = Decimal(str(django_data['loan_amount']))
        interest_rate = Decimal(str(django_data['interest_rate']))
        loan_term_days = django_data['loan_term_days']
        fee_amount = Decimal(str(django_data['fee_amount'])) if django_data['fee_amount'] else Decimal('0')

        # Expected outstanding balances
        expected_principal_outstanding = loan_amount - metrics_data['total_principal_paid']
        expected_interest_outstanding = max(Decimal('0'), (loan_amount * interest_rate) - metrics_data['total_interest_paid'])
        expected_fees_outstanding = max(Decimal('0'), fee_amount - metrics_data['total_fees_paid'])
        expected_total_outstanding = expected_principal_outstanding + expected_interest_outstanding + expected_fees_outstanding

        self.assert_equal(
            "Principal Outstanding",
            expected_principal_outstanding,
            metrics_data['principal_outstanding'],
            tolerance=Decimal('0.01')
        )

        self.assert_equal(
            "Interest Outstanding",
            expected_interest_outstanding,
            metrics_data['interest_outstanding'],
            tolerance=Decimal('0.01')
        )

        self.assert_equal(
            "Fees Outstanding",
            expected_fees_outstanding,
            metrics_data['fees_outstanding'],
            tolerance=Decimal('0.01')
        )

        self.assert_equal(
            "Total Outstanding",
            expected_total_outstanding,
            metrics_data['total_outstanding'],
            tolerance=Decimal('0.01')
        )

    def validate_business_days_calculations(self, loan_id: str, django_data: Dict, metrics_data: Dict):
        """Validate business days calculations"""
        print(f"\n{Colors.BOLD}Business Days Calculations Validation{Colors.END}")

        from datetime import timedelta

        disbursement_date = django_data['disbursement_date']
        first_payment_due_date = django_data['first_payment_due_date']
        maturity_date = django_data['maturity_date']

        # Calculate expected business days since disbursement
        if disbursement_date:
            expected_business_days = self.count_business_days(disbursement_date, date.today())
            self.assert_equal(
                "Business Days Since Disbursement",
                expected_business_days,
                metrics_data['business_days_since_disbursement']
            )

        # Calculate expected loan age
        if disbursement_date:
            expected_loan_age = (date.today() - disbursement_date).days
            self.assert_equal(
                "Loan Age",
                expected_loan_age,
                metrics_data['loan_age']
            )

        # Validate daily repayment amount
        loan_amount = Decimal(str(django_data['loan_amount']))
        interest_rate = Decimal(str(django_data['interest_rate']))
        fee_amount = Decimal(str(django_data['fee_amount'])) if django_data['fee_amount'] else Decimal('0')
        loan_term_days = django_data['loan_term_days']

        repayment_amount = loan_amount * (1 + interest_rate) + fee_amount
        expected_daily_repayment = repayment_amount / loan_term_days if loan_term_days > 0 else Decimal('0')

        self.assert_equal(
            "Daily Repayment Amount",
            expected_daily_repayment,
            metrics_data['daily_repayment_amount'],
            tolerance=Decimal('0.01')
        )

        # Validate repayment days paid
        if metrics_data['daily_repayment_amount'] and metrics_data['daily_repayment_amount'] > 0:
            expected_repayment_days_paid = metrics_data['total_repayments'] / metrics_data['daily_repayment_amount']
            self.assert_equal(
                "Repayment Days Paid",
                expected_repayment_days_paid,
                metrics_data['repayment_days_paid'],
                tolerance=Decimal('0.01')
            )

    def validate_dpd_calculations(self, loan_id: str, metrics_data: Dict):
        """Validate DPD (Days Past Due) calculations"""
        print(f"\n{Colors.BOLD}DPD Calculations Validation{Colors.END}")

        # Calculate expected DPD
        if metrics_data['actual_outstanding'] and metrics_data['actual_outstanding'] > 0:
            expected_dpd = max(0, metrics_data['repayment_days_due_today'] - int(metrics_data['repayment_days_paid']))
        else:
            expected_dpd = 0

        self.assert_equal(
            "Current DPD",
            expected_dpd,
            metrics_data['current_dpd']
        )

        # Validate early indicator tagging
        expected_early_indicator = 1 <= metrics_data['current_dpd'] <= 6
        self.assert_equal(
            "Early Indicator Tagged",
            expected_early_indicator,
            metrics_data['early_indicator_tagged']
        )

    def validate_actual_outstanding(self, loan_id: str, metrics_data: Dict):
        """Validate actual_outstanding calculation"""
        print(f"\n{Colors.BOLD}Actual Outstanding Validation{Colors.END}")

        # Calculate expected actual_outstanding
        if metrics_data['daily_repayment_amount'] and metrics_data['repayment_days_due_today']:
            expected_actual_outstanding = max(Decimal('0'),
                (metrics_data['daily_repayment_amount'] * metrics_data['repayment_days_due_today']) -
                metrics_data['total_repayments']
            )
        else:
            expected_actual_outstanding = Decimal('0')

        self.assert_equal(
            "Actual Outstanding",
            expected_actual_outstanding,
            metrics_data['actual_outstanding'],
            tolerance=Decimal('0.01')
        )

    def validate_repayment_delay_rate(self, loan_id: str, metrics_data: Dict):
        """Validate repayment_delay_rate calculation"""
        print(f"\n{Colors.BOLD}Repayment Delay Rate Validation{Colors.END}")

        loan_age = metrics_data['loan_age']
        current_dpd = metrics_data['current_dpd']
        days_since_last_repayment = metrics_data['days_since_last_repayment']

        # Calculate expected repayment_delay_rate
        if loan_age and loan_age > 0:
            if days_since_last_repayment is not None:
                # Use average of days_since_last_repayment and current_dpd
                expected_rate = (1.0 - ((((days_since_last_repayment + current_dpd) / 2.0) / loan_age) / 0.25)) * 100
            else:
                # No payments yet, use only current_dpd
                expected_rate = (1.0 - ((current_dpd / loan_age) / 0.25)) * 100
        elif loan_age == 0:
            expected_rate = 0
        else:
            expected_rate = None

        if expected_rate is not None:
            self.assert_equal(
                "Repayment Delay Rate",
                Decimal(str(round(expected_rate, 2))),
                metrics_data['repayment_delay_rate'],
                tolerance=Decimal('0.1')
            )
        else:
            print(f"{Colors.YELLOW}⚠{Colors.END} Repayment Delay Rate: NULL (expected)")

    def validate_fimr_tagging(self, loan_id: str, django_repayments: List[Dict], metrics_data: Dict):
        """Validate FIMR (First Installment Missed Repayment) tagging"""
        print(f"\n{Colors.BOLD}FIMR Tagging Validation{Colors.END}")

        first_payment_due_date = metrics_data['first_payment_due_date']
        first_payment_received_date = metrics_data['first_payment_received_date']

        # Calculate expected FIMR tag
        if first_payment_due_date is None:
            expected_fimr = True
        elif first_payment_received_date and first_payment_received_date <= first_payment_due_date:
            expected_fimr = False
        elif first_payment_received_date is None and first_payment_due_date >= date.today():
            expected_fimr = False
        else:
            expected_fimr = True

        self.assert_equal(
            "FIMR Tagged",
            expected_fimr,
            metrics_data['fimr_tagged']
        )

        # Validate first_payment_missed
        if first_payment_received_date and first_payment_due_date:
            expected_first_payment_missed = first_payment_received_date > first_payment_due_date
        elif first_payment_received_date is None:
            expected_first_payment_missed = True
        else:
            expected_first_payment_missed = False

        self.assert_equal(
            "First Payment Missed",
            expected_first_payment_missed,
            metrics_data['first_payment_missed']
        )

    def validate_loan(self, loan_id: str):
        """Run all validations for a single loan"""
        print(f"\n{Colors.BOLD}{Colors.BLUE}{'='*80}{Colors.END}")
        print(f"{Colors.BOLD}{Colors.BLUE}Validating Loan ID: {loan_id}{Colors.END}")
        print(f"{Colors.BOLD}{Colors.BLUE}{'='*80}{Colors.END}")

        # Fetch data from both databases
        django_data = self.get_django_loan_data(loan_id)
        metrics_data = self.get_metrics_loan_data(loan_id)

        if not django_data:
            print(f"{Colors.RED}ERROR: Loan {loan_id} not found in Django database{Colors.END}")
            return

        if not metrics_data:
            print(f"{Colors.RED}ERROR: Loan {loan_id} not found in SeedsMetrics database{Colors.END}")
            return

        # Fetch repayments
        django_repayments = self.get_django_repayments(loan_id)

        # Run all validation tests
        self.validate_basic_fields(loan_id, django_data, metrics_data)
        self.validate_repayment_totals(loan_id, django_repayments, metrics_data)
        self.validate_outstanding_balances(loan_id, django_data, metrics_data)
        self.validate_business_days_calculations(loan_id, django_data, metrics_data)
        self.validate_actual_outstanding(loan_id, metrics_data)
        self.validate_dpd_calculations(loan_id, metrics_data)
        self.validate_repayment_delay_rate(loan_id, metrics_data)
        self.validate_fimr_tagging(loan_id, django_repayments, metrics_data)

    def print_summary(self):
        """Print test summary"""
        print(f"\n{Colors.BOLD}{Colors.BLUE}{'='*80}{Colors.END}")
        print(f"{Colors.BOLD}TEST SUMMARY{Colors.END}")
        print(f"{Colors.BOLD}{Colors.BLUE}{'='*80}{Colors.END}")
        print(f"Total Tests: {self.total_tests}")
        print(f"{Colors.GREEN}Passed: {self.passed_tests}{Colors.END}")
        print(f"{Colors.RED}Failed: {self.failed_tests}{Colors.END}")

        if self.failed_tests == 0:
            print(f"\n{Colors.GREEN}{Colors.BOLD}✓ ALL TESTS PASSED!{Colors.END}")
        else:
            print(f"\n{Colors.RED}{Colors.BOLD}✗ SOME TESTS FAILED{Colors.END}")
            print(f"\nFailed Tests:")
            for result in self.test_results:
                if not result['passed']:
                    print(f"  - {result['test']}")

        pass_rate = (self.passed_tests / self.total_tests * 100) if self.total_tests > 0 else 0
        print(f"\nPass Rate: {pass_rate:.1f}%")
        print(f"{Colors.BOLD}{Colors.BLUE}{'='*80}{Colors.END}\n")


def main():
    """Main test runner"""
    print(f"{Colors.BOLD}{Colors.BLUE}{'='*80}{Colors.END}")
    print(f"{Colors.BOLD}Loan Metrics Validation Test Suite{Colors.END}")
    print(f"{Colors.BOLD}{Colors.BLUE}{'='*80}{Colors.END}\n")

    # Get database credentials from environment
    django_db = DatabaseConnection(
        host=os.getenv('DJANGO_DB_HOST', '164.90.155.2'),
        port=int(os.getenv('DJANGO_DB_PORT', '5432')),
        dbname=os.getenv('DJANGO_DB_NAME', 'savings'),
        user=os.getenv('DJANGO_DB_USER', 'metricsuser'),
        password=os.getenv('DJANGO_DB_PASSWORD', 'EiRXo6IfeHQuM3wcbZ67$LzwmVKCXhpUhWg')
    )

    metrics_db = DatabaseConnection(
        host=os.getenv('SEEDSMETRICS_DB_HOST', 'generaldb-do-user-9489371-0.k.db.ondigitalocean.com'),
        port=int(os.getenv('SEEDSMETRICS_DB_PORT', '25060')),
        dbname=os.getenv('SEEDSMETRICS_DB_NAME', 'seedsmetrics'),
        user=os.getenv('SEEDSMETRICS_DB_USER', 'seedsuser'),
        password=os.getenv('SEEDSMETRICS_DB_PASSWORD', '@seedsuser2020')
    )

    try:
        # Connect to databases
        print("Connecting to Django database...")
        django_db.connect()
        print(f"{Colors.GREEN}✓ Connected to Django database{Colors.END}\n")

        print("Connecting to SeedsMetrics database...")
        metrics_db.connect()
        print(f"{Colors.GREEN}✓ Connected to SeedsMetrics database{Colors.END}\n")

        # Create validator
        validator = MetricsValidator(django_db, metrics_db)

        # Run tests for each loan
        print(f"Testing {len(TEST_LOAN_IDS)} loans: {', '.join(TEST_LOAN_IDS)}\n")

        for loan_id in TEST_LOAN_IDS:
            try:
                validator.validate_loan(loan_id)
            except Exception as e:
                print(f"{Colors.RED}ERROR validating loan {loan_id}: {str(e)}{Colors.END}")
                import traceback
                traceback.print_exc()

        # Print summary
        validator.print_summary()

        # Exit with appropriate code
        sys.exit(0 if validator.failed_tests == 0 else 1)

    except Exception as e:
        print(f"{Colors.RED}FATAL ERROR: {str(e)}{Colors.END}")
        import traceback
        traceback.print_exc()
        sys.exit(1)

    finally:
        # Close connections
        django_db.close()
        metrics_db.close()


if __name__ == '__main__':
    main()

