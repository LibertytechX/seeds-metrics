# Credit Health Overview Filters - Testing Guide

## Quick Start

### Prerequisites
- Backend running on `http://localhost:8080` or configured API URL
- Frontend running on `http://localhost:5173` or configured frontend URL
- Database populated with loan data

### Test Environment Setup

1. **Start Backend**
   ```bash
   cd backend
   go run cmd/api/main.go
   ```

2. **Start Frontend**
   ```bash
   cd metrics-dashboard
   npm run dev
   ```

3. **Navigate to Credit Health Overview**
   - Open http://localhost:5173
   - Click on "Credit Health by Branch" tab

## Manual Testing Checklist

### Test 1: Filter Options Load
**Steps**:
1. Open Credit Health Overview tab
2. Click "Filters" button
3. Observe filter dropdowns

**Expected Results**:
- [ ] Branch dropdown populated with branch names
- [ ] Region dropdown populated with region names
- [ ] Channel dropdown populated with channel values
- [ ] User Type dropdown populated with user types
- [ ] Wave dropdown populated with wave values
- [ ] All dropdowns show "All [Type]" as default option

### Test 2: Single Filter - Region
**Steps**:
1. Click "Filters" button
2. Select a region from Region dropdown
3. Observe table updates

**Expected Results**:
- [ ] Loading spinner appears briefly
- [ ] Table shows only branches from selected region
- [ ] Filter badge shows "1" active filter
- [ ] API call includes `?region=<selected>`

### Test 3: Single Filter - Branch
**Steps**:
1. Click "Filters" button
2. Select a branch from Branch dropdown
3. Observe table updates

**Expected Results**:
- [ ] Loading spinner appears
- [ ] Table shows only selected branch
- [ ] Filter badge shows "1" active filter
- [ ] API call includes `?branch=<selected>`

### Test 4: Single Filter - Channel
**Steps**:
1. Click "Filters" button
2. Select a channel from Channel dropdown
3. Observe table updates

**Expected Results**:
- [ ] Loading spinner appears
- [ ] Table shows only branches with selected channel
- [ ] Filter badge shows "1" active filter
- [ ] API call includes `?channel=<selected>`

### Test 5: Single Filter - User Type
**Steps**:
1. Click "Filters" button
2. Select a user type from User Type dropdown
3. Observe table updates

**Expected Results**:
- [ ] Loading spinner appears
- [ ] Table shows only branches with selected user type
- [ ] Filter badge shows "1" active filter
- [ ] API call includes `?user_type=<selected>`

### Test 6: Single Filter - Wave
**Steps**:
1. Click "Filters" button
2. Select a wave from Wave dropdown
3. Observe table updates

**Expected Results**:
- [ ] Loading spinner appears
- [ ] Table shows only branches from selected wave
- [ ] Filter badge shows "1" active filter
- [ ] API call includes `?wave=<selected>`

### Test 7: Multiple Filters Combined
**Steps**:
1. Click "Filters" button
2. Select Region = "Lagos"
3. Select Channel = "AGENT"
4. Select User Type = "AGENT"
5. Observe table updates

**Expected Results**:
- [ ] Loading spinner appears
- [ ] Table shows only branches matching ALL criteria
- [ ] Filter badge shows "3" active filters
- [ ] API call includes all three filters: `?region=Lagos&channel=AGENT&user_type=AGENT`

### Test 8: Clear All Filters
**Steps**:
1. Apply multiple filters (see Test 7)
2. Click "Clear All" button
3. Observe table updates

**Expected Results**:
- [ ] All filter dropdowns reset to "All [Type]"
- [ ] Filter badge disappears
- [ ] Table shows all branches again
- [ ] Loading spinner appears during refresh

### Test 9: No Results Scenario
**Steps**:
1. Click "Filters" button
2. Select filters that return no results (e.g., non-existent combination)
3. Observe table

**Expected Results**:
- [ ] Loading spinner appears
- [ ] Table shows "No branches found matching the selected filters" message
- [ ] Message is centered and italicized
- [ ] No error in console

