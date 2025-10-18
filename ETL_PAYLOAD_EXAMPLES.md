# ETL Payload Examples

## 📋 Complete JSON Payloads for ETL Integration

This document provides complete, production-ready JSON payload examples for integrating the main business backend with the analytics service.

---

## 1️⃣ Add New Loan - Complete Payload

### **JSON Payload:**

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
  "disbursement_date": "2024-10-15",
  "maturity_date": "2025-04-15",
  "loan_term_days": 180,
  "interest_rate": 0.1500,
  "fee_amount": 25000.00,
  "channel": "Direct",
  "channel_partner": null,
  "status": "Active",
  "closed_date": null
}
```

### **Field Descriptions:**

| Field | Type | Required | Description | Example Values |
|-------|------|----------|-------------|----------------|
| `loan_id` | string | ✅ Yes | Unique loan identifier | "LN2024001234" |
| `customer_id` | string | ✅ Yes | Customer reference ID | "CUST20240567" |
| `customer_name` | string | ✅ Yes | Full customer name | "Adebayo Oluwaseun" |
| `customer_phone` | string | ❌ No | Customer phone (Nigerian format) | "+234-803-456-7890" |
| `officer_id` | string | ✅ Yes | Loan officer ID | "OFF2024012" |
| `officer_name` | string | ✅ Yes | Loan officer name | "Sarah Johnson" |
| `officer_phone` | string | ❌ No | Officer phone | "+234-803-987-6543" |
| `region` | string | ✅ Yes | Geographic region | "South West", "North Central" |
| `branch` | string | ✅ Yes | Branch name | "Lagos Main", "Abuja Branch" |
| `state` | string | ❌ No | State/province | "Lagos", "Abuja" |
| `loan_amount` | decimal | ✅ Yes | Principal amount (NGN) | 500000.00 |
| `disbursement_date` | date | ✅ Yes | Disbursement date (YYYY-MM-DD) | "2024-10-15" |
| `maturity_date` | date | ✅ Yes | Maturity date (YYYY-MM-DD) | "2025-04-15" |
| `loan_term_days` | integer | ✅ Yes | Loan term in days | 180, 90, 365 |
| `interest_rate` | decimal | ❌ No | Annual interest rate (0.15 = 15%) | 0.1500, 0.2000 |
| `fee_amount` | decimal | ❌ No | Total fees (NGN) | 25000.00 |
| `channel` | string | ✅ Yes | Distribution channel | "Direct", "Agent", "Partner" |
| `channel_partner` | string | ❌ No | Partner name (if applicable) | "Kuda Bank", null |
| `status` | string | ✅ Yes | Loan status | "Active", "Closed", "Written Off" |
| `closed_date` | date | ❌ No | Closure date (if closed) | "2025-03-20", null |

### **⚠️ CRITICAL - Do NOT Include These Fields:**

The following fields are **computed by the analytics service** and should **NOT** be sent from the main backend:

❌ `current_dpd`  
❌ `max_dpd_ever`  
❌ `first_payment_missed`  
❌ `first_payment_due_date`  
❌ `first_payment_received_date`  
❌ `principal_outstanding`  
❌ `interest_outstanding`  
❌ `fees_outstanding`  
❌ `total_outstanding`  
❌ `total_principal_paid`  
❌ `total_interest_paid`  
❌ `total_fees_paid`  
❌ `fimr_tagged`  
❌ `early_indicator_tagged`  

**Why?** These fields are automatically calculated from repayments data to reduce compute load on the main business server.

---

## 2️⃣ Add New Repayment - Complete Payload

### **JSON Payload:**

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

### **Field Descriptions:**

| Field | Type | Required | Description | Example Values |
|-------|------|----------|-------------|----------------|
| `repayment_id` | string | ✅ Yes | Unique repayment identifier | "REP2024005678" |
| `loan_id` | string | ✅ Yes | Associated loan ID | "LN2024001234" |
| `payment_date` | date | ✅ Yes | Payment date (YYYY-MM-DD) | "2024-11-01" |
| `payment_amount` | decimal | ✅ Yes | Total payment amount (NGN) | 100000.00 |
| `principal_paid` | decimal | ✅ Yes | Principal portion (NGN) | 80000.00 |
| `interest_paid` | decimal | ✅ Yes | Interest portion (NGN) | 15000.00 |
| `fees_paid` | decimal | ✅ Yes | Fees portion (NGN) | 5000.00 |
| `penalty_paid` | decimal | ❌ No | Penalty portion (NGN) | 0.00, 2000.00 |
| `payment_method` | string | ✅ Yes | Payment method | "Bank Transfer", "Cash", "Card" |
| `payment_reference` | string | ❌ No | Payment reference/transaction ID | "TXN20241101123456" |
| `payment_channel` | string | ❌ No | Payment channel | "Mobile App", "USSD", "Branch" |
| `dpd_at_payment` | integer | ❌ No | DPD when payment was made | 0, 5, 15 |
| `is_backdated` | boolean | ❌ No | Is this a backdated payment? | true, false |
| `is_reversed` | boolean | ✅ Yes | Is this payment reversed? | true, false |
| `reversal_date` | date | ❌ No | Reversal date (if reversed) | "2024-11-05", null |
| `reversal_reason` | string | ❌ No | Reversal reason | "Duplicate payment", null |
| `waiver_amount` | decimal | ❌ No | Waiver amount (NGN) | 5000.00, 0.00 |
| `waiver_type` | string | ❌ No | Waiver type | "Interest", "Penalty", "Fees" |
| `waiver_approved_by` | string | ❌ No | Approver name/ID | "Manager John Doe", null |

### **Validation Rules:**

1. **Payment Amount Validation:**
   ```
   payment_amount = principal_paid + interest_paid + fees_paid + penalty_paid
   ```

2. **Loan Existence:**
   - `loan_id` must exist in the loans table

3. **Date Validation:**
   - `payment_date` should not be in the future
   - If `is_reversed = true`, `reversal_date` should be >= `payment_date`

4. **Reversal Validation:**
   - If `is_reversed = true`, both `reversal_date` and `reversal_reason` should be provided

---

## 3️⃣ Batch Sync Payload (Multiple Loans + Repayments)

### **JSON Payload:**

```json
{
  "sync_timestamp": "2024-10-18T14:30:00Z",
  "sync_type": "incremental",
  "data": {
    "loans": [
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
        "disbursement_date": "2024-10-15",
        "maturity_date": "2025-04-15",
        "loan_term_days": 180,
        "interest_rate": 0.1500,
        "fee_amount": 25000.00,
        "channel": "Direct",
        "channel_partner": null,
        "status": "Active",
        "closed_date": null
      },
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
        "disbursement_date": "2024-10-16",
        "maturity_date": "2025-04-16",
        "loan_term_days": 180,
        "interest_rate": 0.1500,
        "fee_amount": 37500.00,
        "channel": "Agent",
        "channel_partner": "Kuda Bank",
        "status": "Active",
        "closed_date": null
      }
    ],
    "repayments": [
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
      },
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

