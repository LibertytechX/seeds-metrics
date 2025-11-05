# User Type Filter Implementation - Complete

## 📋 Overview

This document describes the implementation of the **User Type Filter** feature for the Officer Performance table in the Seeds Metrics dashboard.

**Feature**: Multi-select dropdown filter that allows filtering officers by their user type (Merchant, Agent, Loan Officer, etc.)

**Status**: ✅ **COMPLETE** (Frontend & Backend)

---

## 🎯 Requirements Met

✅ **Filter Location**: Added to the filter panel in Officer Performance table component  
✅ **Filter Type**: Implemented as multi-select dropdown (not radio buttons)  
✅ **User Type Options**: Dynamically fetched from backend API  
✅ **Data Source**: Backend API endpoint `/api/v1/filters/user-types`  
✅ **Functionality**: Allows multiple selections, includes "Select All" and "Clear All" options  
✅ **Integration**: Works seamlessly with existing filters (region, branch, risk band, etc.)  
✅ **Real-time Updates**: Filter updates immediately when selections change  

---

## 🏗️ Architecture

### **Backend Changes**

#### 1. Database Migration
**File**: `backend/migrations/017_add_user_type_to_officers.sql`

```sql
-- Add user_type column to officers table
ALTER TABLE officers 
ADD COLUMN IF NOT EXISTS user_type VARCHAR(100);

-- Create index for filtering performance
CREATE INDEX IF NOT EXISTS idx_officers_user_type ON officers(user_type);
```

#### 2. Data Model Updates
**File**: `backend/internal/models/officer.go`

```go
type Officer struct {
    OfficerID        string     `json:"officer_id" db:"officer_id"`
    OfficerName      string     `json:"officer_name" db:"officer_name"`
    // ... other fields
    UserType         *string    `json:"user_type,omitempty" db:"user_type"`  // NEW
    // ... other fields
}
```

#### 3. Django Sync Updates
**File**: `backend/internal/repository/django_repository.go`

- Updated `GetOfficers()` to select `user_type` from Django's `accounts_customuser` table
- Updated `GetOfficerByID()` to include user_type field
- Data syncs from Django DB to SeedsMetrics DB

#### 4. Officer Repository Updates
**File**: `backend/internal/repository/officer_repository.go`

- Updated `Create()` to insert/update user_type
- Updated `GetByID()` and `List()` to select user_type
- Handles NULL values gracefully

#### 5. Dashboard Repository Updates
**File**: `backend/internal/repository/dashboard_repository.go`

**New Function**: `getUserTypes()`
```go
func (r *DashboardRepository) getUserTypes() ([]string, error) {
    query := "SELECT DISTINCT user_type FROM officers WHERE user_type IS NOT NULL AND user_type != '' ORDER BY user_type"
    // Returns array of distinct user types
}
```

**Updated Function**: `GetFilterOptions()`
```go
case "user-types":
    return r.getUserTypes()
```

**Updated Function**: `GetOfficers()`
```go
if userType, ok := filters["user_type"].(string); ok && userType != "" {
    query += fmt.Sprintf(" AND o.user_type = $%d", argCount)
    args = append(args, userType)
    argCount++
}
```

#### 6. API Endpoints

**New Endpoint**: `GET /api/v1/filters/user-types`
```json
{
  "status": "success",
  "data": {
    "user_types": [
      "AGENT",
      "AJO_AGENT",
      "DMO_AGENT",
      "LIBERTY_RETAIL",
      "LOTTO_AGENT",
      "MERCHANT",
      "MERCHANT_AGENT",
      "MICRO_SAVER",
      "PERSONAL",
      "PHARMACIST",
      "PROSPER_AGENT",
      "STAFF_AGENT",
      "lite"
    ]
  }
}
```

**Updated Endpoint**: `GET /api/v1/officers?user_type=MERCHANT`
- Now accepts `user_type` query parameter
- Filters officers by user type

---

### **Frontend Changes**

#### 1. New Component: MultiSelect
**Files**: 
- `metrics-dashboard/src/components/MultiSelect.jsx`
- `metrics-dashboard/src/components/MultiSelect.css`

**Features**:
- Reusable multi-select dropdown component
- Checkbox-based selection
- "Select All" and "Clear All" buttons
- Click-outside-to-close functionality
- Displays count when multiple items selected
- Smooth animations and transitions

