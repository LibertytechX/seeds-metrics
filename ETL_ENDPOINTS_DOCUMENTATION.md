# Seeds Metrics ETL API Documentation

**Version:** 1.0.0
**Last Updated:** November 1, 2025
**Base URL:** `https://metrics.seedsandpennies.com/api/v1`

---

## Table of Contents

1. [Overview](#overview)
2. [Authentication](#authentication)
3. [Common Response Formats](#common-response-formats)
4. [ETL Endpoints](#etl-endpoints)
   - [Create Customer](#1-create-customer)
   - [Create Officer](#2-create-officer)
   - [Create Loan](#3-create-loan)
   - [Create Repayment](#4-create-repayment)
   - [Batch Sync](#5-batch-sync)
5. [Error Handling](#error-handling)
6. [Data Validation Rules](#data-validation-rules)
7. [Best Practices](#best-practices)

---

## Overview

The Seeds Metrics ETL API provides endpoints for the main loans backend system to send loan data, customer data, repayment data, and officer data to the Seeds Metrics analytics platform. These endpoints are designed for **data ingestion only** and support both individual record creation and batch synchronization.

### Key Features

- **Individual Record Creation**: Create single customers, officers, loans, or repayments
- **Batch Synchronization**: Sync multiple loans and repayments in a single request
- **Automatic Computation**: Loan metrics are automatically calculated when repayments are created
- **Idempotency**: Duplicate loan IDs are rejected with clear error messages
- **Wave Categorization**: Support for loan wave/cohort tracking (Wave 1, Wave 2, etc.)

---

## Authentication

**Current Status:** No authentication required (internal API)

> **Note:** For production deployment, consider implementing API key authentication or OAuth 2.0.

---

## Common Response Formats

### Success Response

```json
{
  "status": "success",
  "message": "Operation completed successfully",
  "data": {
    // Response data specific to the endpoint
  }
}
```

### Error Response

```json
{
  "status": "error",
  "error": {
    "code": "ERROR_CODE",
    "message": "Human-readable error message",
    "details": {
      // Additional error context
    }
  }
}
```

---

## ETL Endpoints

### 1. Create Customer

Create a new customer record in the system.

#### Endpoint

```
POST /api/v1/etl/customers
```

#### Request Headers

```
Content-Type: application/json
```

#### Request Body

| Field | Type | Required | Description | Validation |
|-------|------|----------|-------------|------------|
| `customer_id` | string | Yes | Unique customer identifier | Must be unique |
| `customer_name` | string | Yes | Full name of the customer | Non-empty string |
| `customer_phone` | string | No | Customer phone number | E.164 format recommended |
| `customer_email` | string | No | Customer email address | Valid email format |
| `date_of_birth` | string | No | Date of birth | Format: YYYY-MM-DD |
| `gender` | string | No | Customer gender | e.g., "Male", "Female" |
| `state` | string | No | State of residence | Nigerian state name |
| `lga` | string | No | Local Government Area | LGA name |
| `address` | string | No | Residential address | Full address |
| `kyc_status` | string | No | KYC verification status | e.g., "Verified", "Pending" |
| `kyc_verified_date` | string | No | KYC verification date | Format: YYYY-MM-DD |

#### Example Request

```json
{
  "customer_id": "CUST20240567",
  "customer_name": "Adebayo Oluwaseun",
  "customer_phone": "+234-803-456-7890",
  "customer_email": "adebayo@example.com",
  "date_of_birth": "1985-06-15",
  "gender": "Male",
  "state": "Lagos",
  "lga": "Ikeja",
  "address": "123 Allen Avenue, Ikeja, Lagos",
  "kyc_status": "Verified",
  "kyc_verified_date": "2024-10-01"
}
```

#### Success Response (201 Created)

```json
{
  "status": "success",
  "message": "Customer created successfully",
  "data": {
    "customer_id": "CUST20240567"
  }
}
```

#### Error Responses

**400 Bad Request - Validation Error**
```json
{
  "status": "error",
  "error": {
    "code": "VALIDATION_ERROR",
    "message": "Invalid request payload",
    "details": {
      "error": "Key: 'CustomerInput.CustomerName' Error:Field validation for 'CustomerName' failed on the 'required' tag"
    }
  }
}
```

**500 Internal Server Error - Database Error**
```json
{
  "status": "error",
  "error": {
    "code": "DATABASE_ERROR",
    "message": "Failed to create customer",
    "details": {
      "error": "duplicate key value violates unique constraint"
    }
  }
}
```

---

### 2. Create Officer

Create a new loan officer record in the system.

#### Endpoint

```
POST /api/v1/etl/officers
```

#### Request Headers

```
Content-Type: application/json
```

#### Request Body

| Field | Type | Required | Description | Validation |
|-------|------|----------|-------------|------------|
| `officer_id` | string | Yes | Unique officer identifier | Must be unique |
| `officer_name` | string | Yes | Full name of the officer | Non-empty string |
| `officer_phone` | string | No | Officer phone number | E.164 format recommended |
| `region` | string | Yes | Geographic region | e.g., "South West", "North Central" |
| `branch` | string | Yes | Branch name | e.g., "Lagos Main", "Abuja Branch" |
| `employment_status` | string | No | Employment status | e.g., "Active", "Inactive" |
| `hire_date` | string | No | Date of hire | Format: YYYY-MM-DD |

#### Example Request

```json
{
  "officer_id": "OFF2024012",
  "officer_name": "Sarah Johnson",
  "officer_phone": "+234-803-987-6543",
  "region": "South West",
  "branch": "Lagos Main",
  "employment_status": "Active",
  "hire_date": "2023-01-15"
}
```

#### Success Response (201 Created)

```json
{
  "status": "success",
  "message": "Officer created successfully",
  "data": {
    "officer_id": "OFF2024012"
  }
}
```

#### Error Responses

Same error format as Create Customer endpoint.

---

### 3. Create Loan

Create a new loan record in the system. This is the primary endpoint for loan data ingestion.

#### Endpoint

```
POST /api/v1/etl/loans
```

#### Request Headers

```
Content-Type: application/json
```

#### Request Body

| Field | Type | Required | Description | Validation | Default |
|-------|------|----------|-------------|------------|---------|
| `loan_id` | string | Yes | Unique loan identifier | Must be unique | - |
| `customer_id` | string | Yes | Customer identifier | Must exist in customers table | - |
| `customer_name` | string | Yes | Customer full name | Non-empty string | - |
| `customer_phone` | string | No | Customer phone number | E.164 format recommended | null |
| `officer_id` | string | Yes | Loan officer identifier | - | - |
| `officer_name` | string | Yes | Officer full name | Non-empty string | - |
| `officer_phone` | string | No | Officer phone number | E.164 format recommended | null |
| `region` | string | Yes | Geographic region | e.g., "South West" | - |
| `branch` | string | Yes | Branch name | e.g., "Lagos Main" | - |
| `state` | string | No | State location | Nigerian state name | null |
| `loan_amount` | number | Yes | Principal loan amount | Must be > 0 | - |
| `repayment_amount` | number | No | Expected repayment amount | Must be >= loan_amount | null |
| `disbursement_date` | string | Yes | Date loan was disbursed | Format: YYYY-MM-DD | - |
| `maturity_date` | string | Yes | Loan maturity date | Format: YYYY-MM-DD, must be after disbursement_date | - |
| `loan_term_days` | integer | Yes | Loan term in days | Must be > 0 | - |
| `interest_rate` | number | No | Annual interest rate | Decimal format (e.g., 0.15 for 15%) | null |
| `fee_amount` | number | No | Processing fee amount | Must be >= 0 | null |
| `channel` | string | Yes | Disbursement channel | e.g., "Direct", "Agent" | - |
| `channel_partner` | string | No | Partner name if channel is Agent | e.g., "Kuda Bank" | null |
| `status` | string | Yes | Current loan status | "Active", "Closed", "Written Off" | - |
| `closed_date` | string | No | Date loan was closed | Format: YYYY-MM-DD | null |
| `wave` | string | No | Loan wave/cohort | "Wave 1" or "Wave 2" | "Wave 2" |

#### Important Notes

- **Interest Rate Format**: Send as decimal (e.g., `0.15` for 15%, `0.20` for 20%)
- **Wave Field**: Optional field for categorizing loans into cohorts. Defaults to "Wave 2" if not provided
- **Duplicate Prevention**: Returns 409 Conflict if loan_id already exists
- **Automatic Computation**: Computed fields (DPD, outstanding amounts, etc.) are calculated automatically

#### Example Request

```json
{
  "loan_id": "LN2024001234",
  "customer_id": "CUST20240567",
  "customer_name": "Adebayo Oluwaseun",
  "customer_phone": "+234-803-456-7890",
  "officer_id": "OFF2024012",
  "officer_name": "Sarah Johnson",
  "officer_phone": "+234-803-987-6543",
  "region": "South West",
  "branch": "Lagos Main",
  "state": "Lagos",
  "loan_amount": 500000.00,
  "repayment_amount": 575000.00,
  "disbursement_date": "2024-10-15",
  "maturity_date": "2025-04-15",
  "loan_term_days": 180,
  "interest_rate": 0.15,
  "fee_amount": 25000.00,
  "channel": "Direct",
  "channel_partner": null,
  "status": "Active",
  "closed_date": null,
  "wave": "Wave 1"
}
```

#### Success Response (201 Created)

```json
{
  "status": "success",
  "message": "Loan created successfully",
  "data": {
    "loan_id": "LN2024001234"
  }
}
```

#### Error Responses

**400 Bad Request - Validation Error**
```json
{
  "status": "error",
  "error": {
    "code": "VALIDATION_ERROR",
    "message": "Invalid request payload",
    "details": {
      "error": "Key: 'LoanInput.LoanAmount' Error:Field validation for 'LoanAmount' failed on the 'required' tag"
    }
  }
}
```

**409 Conflict - Duplicate Loan ID**
```json
{
  "status": "error",
  "error": {
    "code": "DUPLICATE_LOAN_ID",
    "message": "Loan ID already exists",
    "details": {
      "loan_id": "LN2024001234",
      "customer_name": "Adebayo Oluwaseun"
    }
  }
}
```

**500 Internal Server Error - Database Error**
```json
{
  "status": "error",
  "error": {
    "code": "DATABASE_ERROR",
    "message": "Failed to create loan",
    "details": {
      "error": "invalid input syntax for type numeric"
    }
  }
}
```

---

### 4. Create Repayment

Create a new repayment record for an existing loan. This endpoint automatically triggers recalculation of loan computed fields.

#### Endpoint

```
POST /api/v1/etl/repayments
```

#### Request Headers

```
Content-Type: application/json
```

#### Request Body

| Field | Type | Required | Description | Validation | Default |
|-------|------|----------|-------------|------------|---------|
| `repayment_id` | string | Yes | Unique repayment identifier | Must be unique | - |
| `loan_id` | string | Yes | Associated loan identifier | Loan must exist | - |
| `payment_date` | string | Yes | Date payment was received | Format: YYYY-MM-DD | - |
| `payment_amount` | number | Yes | Total payment amount | Must equal sum of components | - |
| `principal_paid` | number | Yes | Principal portion | Must be >= 0 | - |
| `interest_paid` | number | Yes | Interest portion | Must be >= 0 | - |
| `fees_paid` | number | Yes | Fees portion | Must be >= 0 | - |
| `penalty_paid` | number | No | Penalty/late fee portion | Must be >= 0 | 0.00 |
| `payment_method` | string | Yes | Payment method | e.g., "Bank Transfer", "Cash", "Card Payment" | - |
| `payment_reference` | string | No | Transaction reference | Unique transaction ID | null |
| `payment_channel` | string | No | Payment channel | e.g., "Mobile App", "Web Portal", "USSD" | null |
| `dpd_at_payment` | integer | No | Days past due at payment | Must be >= 0 | 0 |
| `is_backdated` | boolean | No | Whether payment is backdated | true/false | false |
| `is_reversed` | boolean | No | Whether payment is reversed | true/false | false |
| `reversal_date` | string | No | Date of reversal | Format: YYYY-MM-DD | null |
| `reversal_reason` | string | No | Reason for reversal | Free text | null |
| `waiver_amount` | number | No | Amount waived | Must be >= 0 | 0.00 |
| `waiver_type` | string | No | Type of waiver | e.g., "Interest", "Penalty" | null |
| `waiver_approved_by` | string | No | Approver identifier | Officer ID or name | null |

#### Important Notes

- **Payment Amount Validation**: `payment_amount` must equal `principal_paid + interest_paid + fees_paid + penalty_paid`
- **Loan Existence**: The loan must exist before creating a repayment
- **Automatic Updates**: Creating a repayment automatically updates loan computed fields:
  - Current DPD
  - Outstanding amounts (principal, interest, fees)
  - Total repayments
  - Days since last repayment
  - Repayment delay rate
  - FIMR tagging
  - Early indicator tagging

#### Example Request

```json
{
  "repayment_id": "REP2024005678",
  "loan_id": "LN2024001234",
  "payment_date": "2024-11-01",
  "payment_amount": 100000.00,
  "principal_paid": 80000.00,
  "interest_paid": 15000.00,
  "fees_paid": 5000.00,
  "penalty_paid": 0.00,
  "payment_method": "Bank Transfer",
  "payment_reference": "TXN20241101123456",
  "payment_channel": "Mobile App",
  "dpd_at_payment": 0,
  "is_backdated": false,
  "is_reversed": false,
  "reversal_date": null,
  "reversal_reason": null,
  "waiver_amount": 0.00,
  "waiver_type": null,
  "waiver_approved_by": null
}
```

#### Success Response (201 Created)

```json
{
  "status": "success",
  "message": "Repayment created successfully. Loan computed fields will be updated automatically.",
  "data": {
    "repayment_id": "REP2024005678",
    "loan_id": "LN2024001234"
  }
}
```

#### Error Responses

**400 Bad Request - Validation Error**
```json
{
  "status": "error",
  "error": {
    "code": "VALIDATION_ERROR",
    "message": "Invalid request payload",
    "details": {
      "error": "Key: 'RepaymentInput.PaymentAmount' Error:Field validation for 'PaymentAmount' failed on the 'required' tag"
    }
  }
}
```

**400 Bad Request - Loan Not Found**
```json
{
  "status": "error",
  "error": {
    "code": "LOAN_NOT_FOUND",
    "message": "Loan with ID LN2024001234 does not exist"
  }
}
```

**500 Internal Server Error - Database Error**
```json
{
  "status": "error",
  "error": {
    "code": "DATABASE_ERROR",
    "message": "Failed to create repayment",
    "details": {
      "error": "duplicate key value violates unique constraint"
    }
  }
}
```

---

### 5. Batch Sync

Synchronize multiple loans and repayments in a single batch request. This is the recommended endpoint for bulk data synchronization.

#### Endpoint

```
POST /api/v1/etl/sync
```

#### Request Headers

```
Content-Type: application/json
```

#### Request Body

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `sync_timestamp` | string | Yes | Timestamp of sync operation (ISO 8601 format) |
| `sync_type` | string | Yes | Type of sync: "incremental" or "full" |
| `data` | object | Yes | Container for loans and repayments |
| `data.loans` | array | No | Array of loan objects (same structure as Create Loan) |
| `data.repayments` | array | No | Array of repayment objects (same structure as Create Repayment) |
| `metadata` | object | No | Sync metadata |
| `metadata.total_loans` | integer | No | Total number of loans in sync |
| `metadata.total_repayments` | integer | No | Total number of repayments in sync |
| `metadata.source_system` | string | No | Source system identifier |
| `metadata.etl_version` | string | No | ETL version number |

#### Important Notes

- **Partial Success**: The endpoint processes all records and returns partial success if some fail
- **Error Tracking**: Failed records are reported in the `errors` array with details
- **Automatic Computation**: Loan computed fields are updated after all repayments are processed
- **Recommended Frequency**: Sync every 15 minutes for incremental updates
- **Wave Field**: Include the `wave` field in loan objects for proper categorization

#### Example Request

```json
{
  "sync_timestamp": "2024-10-18T14:30:00Z",
  "sync_type": "incremental",
  "data": {
    "loans": [
      {
        "loan_id": "LN2024001235",
        "customer_id": "CUST20240568",
        "customer_name": "Chioma Nwosu",
        "customer_phone": "+234-805-123-4567",
        "officer_id": "OFF2024012",
        "officer_name": "Sarah Johnson",
        "officer_phone": "+234-803-987-6543",
        "region": "South East",
        "branch": "Enugu Branch",
        "state": "Enugu",
        "loan_amount": 750000.00,
        "repayment_amount": 862500.00,
        "disbursement_date": "2024-10-16",
        "maturity_date": "2025-04-16",
        "loan_term_days": 180,
        "interest_rate": 0.15,
        "fee_amount": 37500.00,
        "channel": "Agent",
        "channel_partner": "Kuda Bank",
        "status": "Active",
        "closed_date": null,
        "wave": "Wave 2"
      },
      {
        "loan_id": "LN2024001236",
        "customer_id": "CUST20240569",
        "customer_name": "Ibrahim Musa",
        "customer_phone": "+234-806-234-5678",
        "officer_id": "OFF2024013",
        "officer_name": "Michael Chen",
        "officer_phone": "+234-804-111-2222",
        "region": "North Central",
        "branch": "Abuja Branch",
        "state": "FCT",
        "loan_amount": 1000000.00,
        "repayment_amount": 1150000.00,
        "disbursement_date": "2024-10-17",
        "maturity_date": "2025-04-17",
        "loan_term_days": 180,
        "interest_rate": 0.15,
        "fee_amount": 50000.00,
        "channel": "Direct",
        "channel_partner": null,
        "status": "Active",
        "closed_date": null,
        "wave": "Wave 2"
      }
    ],
    "repayments": [
      {
        "repayment_id": "REP2024005679",
        "loan_id": "LN2024001235",
        "payment_date": "2024-11-02",
        "payment_amount": 125000.00,
        "principal_paid": 100000.00,
        "interest_paid": 18750.00,
        "fees_paid": 6250.00,
        "penalty_paid": 0.00,
        "payment_method": "Card Payment",
        "payment_reference": "TXN20241102987654",
        "payment_channel": "Web Portal",
        "dpd_at_payment": 0,
        "is_backdated": false,
        "is_reversed": false,
        "reversal_date": null,
        "reversal_reason": null,
        "waiver_amount": 0.00,
        "waiver_type": null,
        "waiver_approved_by": null
      },
      {
        "repayment_id": "REP2024005680",
        "loan_id": "LN2024001236",
        "payment_date": "2024-11-03",
        "payment_amount": 150000.00,
        "principal_paid": 120000.00,
        "interest_paid": 22500.00,
        "fees_paid": 7500.00,
        "penalty_paid": 0.00,
        "payment_method": "Bank Transfer",
        "payment_reference": "TXN20241103456789",
        "payment_channel": "Mobile App",
        "dpd_at_payment": 0,
        "is_backdated": false,
        "is_reversed": false,
        "reversal_date": null,
        "reversal_reason": null,
        "waiver_amount": 0.00,
        "waiver_type": null,
        "waiver_approved_by": null
      }
    ]
  },
  "metadata": {
    "total_loans": 2,
    "total_repayments": 2,
    "source_system": "main_backend",
    "etl_version": "1.0.0"
  }
}
```

#### Success Response (200 OK)

```json
{
  "status": "success",
  "sync_id": "550e8400-e29b-41d4-a716-446655440000",
  "timestamp": "2024-10-18T14:30:15Z",
  "results": {
    "loans": {
      "inserted": 2,
      "updated": 0,
      "failed": 0
    },
    "repayments": {
      "inserted": 2,
      "updated": 0,
      "failed": 0
    }
  },
  "computed_fields_updated": {
    "loans_affected": 2,
    "computation_time_ms": 45
  },
  "next_sync_recommended": "2024-10-18T14:45:00Z",
  "errors": []
}
```

#### Partial Success Response (207 Multi-Status)

```json
{
  "status": "partial_success",
  "sync_id": "550e8400-e29b-41d4-a716-446655440001",
  "timestamp": "2024-10-18T14:30:15Z",
  "results": {
    "loans": {
      "inserted": 1,
      "updated": 0,
      "failed": 1
    },
    "repayments": {
      "inserted": 2,
      "updated": 0,
      "failed": 0
    }
  },
  "computed_fields_updated": {
    "loans_affected": 2,
    "computation_time_ms": 38
  },
  "next_sync_recommended": "2024-10-18T14:45:00Z",
  "errors": [
    {
      "entity_type": "loan",
      "entity_id": "LN2024001236",
      "error_code": "CREATE_FAILED",
      "error_message": "duplicate key value violates unique constraint \"loans_pkey\""
    }
  ]
}
```

#### Error Response (400 Bad Request)

```json
{
  "status": "error",
  "sync_id": "550e8400-e29b-41d4-a716-446655440002",
  "timestamp": "2024-10-18T14:30:15Z",
  "results": {
    "loans": {
      "inserted": 0,
      "updated": 0,
      "failed": 2
    },
    "repayments": {
      "inserted": 0,
      "updated": 0,
      "failed": 2
    }
  },
  "computed_fields_updated": {
    "loans_affected": 0,
    "computation_time_ms": 12
  },
  "next_sync_recommended": "2024-10-18T14:45:00Z",
  "errors": [
    {
      "entity_type": "loan",
      "entity_id": "LN2024001235",
      "error_code": "CREATE_FAILED",
      "error_message": "invalid input syntax for type numeric: \"invalid\""
    },
    {
      "entity_type": "loan",
      "entity_id": "LN2024001236",
      "error_code": "CREATE_FAILED",
      "error_message": "duplicate key value violates unique constraint"
    },
    {
      "entity_type": "repayment",
      "entity_id": "REP2024005679",
      "error_code": "CREATE_FAILED",
      "error_message": "loan with ID LN2024001235 does not exist"
    },
    {
      "entity_type": "repayment",
      "entity_id": "REP2024005680",
      "error_code": "CREATE_FAILED",
      "error_message": "loan with ID LN2024001236 does not exist"
    }
  ]
}
```

---

## Error Handling

### Error Codes

| Error Code | HTTP Status | Description | Resolution |
|------------|-------------|-------------|------------|
| `VALIDATION_ERROR` | 400 | Request payload validation failed | Check required fields and data types |
| `DUPLICATE_LOAN_ID` | 409 | Loan ID already exists | Use a different loan_id or update existing loan |
| `LOAN_NOT_FOUND` | 400 | Referenced loan does not exist | Create the loan before creating repayments |
| `DATABASE_ERROR` | 500 | Database operation failed | Check database connectivity and data integrity |
| `CREATE_FAILED` | Various | Entity creation failed (batch sync) | Check individual error messages in errors array |

### Error Response Structure

All error responses follow this structure:

```json
{
  "status": "error",
  "error": {
    "code": "ERROR_CODE",
    "message": "Human-readable error message",
    "details": {
      // Additional context (optional)
    }
  }
}
```

---

## Data Validation Rules

### General Rules

1. **Date Format**: All dates must be in `YYYY-MM-DD` format (e.g., "2024-10-15")
2. **Decimal Numbers**: Use decimal format for monetary values (e.g., 500000.00)
3. **Phone Numbers**: E.164 format recommended (e.g., "+234-803-456-7890")
4. **Required Fields**: Fields marked as "required" must be present and non-null
5. **Unique Identifiers**: IDs must be unique across the system

### Loan-Specific Rules

1. **Loan Amount**: Must be greater than 0
2. **Repayment Amount**: Must be greater than or equal to loan_amount
3. **Maturity Date**: Must be after disbursement_date
4. **Loan Term Days**: Must be greater than 0
5. **Interest Rate**: Decimal format (0.15 for 15%, not 15)
6. **Fee Amount**: Must be greater than or equal to 0
7. **Status**: Must be one of: "Active", "Closed", "Written Off"
8. **Wave**: Must be one of: "Wave 1", "Wave 2" (defaults to "Wave 2" if not provided)

### Repayment-Specific Rules

1. **Payment Amount**: Must equal `principal_paid + interest_paid + fees_paid + penalty_paid`
2. **Component Amounts**: All payment components must be >= 0
3. **Loan Existence**: The loan_id must exist in the loans table
4. **Payment Date**: Should not be before loan disbursement_date

---

## Best Practices

### 1. Use Batch Sync for Bulk Operations

For syncing multiple records, use the `/api/v1/etl/sync` endpoint instead of making individual API calls. This:
- Reduces network overhead
- Provides atomic-like behavior
- Returns comprehensive error reporting
- Optimizes database operations

### 2. Implement Retry Logic

Implement exponential backoff retry logic for failed requests:

```python
import time
import requests

def sync_with_retry(payload, max_retries=3):
    for attempt in range(max_retries):
        try:
            response = requests.post(
                "https://metrics.seedsandpennies.com/api/v1/etl/sync",
                json=payload,
                timeout=30
            )
            if response.status_code in [200, 207]:
                return response.json()
            elif response.status_code >= 500:
                # Server error, retry
                time.sleep(2 ** attempt)
                continue
            else:
                # Client error, don't retry
                return response.json()
        except requests.exceptions.RequestException as e:
            if attempt == max_retries - 1:
                raise
            time.sleep(2 ** attempt)
```

### 3. Validate Data Before Sending

Validate data on the client side before sending to reduce errors:

```python
def validate_loan(loan_data):
    required_fields = [
        'loan_id', 'customer_id', 'customer_name',
        'officer_id', 'officer_name', 'region', 'branch',
        'loan_amount', 'disbursement_date', 'maturity_date',
        'loan_term_days', 'channel', 'status'
    ]

    for field in required_fields:
        if field not in loan_data or loan_data[field] is None:
            raise ValueError(f"Missing required field: {field}")

    # Validate wave field if provided
    if 'wave' in loan_data and loan_data['wave'] not in ['Wave 1', 'Wave 2', None]:
        raise ValueError(f"Invalid wave value: {loan_data['wave']}")

    return True
```

### 4. Handle Partial Success

When using batch sync, always check for partial success and handle failed records:

```python
response = sync_with_retry(payload)

if response['status'] == 'partial_success':
    # Log failed records
    for error in response['errors']:
        print(f"Failed {error['entity_type']} {error['entity_id']}: {error['error_message']}")

    # Retry failed records individually or log for manual review
```

### 5. Include Wave Field for New Loans

Always include the `wave` field when creating new loans to ensure proper categorization:

```python
loan_data = {
    "loan_id": "LN2024001234",
    # ... other fields ...
    "wave": "Wave 2"  # or "Wave 1" based on your business logic
}
```

### 6. Monitor Sync Performance

Track sync performance metrics:
- Total records synced
- Success/failure rates
- Computation time
- API response times

### 7. Sync Frequency Recommendations

- **Incremental Sync**: Every 15 minutes for new/updated records
- **Full Sync**: Daily at off-peak hours (e.g., 2 AM)
- **Real-time**: Use individual endpoints for critical updates

---

## Support and Contact

For technical support or questions about the ETL API:

- **Email**: support@seedsandpennies.com
- **Documentation**: https://metrics.seedsandpennies.com/docs
- **API Status**: https://status.seedsandpennies.com

---

**Document Version:** 1.0.0
**Last Updated:** November 1, 2025
**Maintained by:** Seeds & Pennies Engineering Team