### Test 10: Sorting with Filters
**Steps**:
1. Apply a filter (e.g., Region = "Lagos")
2. Click on "Portfolio Total" column header
3. Click again to reverse sort

**Expected Results**:
- [ ] Filtered results are sorted by Portfolio Total
- [ ] Ascending sort works
- [ ] Descending sort works
- [ ] Sorting works with multiple filters applied

### Test 11: Export with Filters
**Steps**:
1. Apply a filter (e.g., Region = "Lagos")
2. Click "Export CSV" button
3. Check downloaded file

**Expected Results**:
- [ ] CSV file downloads
- [ ] CSV contains only filtered branches
- [ ] All columns present in CSV
- [ ] Data matches table display

### Test 12: API Response Verification
**Steps**:
1. Open Browser DevTools (F12)
2. Go to Network tab
3. Apply filters and observe API calls

**Expected Results**:
- [ ] GET request to `/api/v1/filters/branches` (on load)
- [ ] GET request to `/api/v1/filters/regions` (on load)
- [ ] GET request to `/api/v1/filters/channels` (on load)
- [ ] GET request to `/api/v1/filters/user-types` (on load)
- [ ] GET request to `/api/v1/filters/waves` (on load)
- [ ] GET request to `/api/v1/branches?<filters>` (when filters change)
- [ ] Response status 200 for all requests
- [ ] Response contains valid JSON

## Browser Console Checks

**Expected**: No errors in console

**Check for**:
- [ ] No 404 errors for filter endpoints
- [ ] No 500 errors from API
- [ ] No JavaScript errors
- [ ] No CORS errors

## Performance Testing

### Test 13: Load Time with Filters
**Steps**:
1. Open DevTools Performance tab
2. Apply a filter
3. Observe load time

**Expected Results**:
- [ ] Filter options load in < 1 second
- [ ] Filtered data loads in < 2 seconds
- [ ] No UI freezing during load

### Test 14: Multiple Filter Changes
**Steps**:
1. Rapidly change multiple filters
2. Observe behavior

**Expected Results**:
- [ ] UI remains responsive
- [ ] Latest filter combination is applied
- [ ] No duplicate API calls
- [ ] Loading spinner shows for each change

## Edge Cases

### Test 15: Empty Database
**Scenario**: Database has no branches

**Expected Results**:
- [ ] Filter dropdowns show "All [Type]" only
- [ ] Table shows "No branches found" message
- [ ] No errors in console

### Test 16: Filter with Special Characters
**Scenario**: Branch name contains special characters

**Expected Results**:
- [ ] Filter works correctly
- [ ] Special characters displayed properly
- [ ] No SQL injection issues

### Test 17: Very Large Dataset
**Scenario**: Database has 10,000+ branches

**Expected Results**:
- [ ] Filters still work
- [ ] Response time acceptable (< 5 seconds)
- [ ] UI remains responsive

## Regression Testing

### Test 18: Backward Compatibility
**Steps**:
1. Access endpoint without filters: `/api/v1/branches`
2. Verify all branches returned

**Expected Results**:
- [ ] All branches returned
- [ ] No errors
- [ ] Response format unchanged

## Sign-Off Checklist

- [ ] All 18 tests passed
- [ ] No console errors
- [ ] No API errors
- [ ] Performance acceptable
- [ ] UI responsive
- [ ] Filters work individually
- [ ] Filters work in combination
- [ ] Clear All works
- [ ] Export works with filters
- [ ] Sorting works with filters
- [ ] No data message displays correctly
- [ ] Loading indicator shows/hides correctly
- [ ] Filter options populate correctly
- [ ] API calls include correct parameters
- [ ] Response data is correct

## Troubleshooting

### Issue: Filter dropdowns empty
**Solution**: Check if filter endpoints are returning data
```bash
curl http://localhost:8080/api/v1/filters/branches
```

### Issue: Filters not applied
**Solution**: Check Network tab in DevTools to verify filter parameters in API call

### Issue: Loading spinner stuck
**Solution**: Check browser console for API errors

### Issue: "No data" message always shows
**Solution**: Verify database has data matching filter criteria