**Props**:
```javascript
<MultiSelect
  options={['Option1', 'Option2', ...]}
  selectedValues={['Option1']}
  onChange={(values) => handleChange(values)}
  placeholder="Select..."
  label="Label"
/>
```

#### 2. Updated Component: AgentPerformance
**File**: `metrics-dashboard/src/components/AgentPerformance.jsx`

**Changes**:

1. **Import MultiSelect**:
```javascript
import MultiSelect from './MultiSelect';
```

2. **Add State**:
```javascript
const [allUserTypes, setAllUserTypes] = useState([]);
const [filters, setFilters] = useState({
  // ... existing filters
  userTypes: [],  // NEW
});
```

3. **Fetch User Types on Mount**:
```javascript
useEffect(() => {
  const fetchUserTypes = async () => {
    const API_BASE_URL = import.meta.env.VITE_API_URL || 'http://localhost:8081/api/v1';
    const response = await fetch(`${API_BASE_URL}/filters/user-types`);
    const data = await response.json();
    if (data.status === 'success') {
      setAllUserTypes(data.data.user_types || []);
    }
  };
  fetchUserTypes();
}, []);
```

4. **Apply Filter**:
```javascript
const filteredAgents = useMemo(() => {
  return agentData.filter(agent => {
    // ... existing filters
    
    // User type filter (multi-select)
    if (filters.userTypes.length > 0 && !filters.userTypes.includes(agent.userType)) {
      return false;
    }
    
    return true;
  });
}, [agentData, filters]);
```

5. **Add to Filter Panel**:
```jsx
<div className="filter-group">
  <MultiSelect
    options={allUserTypes}
    selectedValues={filters.userTypes}
    onChange={(values) => handleFilterChange('userTypes', values)}
    placeholder="All User Types"
  />
</div>
```

6. **Update Clear Filters**:
```javascript
const clearFilters = () => {
  setFilters({
    // ... existing filters
    userTypes: [],  // NEW
  });
};
```

7. **Update Active Filter Count**:
```javascript
const activeFilterCount = Object.entries(filters).filter(([key, value]) => {
  if (key === 'userTypes') return value.length > 0;
  return value !== '';
}).length;
```

---

## 📊 User Types Available

Based on Django database query, the following user types are available:

| User Type | Count |
|-----------|-------|
| lite | 2032 |
| MERCHANT | 1489 |
| LOTTO_AGENT | 698 |
| STAFF_AGENT | 639 |
| PERSONAL | 473 |
| AGENT | 452 |
| AJO_AGENT | 98 |
| DMO_AGENT | 30 |
| MERCHANT_AGENT | 27 |
| LIBERTY_RETAIL | 24 |
| PHARMACIST | 13 |
| PROSPER_AGENT | 5 |
| MICRO_SAVER | 1 |

**Total**: 13 distinct user types

---

## 🚀 Deployment

### **Automated Deployment Script**

A deployment script has been created: `deploy-user-type-feature.sh`

**What it does**:
1. Syncs backend code files to production server
2. Builds backend binary on server
3. Runs database migration
4. Syncs user_type data from Django DB
5. Restarts backend service
6. Tests API endpoints
7. Builds frontend
8. Deploys frontend to production
9. Verifies deployment

**Usage**:
```bash
./deploy-user-type-feature.sh
```

### **Manual Deployment Steps**

If the automated script fails due to network issues, follow these manual steps:

#### **Backend Deployment**

1. **SSH to production server**:
```bash
ssh root@143.198.146.44
cd /home/seeds-metrics-backend/backend
```

2. **Update code files** (copy from local):
```bash
# On local machine
scp backend/internal/models/officer.go root@143.198.146.44:/home/seeds-metrics-backend/backend/internal/models/
scp backend/internal/repository/officer_repository.go root@143.198.146.44:/home/seeds-metrics-backend/backend/internal/repository/
scp backend/internal/repository/django_repository.go root@143.198.146.44:/home/seeds-metrics-backend/backend/internal/repository/
scp backend/internal/repository/dashboard_repository.go root@143.198.146.44:/home/seeds-metrics-backend/backend/internal/repository/
scp backend/scripts/sync_from_django.go root@143.198.146.44:/home/seeds-metrics-backend/backend/scripts/
```

3. **Build backend**:
```bash
cd /home/seeds-metrics-backend/backend
go build -o seeds-metrics-api ./cmd/api
```

