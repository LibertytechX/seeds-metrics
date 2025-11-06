# Analytics Backend API Endpoints

## 🚀 Base URL
```
http://localhost:8080
```

## ✅ Implemented Endpoints (Phase 1 - Core Dashboard)

### 1. Health Check
**GET** `/health`

**Response:**
```json
{
  "status": "healthy",
  "database": "connected"
}
```

---

### 2. Portfolio Metrics
**GET** `/api/v1/metrics/portfolio`

**Description:** Get portfolio-level aggregated metrics across all officers

**Response:**
```json
{
  "status": "success",
  "data": {
    "totalOverdue15d": 0,
    "avgDQI": 99,
    "avgAYR": 0.024,
    "avgRiskScore": 100,
    "topOfficer": {
      "officer_id": "OFF2024012",
      "name": "Sarah Johnson",
      "ayr": 0.048
    },
    "watchlistCount": 0,
    "totalOfficers": 2,
    "totalLoans": 1,
    "totalPortfolio": 420000
  }
}
```

---

### 3. Officers List
**GET** `/api/v1/officers`

**Description:** Get list of all loan officers with their metrics

**Query Parameters:**
- `region` (optional): Filter by region
- `branch` (optional): Filter by branch
- `channel` (optional): Filter by channel
- `sort_by` (optional): Sort field (default: officer_name)
- `sort_order` (optional): asc or desc (default: asc)
- `limit` (optional): Number of results (default: 100)
- `offset` (optional): Pagination offset (default: 0)

**Response:**
```json
{
  "status": "success",
  "data": {
    "officers": [
      {
        "id": 0,
        "officer_id": "OFF2024012",
        "name": "Sarah Johnson",
        "region": "South West",
        "branch": "Lagos Main",
        "channel": "Direct",
        "rawMetrics": {
          "firstMiss": 0,
          "disbursed": 1,
          "dpd1to6Bal": 0,
          "amountDue7d": 0,
          "movedTo7to30": 0,
          "prevDpd1to6Bal": 0,
          "feesCollected": 0,
          "feesDue": 0,
          "interestCollected": 0,
          "overdue15d": 0,
          "totalPortfolio": 420000,
          "par15MidMonth": 0,
          "waivers": 0,
          "backdated": 0,
          "entries": 0,
          "reversals": 0,
          "hadFloatGap": false
        },
        "calculatedMetrics": {
          "fimr": 0,
          "slippage": 0,
          "roll": 0,
          "frr": 0,
          "ayr": 0.048,
          "dqi": 99,
          "riskScore": 100,
          "yield": 0,
          "overdue15dVolume": 0,
          "riskScoreNorm": 1,
          "onTimeRate": 1,
          "channelPurity": 1,
          "porr": 0
        },
        "riskBand": "Green"
      }
    ],
    "total": 2
  }
}
```

---

### 4. Officer Detail
**GET** `/api/v1/officers/:officer_id`

**Description:** Get detailed metrics for a specific officer

**Response:** Same structure as single officer in officers list

---

### 5. FIMR Loans
**GET** `/api/v1/fimr/loans`

**Description:** Get loans that missed first installment

**Query Parameters:**
- `officer_id` (optional): Filter by officer
- `branch` (optional): Filter by branch
- `region` (optional): Filter by region
- `limit` (optional): Number of results (default: 100)
- `offset` (optional): Pagination offset (default: 0)

**Response:**
```json
{
  "status": "success",
  "data": {
    "loans": [],
    "total": 0
  }
}
```

---

### 6. Early Indicator Loans
**GET** `/api/v1/early-indicators/loans`

**Description:** Get loans in early delinquency (DPD 1-30)

**Query Parameters:**
- `officer_id` (optional): Filter by officer
- `branch` (optional): Filter by branch
- `region` (optional): Filter by region
- `dpd_min` (optional): Minimum DPD (default: 1)
- `dpd_max` (optional): Maximum DPD (default: 30)
- `limit` (optional): Number of results (default: 100)
- `offset` (optional): Pagination offset (default: 0)

**Response:**
```json
{
  "status": "success",
  "data": {
    "loans": [],
    "total": 0
  }
}
```

---

### 7. Branches
**GET** `/api/v1/branches`

**Description:** Get branch-level aggregated metrics

**Query Parameters:**
- `branch` (optional): Filter by branch name
- `region` (optional): Filter by region
- `channel` (optional): Filter by channel
- `user_type` (optional): Filter by user type
- `wave` (optional): Filter by wave/cohort
- `sort_by` (optional): Sort field (default: branch)
- `sort_dir` (optional): Sort direction - asc or desc (default: asc)

**Response:**
```json
{
  "status": "success",
  "data": {
    "branches": [
      {
        "branch": "Lagos Main",
        "region": "South West",
        "portfolio_total": 420000,
        "overdue_15d": 0,
        "par15_ratio": 0,
        "ayr": 0,
        "dqi": 0,
        "fimr": 0,
        "active_loans": 1,
        "total_officers": 1
      }
    ],
    "summary": {
      "total_branches": 1,
      "total_portfolio": 420000,
      "total_overdue_15d": 0,
      "avg_par15_ratio": 0
    }
  }
}
```

---

### 8. Team Members
**GET** `/api/v1/team-members`

**Description:** Get list of team members for audit assignment

**Response:**
```json
{
  "status": "success",
  "data": [
    {
      "id": 0,
      "name": "Unassigned",
      "role": ""
    },
    {
      "id": "me",
      "name": "Assigned to Me",
      "role": "Current User"
    }
  ]
}
```

---

## 🔄 ETL Endpoints (Already Implemented)

### 9. Create/Update Loan
**POST** `/api/v1/etl/loans`

### 10. Create Repayment
**POST** `/api/v1/etl/repayments`

### 11. Batch Sync
**POST** `/api/v1/etl/sync`

---

## 📊 Frontend Integration

All endpoints return data in the format expected by the React frontend components:

- **KPIStrip.jsx** → `/api/v1/metrics/portfolio`
- **AgentPerformance.jsx** → `/api/v1/officers`
- **FIMRDrilldown.jsx** → `/api/v1/fimr/loans`
- **EarlyIndicatorsDrilldown.jsx** → `/api/v1/early-indicators/loans`
- **CreditHealthByBranch.jsx** → `/api/v1/branches`
- **DataTables.jsx** → Multiple endpoints

---

## 🧪 Testing

Run the test script to verify all endpoints:

```bash
bash backend/test-endpoints.sh
```

All 8 endpoints are fully functional and tested! ✅

