# ETL System Database Schema Documentation

## Overview

This document describes the complete database schema used by the ETL (Extract, Transform, Load) system for synchronizing loan analytics data from the Django application to the external analytics platform.

## Table of Contents

1. [ETL Tracking Tables](#etl-tracking-tables)
2. [Core Business Tables](#core-business-tables)
3. [Entity Relationships](#entity-relationships)
4. [Field Descriptions](#field-descriptions)

---

## ETL Tracking Tables

These tables are specific to the ETL system and track synchronization operations, errors, and timestamps.

### 1. ETLSyncHistory

**Purpose**: Tracks each ETL sync operation (incremental or full) with detailed metrics and results.

**Table Name**: `etl_folder_etlsynchistory`

| Field Name | Type | Constraints | Description |
|------------|------|-------------|-------------|
| `id` | Integer | Primary Key, Auto-increment | Unique record identifier |
| `sync_id` | CharField(100) | Unique, Indexed | Unique sync operation identifier (UUID format) |
| `sync_type` | CharField(20) | Choices: 'incremental', 'full' | Type of sync operation |
| `status` | CharField(20) | Choices: 'pending', 'in_progress', 'completed', 'failed', 'partial_success' | Current sync status |
| `started_at` | DateTimeField | Auto-now-add | Sync start timestamp |
| `completed_at` | DateTimeField | Nullable | Sync completion timestamp |
| `sync_start_date` | DateTimeField | Nullable | Data window start date |
| `sync_end_date` | DateTimeField | Nullable | Data window end date |
| `total_loans` | IntegerField | Default: 0 | Total number of loans to sync |
| `total_repayments` | IntegerField | Default: 0 | Total number of repayments to sync |
| `total_customers` | IntegerField | Default: 0 | Total number of customers to sync |
| `total_officers` | IntegerField | Default: 0 | Total number of officers to sync |
| `loans_success` | IntegerField | Default: 0 | Successfully synced loans |
| `loans_failed` | IntegerField | Default: 0 | Failed loan syncs |
| `repayments_success` | IntegerField | Default: 0 | Successfully synced repayments |
| `repayments_failed` | IntegerField | Default: 0 | Failed repayment syncs |
| `customers_success` | IntegerField | Default: 0 | Successfully synced customers |
| `customers_failed` | IntegerField | Default: 0 | Failed customer syncs |
| `officers_success` | IntegerField | Default: 0 | Successfully synced officers |
| `officers_failed` | IntegerField | Default: 0 | Failed officer syncs |
| `api_response` | JSONField | Nullable | Full API response data |
| `error_message` | TextField | Nullable | Error details if sync failed |
| `duration_seconds` | FloatField | Nullable | Total sync duration in seconds |

**Indexes**:
- `sync_id` (unique)
- `started_at` (descending)
- `status`
- `sync_type`

**Methods**:
- `mark_completed()`: Sets status to 'completed' and calculates duration
- `mark_failed(error_message)`: Sets status to 'failed' with error details
- `mark_partial_success()`: Sets status to 'partial_success' when some records fail

---

### 2. ETLSyncError

**Purpose**: Tracks individual errors for specific entities during sync operations.

**Table Name**: `etl_folder_etlsyncerror`

| Field Name | Type | Constraints | Description |
|------------|------|-------------|-------------|
| `id` | Integer | Primary Key, Auto-increment | Unique error record identifier |
| `sync_history` | ForeignKey | References ETLSyncHistory, CASCADE | Parent sync operation |
| `entity_type` | CharField(20) | Choices: 'loan', 'repayment', 'customer', 'officer' | Type of entity that failed |
| `entity_id` | CharField(100) | | ID of the failed entity |
| `error_code` | CharField(100) | Nullable | Error code from API |
| `error_message` | TextField | | Detailed error description |
| `error_details` | JSONField | Nullable | Additional error metadata |
| `created_at` | DateTimeField | Auto-now-add | Error timestamp |
| `retry_count` | IntegerField | Default: 0 | Number of retry attempts |
| `last_retry_at` | DateTimeField | Nullable | Last retry timestamp |
| `resolved` | BooleanField | Default: False | Whether error was resolved |
| `resolved_at` | DateTimeField | Nullable | Resolution timestamp |

**Indexes**:
- `entity_type, entity_id` (composite)
- `resolved`
- `created_at` (descending)

**Relationships**:
- **Many-to-One** with `ETLSyncHistory` (via `sync_history` field)
- Related name: `errors` (access via `sync_history.errors.all()`)

**Methods**:
- `mark_resolved()`: Sets resolved to True with timestamp

---

### 3. ETLLastSync

**Purpose**: Tracks the last successful sync timestamp for each entity type to enable incremental syncs.

**Table Name**: `etl_folder_etllastsync`

| Field Name | Type | Constraints | Description |
|------------|------|-------------|-------------|
| `id` | Integer | Primary Key, Auto-increment | Unique record identifier |
| `entity_type` | CharField(20) | Unique, Choices: 'loan', 'repayment', 'customer', 'officer' | Entity type |
| `last_sync_timestamp` | DateTimeField | | Last successful sync timestamp |
| `last_sync_id` | CharField(100) | Nullable | Last sync operation ID |
| `updated_at` | DateTimeField | Auto-now | Record update timestamp |

**Class Methods**:
- `update_last_sync(entity_type, sync_id)`: Updates or creates last sync record
- `get_last_sync_timestamp(entity_type)`: Returns last sync timestamp (defaults to 30 days ago if not found)

---

## Core Business Tables

These are the main application tables that the ETL system extracts data from.

### 4. CustomUser (Officers/Agents)

**Purpose**: Stores user accounts for agents, officers, and staff who manage loans and customers.

**Table Name**: `accounts_customuser`

**Django Model**: `accounts.models.CustomUser`

**Key Fields**:

| Field Name | Type | Constraints | Description |
|------------|------|-------------|-------------|
| `id` | Integer | Primary Key | Unique user identifier |
| `email` | EmailField | Unique, Indexed | User email address |
| `username` | CharField(100) | Nullable | Username (not unique) |
| `customer_user_id` | BigIntegerField | Unique, Nullable, Indexed | Customer user ID |
| `user_password` | CharField(500) | Nullable | Encrypted password |
| `user_type` | CharField(100) | Nullable | **CRITICAL**: User categorization |
| `account_type` | CharField(100) | Default: 'REGULAR' | Account tier (REGULAR or KEY_ACCOUNT) |
| `user_phone` | CharField(100) | Unique, Nullable | Phone number |
| `user_branch` | CharField(150) | Nullable | Branch name |
| `bvn_number` | CharField(100) | Nullable | Bank Verification Number |
| `kyc_level` | SmallIntegerField | Default: 0 | KYC verification level (0-3) |
| `performance_status` | CharField(100) | Nullable | Agent performance status |
| `referral_code` | CharField(100) | Nullable | Referral code |
| `referrals_count` | IntegerField | Default: 0 | Number of referrals made |
| `call_agent_id` | IntegerField | Unique, Nullable | Call center agent ID |
| `is_call_agent` | BooleanField | Default: False | Is call center agent |
| `is_call_agent_admin` | BooleanField | Default: False | Is call center admin |
| `is_ajo_admin` | BooleanField | Default: False | Is AJO admin |
| `bucket` | JSONField | Default: [] | Call center bucket assignments |
| `bucket_designation` | CharField(5) | Nullable | Bucket designation code |
| `pnd` | BooleanField | Default: False | PND status |
| `address` | CharField(300) | Nullable | Physical address |
| `date_joined` | DateTimeField | Auto-now-add | Account creation date |
| `last_login` | DateTimeField | Nullable | Last login timestamp |

**User Type Values** (stored in `user_type` field):
- `AGENT` - Regular agent
- `STAFF_AGENT` - Staff agent
- `PROSPER_AGENT` - Prosper agent
- `DMO_AGENT` - DMO agent
- `AJO_AGENT` - AJO agent
- `RECOVERY_AGENT` - Recovery agent
- `MERCHANT` - Merchant user
- `KEY_ACCOUNT` - Key account user
- `PERSONAL` - Personal user
- `LIBERTY_RETAIL` - Liberty retail user
- `LOTTO_AGENT` - Lotto agent

**Account Type Values** (stored in `account_type` field):
- `REGULAR` - Regular account tier
- `KEY_ACCOUNT` - Key account tier (premium)

**ETL Usage**:
- Officers/agents are filtered by `user_type__in=[AGENT, STAFF_AGENT, PROSPER_AGENT, DMO_AGENT, AJO_AGENT, RECOVERY_AGENT]`
- **NOT** filtered by `account_type` (common mistake in v1.0.0-v1.0.2)

---

### 5. AjoUser (Customers/Borrowers)

**Purpose**: Stores customer/borrower information for the AJO savings and loan system.

**Table Name**: `ajo_ajouser`

**Django Model**: `ajo.models.AjoUser`

**Key Fields**:

| Field Name | Type | Constraints | Description |
|------------|------|-------------|-------------|
| `id` | Integer | Primary Key | Unique customer identifier |
| `user` | ForeignKey | References CustomUser, SET_NULL, Indexed | Agent managing this customer |
| `lite_user` | OneToOneField | References CustomUser, PROTECT, Indexed | Lite app user account (optional) |
| `phone_number` | CharField(100) | Unique, Indexed | Primary phone number |
| `pin` | CharField(100) | | 4-digit PIN for authentication |
| `lite_transaction_pin` | CharField(255) | Nullable | Transaction PIN for lite app |
| `first_name` | CharField(100) | Nullable | Customer first name |
| `last_name` | CharField(100) | Nullable | Customer last name |
| `alias` | CharField(100) | Nullable | Customer nickname/alias |
| `alt_phone` | CharField(100) | Nullable | Alternative phone number |
| `gender` | CharField(100) | Nullable | Gender (MALE/FEMALE) |
| `marital_status` | CharField(100) | Nullable | Marital status |
| `dob` | DateField | Nullable | Date of birth |
| `state` | CharField(100) | Nullable | State of residence |
| `lga` | CharField(100) | Nullable | Local Government Area |
| `address` | CharField(355) | Nullable | Physical address |
| `bus_stop` | CharField(100) | Nullable | Nearest bus stop |
| `landmark` | CharField(550) | Nullable | Landmark near address |
| `trade` | CharField(100) | Nullable | Occupation/trade |
| `trade_location` | CharField(255) | Nullable | Business location |
| `bvn` | CharField(100) | Nullable | Bank Verification Number |
| `nin` | CharField(100) | Unique, Nullable | National ID Number |
| `image` | FileField | Nullable | Profile photo |
| `longitude` | FloatField | Default: 0.0 | GPS longitude |
| `latitude` | FloatField | Default: 0.0 | GPS latitude |
| `location_state` | CharField(150) | Nullable | GPS-derived state |
| `location_city` | CharField(150) | Nullable | GPS-derived city |
| `location_country` | CharField(100) | Nullable | GPS-derived country |
| `location_address` | CharField(550) | Nullable | GPS-derived address |
| `onboarding_verified` | BooleanField | Default: False, Indexed | Onboarding verification status |
| `bvn_verified` | BooleanField | Default: False, Indexed | BVN verification status |
| `onboarding_complete` | BooleanField | Default: False, Indexed | Full onboarding completion |
| `onboarding_stage` | CharField(100) | Nullable | Current onboarding stage |
| `total_money_saved` | FloatField | Default: 0.0 | Total savings amount |
| `total_money_withdrawn` | FloatField | Default: 0.0 | Total withdrawals amount |
| `total_withdrawals` | IntegerField | Default: 0 | Number of withdrawals |
| `loandisk_borrower_id` | CharField(300) | Nullable | LoanDisk system borrower ID |
| `branch_name` | CharField(100) | Nullable | Branch name |
| `age` | IntegerField | Nullable | Age in years |
| `provider` | CharField(100) | Default: 'ajo' | Provider type (ajo/lite) |
| `date_created` | DateTimeField | Auto-now-add | Record creation timestamp |
| `date_modified` | DateTimeField | Auto-now | Last modification timestamp |

**Computed Properties**:
- `fullname`: Returns `f"{first_name} {last_name}"` or phone_number if names not set

**Relationships**:
- **Many-to-One** with `CustomUser` (agent) via `user` field
- **One-to-One** with `CustomUser` (lite user) via `lite_user` field
- **One-to-Many** with `AjoLoan` (borrower's loans)
- **One-to-Many** with `AjoLoanRepayment` (borrower's repayments)

---

### 6. AjoLoan (Loans)

**Purpose**: Stores loan application and disbursement information.

**Table Name**: `loans_ajoloan`

**Django Model**: `loans.models.AjoLoan`

**Key Fields**:

| Field Name | Type | Constraints | Description |
|------------|------|-------------|-------------|
| `id` | Integer | Primary Key | Unique loan identifier |
| `agent` | ForeignKey | References CustomUser, CASCADE | Agent managing the loan |
| `borrower` | ForeignKey | References AjoUser, CASCADE | Borrower/customer |
| `borrower_full_name` | CharField(100) | Nullable | Borrower's full name (cached) |
| `borrower_phone_number` | CharField(100) | Nullable | Borrower's phone (cached) |
| `eligibility` | ForeignKey | References LoanEligibility, CASCADE | Loan eligibility record |
| `guarantor` | ForeignKey | References LoanGuarantor, PROTECT, Nullable | Loan guarantor |
| `guarantor_full_name` | CharField(100) | Nullable | Guarantor's name |
| `guarantor_phone_number` | CharField(100) | Nullable | Guarantor's phone |
| `amount` | FloatField | Default: 0.0 | Requested loan amount |
| `amount_disbursed` | FloatField | Default: 0.0 | Actual disbursed amount |
| `amount_saved` | FloatField | Default: 0.0 | Savings amount |
| `bnpl_amount` | FloatField | Default: 0.0 | Buy Now Pay Later amount |
| `loan_type` | CharField(100) | Default: 'AJO' | Loan type (AJO, BNPL, etc.) |
| `application_type` | CharField(100) | Default: 'PRIMARY' | Application type |
| `status` | CharField(100) | Default: 'PENDING' | Loan status |
| `performance_status` | CharField(100) | Nullable | Performance status |
| `loan_behavior` | CharField(100) | Nullable | Loan behavior classification |
| `total_paid_amount` | FloatField | Default: 0.0 | Total amount repaid |
| `total_outstanding_balance` | FloatField | Default: 0.0 | Outstanding balance |
| `last_paid_amount` | FloatField | Default: 0.0 | Last payment amount |
| `interest_rate` | FloatField | Default: 0.0 | Interest rate percentage |
| `interest_amount` | FloatField | Default: 0.0 | Total interest amount |
| `processing_fee` | FloatField | Default: 0.0 | Processing fee charged |
| `nem_fee` | FloatField | Default: 0.0 | NEM fee |
| `repayment_amount` | FloatField | Default: 0.0 | Expected total repayment |
| `principal_repayment` | FloatField | Default: 0.0 | Principal repayment amount |
| `expected_repayment_count` | IntegerField | Default: 0 | Expected number of payments |
| `count_of_recorded_repayments` | IntegerField | Default: 0 | Actual payments recorded |
| `daily_repayment_amount` | FloatField | Default: 0.0 | Daily repayment amount |
| `principal_daily_repayment` | FloatField | Default: 0.0 | Daily principal repayment |
| `is_disbursed` | BooleanField | Default: False | Disbursement status flag |
| `date_disbursed` | DateTimeField | Nullable | Disbursement timestamp |
| `start_date` | DateField | Nullable | Loan start date |
| `end_date` | DateField | Nullable | Loan end date |
| `expected_end_date` | DateField | Nullable | Expected completion date |
| `actual_end_date` | DateField | Nullable | Actual completion date |
| `date_completed` | DateTimeField | Nullable | Completion timestamp |
| `tenor` | CharField(100) | Default: 'ONE_MONTH' | Loan tenor period |
| `tenor_in_days` | IntegerField | Default: 0 | Tenor in days |
| `repayment_type` | CharField(100) | Default: 'DAILY' | Repayment frequency |
| `loandisk_loan_id` | CharField(100) | Nullable | LoanDisk system loan ID |
| `unique_loan_ref` | CharField(300) | Nullable | Unique loan reference |
| `loan_ref` | CharField(300) | Default: uuid4 | Loan reference code |
| `escrow_amount` | FloatField | Default: 0.0 | Escrow amount |
| `escrow_settled` | BooleanField | Default: False | Escrow settlement status |
| `date_created` | DateTimeField | Auto-now-add | Record creation timestamp |
| `date_modified` | DateTimeField | Auto-now | Last modification timestamp |

**Loan Status Values**:
- `PENDING` - Awaiting approval
- `APPROVED` - Approved but not disbursed
- `DISBURSED` - Funds disbursed
- `COMPLETED` - Fully repaid
- `REJECTED` - Application rejected
- `DEFAULTED` - In default

**Tenor Values**:
- `ONE_MONTH` - 1 month
- `TWO_MONTHS` - 2 months
- `THREE_MONTHS` - 3 months
- etc.

**Repayment Type Values**:
- `DAILY` - Daily repayments
- `WEEKLY` - Weekly repayments
- `MONTHLY` - Monthly repayments

**Relationships**:
- **Many-to-One** with `CustomUser` (agent) via `agent` field
- **Many-to-One** with `AjoUser` (borrower) via `borrower` field
- **One-to-Many** with `AjoLoanRepayment` (loan repayments)
- **One-to-Many** with `AjoLoanSchedule` (payment schedule)

**ETL Filtering**:
- Loans are filtered by `is_disbursed=True` for full syncs
- Incremental syncs filter by `date_modified__gte=last_sync_timestamp`

---

### 7. AjoLoanRepayment (Repayments)

**Purpose**: Stores individual loan repayment transactions.

**Table Name**: `loans_ajoloanrepayment`

**Django Model**: `loans.models.AjoLoanRepayment`

**Key Fields**:

| Field Name | Type | Constraints | Description |
|------------|------|-------------|-------------|
| `id` | Integer | Primary Key | Unique repayment identifier |
| `agent` | ForeignKey | References CustomUser, CASCADE | Agent processing repayment |
| `borrower` | ForeignKey | References AjoUser, CASCADE | Borrower making payment |
| `ajo_loan` | ForeignKey | References AjoLoan, CASCADE | Associated loan |
| `repayment_type` | CharField(100) | | Payment method |
| `loan_amount` | FloatField | Default: 0.0 | Loan amount at time of payment |
| `repayment_amount` | FloatField | Default: 0.0 | Payment amount |
| `health_insurance` | FloatField | Default: 0.0 | Health insurance portion |
| `paid_date` | DateTimeField | Auto-now-add | Payment timestamp |
| `repayment_ref` | UUIDField | Default: uuid4 | Unique payment reference |
| `settlement_status` | BooleanField | Default: False | Settlement status |
| `repayment_meta_response` | TextField | Nullable | Payment gateway response |
| `ledger_amount` | FloatField | Default: 0.0 | Ledger amount |
| `comment` | CharField(300) | Nullable | Payment comment/note |
| `borrower_notified` | BooleanField | Default: False | Notification sent flag |
| `sms_body` | TextField | Nullable | SMS notification content |
| `sms_result` | TextField | Nullable | SMS delivery result |
| `marked_as` | CharField(255) | Nullable | Special marking (RECOVERY, REVERSAL) |
| `expected_recovery_amt` | FloatField | Default: 0.0 | Expected recovery amount |
| `applied_to_loan` | BooleanField | Nullable | Whether applied to loan balance |
| `date_created` | DateTimeField | Auto-now-add | Record creation timestamp |
| `date_modified` | DateTimeField | Auto-now | Last modification timestamp |

**Repayment Type Values**:
- `CASH` - Cash payment
- `TRANSFER` - Bank transfer
- `CARD` - Card payment
- `USSD` - USSD payment
- `DIRECT_DEBIT` - Direct debit

**Relationships**:
- **Many-to-One** with `CustomUser` (agent) via `agent` field
- **Many-to-One** with `AjoUser` (borrower) via `borrower` field
- **Many-to-One** with `AjoLoan` (loan) via `ajo_loan` field

**ETL Filtering**:
- Incremental syncs filter by `date_created__gte=last_sync_timestamp`
- Full syncs include all repayments

---

### 8. AjoLoanSchedule (Payment Schedule)

**Purpose**: Stores the payment schedule for loans with installment plans.

**Table Name**: `loans_ajoloanschedule`

**Django Model**: `loans.models.AjoLoanSchedule`

**Key Fields**:

| Field Name | Type | Constraints | Description |
|------------|------|-------------|-------------|
| `id` | Integer | Primary Key | Unique schedule entry identifier |
| `loan` | ForeignKey | References AjoLoan, CASCADE | Associated loan |
| `due_amount` | FloatField | Default: 0.0 | Amount due for this installment |
| `amount_left` | FloatField | Default: 0.0 | Remaining amount (calculated) |
| `paid_amount` | FloatField | Default: 0.0 | Amount paid for this installment |
| `due_date` | DateField | Nullable | Payment due date |
| `paid_date` | DateField | Nullable | Actual payment date |
| `fully_paid` | BooleanField | Default: False | Fully paid status |
| `fully_paid_date` | DateField | Nullable | Full payment date |
| `is_late` | BooleanField | Default: False | Late payment flag |
| `late_days` | IntegerField | Default: 0 | Number of days late |
| `date_created` | DateTimeField | Auto-now-add | Record creation timestamp |
| `date_modified` | DateTimeField | Auto-now | Last modification timestamp |

**Computed Properties**:
- `loan_status`: Returns 'paid', 'partially_paid', or 'pending' based on payment status

**Relationships**:
- **Many-to-One** with `AjoLoan` via `loan` field

---

## Entity Relationships

### Relationship Diagram

```
CustomUser (Officers/Agents)
    ├── One-to-Many → AjoUser (manages customers)
    ├── One-to-Many → AjoLoan (manages loans as agent)
    └── One-to-Many → AjoLoanRepayment (processes repayments)

AjoUser (Customers/Borrowers)
    ├── Many-to-One → CustomUser (managed by agent)
    ├── One-to-One → CustomUser (lite_user - optional)
    ├── One-to-Many → AjoLoan (takes loans)
    └── One-to-Many → AjoLoanRepayment (makes repayments)

AjoLoan (Loans)
    ├── Many-to-One → CustomUser (agent)
    ├── Many-to-One → AjoUser (borrower)
    ├── One-to-Many → AjoLoanRepayment (has repayments)
    └── One-to-Many → AjoLoanSchedule (has payment schedule)

AjoLoanRepayment (Repayments)
    ├── Many-to-One → CustomUser (agent)
    ├── Many-to-One → AjoUser (borrower)
    └── Many-to-One → AjoLoan (for loan)

ETLSyncHistory (Sync Operations)
    └── One-to-Many → ETLSyncError (has errors)
```

### Key Relationships

1. **CustomUser ↔ AjoUser**
   - **Agent Relationship**: `AjoUser.user` → `CustomUser` (Many-to-One)
     - An agent (CustomUser) manages many customers (AjoUser)
     - Access: `agent.ajouser_set.all()` or `customer.user`

   - **Lite User Relationship**: `AjoUser.lite_user` → `CustomUser` (One-to-One, Optional)
     - A customer can have a lite app account
     - Access: `customer.lite_user` or `lite_user.lite_users`

2. **CustomUser ↔ AjoLoan**
   - **Agent Relationship**: `AjoLoan.agent` → `CustomUser` (Many-to-One)
     - An agent manages many loans
     - Access: `agent.ajoloans_set.all()` or `loan.agent`

3. **AjoUser ↔ AjoLoan**
   - **Borrower Relationship**: `AjoLoan.borrower` → `AjoUser` (Many-to-One)
     - A customer can have many loans
     - Access: `customer.ajoloan_set.all()` or `loan.borrower`

4. **AjoLoan ↔ AjoLoanRepayment**
   - **Loan Repayments**: `AjoLoanRepayment.ajo_loan` → `AjoLoan` (Many-to-One)
     - A loan has many repayments
     - Access: `loan.ajoloanrepayment_set.all()` or `repayment.ajo_loan`

5. **ETLSyncHistory ↔ ETLSyncError**
   - **Sync Errors**: `ETLSyncError.sync_history` → `ETLSyncHistory` (Many-to-One)
     - A sync operation can have many errors
     - Access: `sync.errors.all()` or `error.sync_history`

---

## Database Indexes

### ETLSyncHistory Indexes
- `sync_id` (unique, B-tree)
- `started_at` (descending, B-tree)
- `status` (B-tree)
- `sync_type` (B-tree)

### ETLSyncError Indexes
- `(entity_type, entity_id)` (composite, B-tree)
- `resolved` (B-tree)
- `created_at` (descending, B-tree)

### CustomUser Indexes
- `email` (unique, B-tree)
- `customer_user_id` (unique, B-tree)
- `user_phone` (unique, B-tree)
- `call_agent_id` (unique, B-tree)

### AjoUser Indexes
- `user_id` (foreign key, B-tree)
- `lite_user_id` (unique, B-tree)
- `phone_number` (unique, B-tree)
- `onboarding_verified` (B-tree)
- `bvn_verified` (B-tree)
- `onboarding_complete` (B-tree)
- `card_issued` (B-tree)
- `need_card` (B-tree)

### AjoLoan Indexes
- `agent_id` (foreign key, B-tree)
- `borrower_id` (foreign key, B-tree)
- `is_disbursed` (B-tree)
- `status` (B-tree)

### AjoLoanRepayment Indexes
- `agent_id` (foreign key, B-tree)
- `borrower_id` (foreign key, B-tree)
- `ajo_loan_id` (foreign key, B-tree)

---

## ETL Data Flow

### 1. Customer Sync
```
AjoUser (source)
    ↓ Extract
    ↓ Filter: onboarding_complete=True
    ↓ Transform: CustomerTransformer
    ↓ Load: POST /etl/customers
    ↓ Track: ETLSyncHistory
```

### 2. Officer Sync
```
CustomUser (source)
    ↓ Extract
    ↓ Filter: user_type IN [AGENT, STAFF_AGENT, PROSPER_AGENT, DMO_AGENT, AJO_AGENT, RECOVERY_AGENT]
    ↓ Transform: OfficerTransformer
    ↓ Load: POST /etl/officers
    ↓ Track: ETLSyncHistory
```

### 3. Loan Sync
```
AjoLoan (source)
    ↓ Extract
    ↓ Filter: is_disbursed=True (full) OR date_modified__gte=last_sync (incremental)
    ↓ Transform: LoanTransformer
    ↓ Load (chunked): POST /etl/sync (batches of 100)
    ↓ Track: ETLSyncHistory
```

### 4. Repayment Sync
```
AjoLoanRepayment (source)
    ↓ Extract
    ↓ Filter: date_created__gte=last_sync (incremental) OR all (full)
    ↓ Transform: RepaymentTransformer
    ↓ Load (chunked): POST /etl/sync (batches of 200)
    ↓ Track: ETLSyncHistory
```

---

## Performance Considerations

### Memory Optimization (v1.0.4+)

The ETL system uses **chunked processing** to handle large datasets:

1. **QuerySet Iteration**: Uses `queryset.iterator(chunk_size=N)` instead of `list(queryset)`
2. **Batch Processing**: Processes records in configurable batches
3. **Memory Cleanup**: Clears processed chunks from memory before loading next batch

**Configuration**:
- Loan batch size: 100 (configurable via `ETL_BATCH_SIZE_LOANS`)
- Repayment batch size: 200 (configurable via `ETL_BATCH_SIZE_REPAYMENTS`)

**Memory Usage**:
- Before optimization (v1.0.3): ~6-8 GB for full sync
- After optimization (v1.0.4+): ~50 MB for full sync

### Query Optimization

1. **Select Related**: Pre-fetches related objects to reduce queries
   ```python
   loans = AjoLoan.objects.select_related('agent', 'borrower', 'borrower__user')
   ```

2. **Prefetch Related**: Pre-fetches many-to-many and reverse foreign keys
   ```python
   loans = AjoLoan.objects.prefetch_related('ajoloanrepayment_set')
   ```

3. **Count Queries**: Uses `.count()` instead of `len(queryset)` to avoid loading data

4. **Indexed Filters**: All filter fields have database indexes for fast lookups

---

## Version History

- **v1.0.0** - Initial ETL system with all tables
- **v1.0.1** - Fixed QuerySet import error
- **v1.0.2** - Fixed config value parsing
- **v1.0.3** - Fixed officer extraction (user_type vs account_type)
- **v1.0.4** - Implemented chunked processing for memory optimization
- **v1.0.5** - Fixed chunk processing API method calls

---

## Related Documentation

- `CHANGELOG.md` - Version history and changes
- `MEMORY_OPTIMIZATION_FIX.md` - Memory optimization details
- `OFFICER_EXTRACTION_FIX.md` - Officer extraction fix details
- `TROUBLESHOOTING.md` - Common issues and solutions

---

**Last Updated**: 2025-11-03
**ETL Version**: 1.0.5