4. **Run migration**:
```bash
source .env
psql "host=$DB_HOST port=$DB_PORT dbname=$DB_NAME user=$DB_USER password=$DB_PASSWORD sslmode=$DB_SSLMODE" \
  -f migrations/017_add_user_type_to_officers.sql
```

5. **Sync data**:
```bash
./sync_from_django
```

6. **Deploy binary**:
```bash
mv api api.old.$(date +%Y%m%d_%H%M%S)
mv seeds-metrics-api api
chmod +x api
```

7. **Restart service**:
```bash
sudo systemctl restart seeds-metrics-api
sudo systemctl status seeds-metrics-api
```

#### **Frontend Deployment**

1. **Build frontend** (on local machine):
```bash
cd metrics-dashboard
npm run build
```

2. **Deploy to production**:
```bash
rsync -avz --delete dist/ root@143.198.146.44:/home/seeds-metrics-frontend/
```

---

## 🧪 Testing

### **Backend API Testing**

1. **Test user types endpoint**:
```bash
curl "https://metrics.seedsandpennies.com/api/v1/filters/user-types" | jq '.'
```

Expected response:
```json
{
  "status": "success",
  "data": {
    "user_types": ["AGENT", "MERCHANT", "LOTTO_AGENT", ...]
  }
}
```

2. **Test officer filtering by user type**:
```bash
curl "https://metrics.seedsandpennies.com/api/v1/officers?user_type=MERCHANT&page=1&limit=5" | \
  jq '.data.officers[] | {officer_id, officer_name, user_type}'
```

Expected response:
```json
{
  "officer_id": "123",
  "officer_name": "John Doe",
  "user_type": "MERCHANT"
}
```

### **Frontend Testing**

1. Open https://metrics.seedsandpennies.com
2. Navigate to **"Agent Performance"** tab
3. Click **"Filters"** button to show filter panel
4. Locate the **"User Type"** multi-select dropdown
5. Click to open dropdown
6. Verify all user types are listed
7. Select one or more user types
8. Verify the officer list updates immediately
9. Verify the filter count badge updates
10. Click **"Select All"** - verify all types are selected
11. Click **"Clear All"** - verify all selections are cleared
12. Test combination with other filters (region, branch, etc.)
13. Click **"Clear All Filters"** - verify user type filter is also cleared

---

## 📁 Files Created/Modified

### **Created**:
- `backend/migrations/017_add_user_type_to_officers.sql` - Database migration
- `metrics-dashboard/src/components/MultiSelect.jsx` - Multi-select component
- `metrics-dashboard/src/components/MultiSelect.css` - Multi-select styles
- `deploy-user-type-feature.sh` - Deployment script
- `USER_TYPE_FILTER_IMPLEMENTATION.md` - This documentation

### **Modified**:
- `backend/internal/models/officer.go` - Added UserType field
- `backend/internal/repository/officer_repository.go` - Handle user_type CRUD
- `backend/internal/repository/django_repository.go` - Sync user_type from Django
- `backend/internal/repository/dashboard_repository.go` - Filter options & officer filtering
- `backend/scripts/sync_from_django.go` - Sync user_type field
- `metrics-dashboard/src/components/AgentPerformance.jsx` - Added user type filter

---

## ✅ Completion Checklist

- [x] Database migration created and tested
- [x] Backend models updated with user_type field
- [x] Django sync updated to include user_type
- [x] Officer repository updated for CRUD operations
- [x] Dashboard repository updated with getUserTypes() function
- [x] API endpoint `/api/v1/filters/user-types` implemented
- [x] API endpoint `/api/v1/officers` updated to support user_type filtering
- [x] Backend compiled successfully with no errors
- [x] MultiSelect component created and styled
- [x] AgentPerformance component updated with user type filter
- [x] Frontend built successfully with no errors
- [x] Deployment script created
- [x] Documentation completed

---

## 🎉 Success!

The User Type Filter feature is now **fully implemented** and ready for deployment!

**Next Steps**:
1. Deploy to production using `./deploy-user-type-feature.sh`
2. Test the feature on production
3. Gather user feedback
4. Monitor performance and usage

---

## 📞 Support

If you encounter any issues during deployment or testing, check:
1. Backend logs: `sudo journalctl -u seeds-metrics-api -f`
2. Browser console for frontend errors (F12)
3. Network tab for API call failures
4. Database connection and migration status