---

## 4️⃣ API Endpoint Specification

### **POST /api/v1/etl/sync**

**Request Headers:**
```
Content-Type: application/json
Authorization: Bearer <ETL_SERVICE_TOKEN>
X-Sync-Type: incremental | full
X-Sync-Timestamp: 2024-10-18T14:30:00Z
```

**Success Response (200 OK):**
```json
{
  "status": "success",
  "sync_id": "SYNC20241018143000",
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
  "next_sync_recommended": "2024-10-18T15:00:00Z"
}
```

**Partial Failure Response (207 Multi-Status):**
```json
{
  "status": "partial_success",
  "sync_id": "SYNC20241018143000",
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
  "errors": [
    {
      "entity_type": "loan",
      "entity_id": "LN2024001235",
      "error_code": "DUPLICATE_LOAN_ID",
      "error_message": "Loan with ID LN2024001235 already exists",
      "field": "loan_id"
    }
  ]
}
```

**Error Response (400 Bad Request):**
```json
{
  "status": "error",
  "error_code": "VALIDATION_ERROR",
  "error_message": "Invalid payload format",
  "errors": [
    {
      "field": "loans[0].loan_amount",
      "error": "Must be a positive number"
    },
    {
      "field": "repayments[1].payment_date",
      "error": "Date cannot be in the future"
    }
  ]
}
```

---

## 5️⃣ Integration Examples

### **Python (requests library):**

```python
import requests
from datetime import datetime

def sync_loan_to_analytics(loan_data: dict):
    """Send loan data to analytics service."""
    
    url = "https://analytics.example.com/api/v1/etl/sync"
    headers = {
        "Content-Type": "application/json",
        "Authorization": f"Bearer {ETL_SERVICE_TOKEN}",
        "X-Sync-Type": "incremental",
        "X-Sync-Timestamp": datetime.utcnow().isoformat() + "Z"
    }
    
    payload = {
        "sync_timestamp": datetime.utcnow().isoformat() + "Z",
        "sync_type": "incremental",
        "data": {
            "loans": [loan_data],
            "repayments": []
        },
        "metadata": {
            "total_loans": 1,
            "total_repayments": 0,
            "source_system": "main_backend",
            "etl_version": "1.0.0"
        }
    }
    
    response = requests.post(url, json=payload, headers=headers)
    
    if response.status_code == 200:
        print(f"✅ Loan {loan_data['loan_id']} synced successfully")
        return response.json()
    else:
        print(f"❌ Failed to sync loan: {response.text}")
        raise Exception(f"Sync failed: {response.status_code}")
```

---

## 📊 Summary

**Total Fields:**
- **Loans**: 20 fields (all from main backend)
- **Repayments**: 19 fields (all from main backend)
- **Computed**: 14 fields (calculated by analytics service)

**Key Points:**
- ✅ Main backend sends only raw transactional data
- ✅ Analytics service computes all derived fields
- ✅ Reduces load on main business server
- ✅ Centralized calculation logic
- ✅ Real-time updates via database triggers

