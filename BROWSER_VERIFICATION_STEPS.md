# Browser Verification Steps - Credit Health Overview Filters

## Quick Verification Checklist

### Step 1: Open Production Website
- [ ] Navigate to http://metrics.seedsandpennies.com
- [ ] Wait for page to fully load (should see dashboard)
- [ ] Check browser console for errors (F12 → Console tab)

### Step 2: Navigate to Credit Health Overview
- [ ] Look for tabs at the top of the page
- [ ] Click on "Credit Health Overview" tab
- [ ] Wait for data to load
- [ ] Should see branch data in a table

### Step 3: Locate Filter Button
- [ ] Look for "Show Filters" or "Filters" button
- [ ] Button should be near the top of the Credit Health Overview section
- [ ] Click the button to expand filter panel

### Step 4: Verify Filter Dropdowns Appear
- [ ] **Branch** dropdown - should show list of branches
- [ ] **Region** dropdown - should show "Nigeria"
- [ ] **Channel** dropdown - should show channel options
- [ ] **User Type** dropdown - should show user type options
- [ ] **Wave** dropdown - should show wave options

### Step 5: Test Individual Filters
- [ ] Select a value from **Branch** dropdown
  - [ ] Data should update automatically
  - [ ] Loading spinner should appear briefly
  - [ ] Table should show filtered results
  
- [ ] Select a value from **Region** dropdown
  - [ ] Data should update
  - [ ] Should see "Nigeria" as the only option
  
- [ ] Select a value from **Channel** dropdown
  - [ ] Data should update
  - [ ] Should see filtered results

- [ ] Select a value from **User Type** dropdown
  - [ ] Data should update
  - [ ] Should see filtered results

- [ ] Select a value from **Wave** dropdown
  - [ ] Data should update
  - [ ] Should see filtered results

### Step 6: Test Multiple Filters
- [ ] Select Branch = "AGEGE"
- [ ] Select Region = "Nigeria"
- [ ] Select Channel = "AGENT"
- [ ] Data should show only records matching ALL filters
- [ ] Summary should update accordingly

### Step 7: Test Clear All Button
- [ ] Click "Clear All" button
- [ ] All filters should reset to empty
- [ ] Data should show all branches again

### Step 8: Check Network Requests
- [ ] Open DevTools (F12)
- [ ] Go to Network tab
- [ ] Select a filter
- [ ] Look for API request to `/api/v1/branches`
- [ ] Verify query parameters include filter values
- [ ] Example: `/api/v1/branches?branch=AGEGE&region=Nigeria`

### Step 9: Verify Loading Indicator
- [ ] Select a filter
- [ ] Should see loading spinner briefly
- [ ] Spinner should disappear when data loads

### Step 10: Check for Errors
- [ ] Open DevTools Console (F12 → Console)
- [ ] Should see NO red error messages
- [ ] Should see successful API responses

---

## Expected Results

### Filter Dropdowns Should Show
```
Branch:     [All Branches ▼]
Region:     [All Regions ▼]
Channel:    [All Channels ▼]
User Type:  [All User Types ▼]
Wave:       [All Waves ▼]
```

### Sample Filter Values
- **Branch**: AGEGE, AJAH, AJEROMI IFELODUN, ALABA, etc.
- **Region**: Nigeria
- **Channel**: AGENT, BRANCH, etc.
- **User Type**: AGENT, OFFICER, etc.
- **Wave**: Wave1, Wave2, etc.

### API Response Example
```json
{
  "status": "success",
  "data": {
    "branches": [
      {
        "branch": "AGEGE",
        "region": "Nigeria",
        "portfolio_total": 8736582.04,
        "overdue_15d": 4528480.04,
        "par15_ratio": 0.5183,
        "active_loans": 402,
        "total_officers": 9
      }
    ],
    "summary": {
      "avg_par15_ratio": 0.5183,
      "total_branches": 1,
      "total_overdue_15d": 4528480.04,
      "total_portfolio": 8736582.04
    }
  }
}
```

---

## Troubleshooting

### Issue: Filters Not Showing
**Solution**:
1. Hard refresh browser: Ctrl+Shift+R (Windows) or Cmd+Shift+R (Mac)
2. Clear browser cache: F12 → Application → Clear Storage
3. Check browser console for errors
4. Try a different browser

### Issue: Filters Not Working
**Solution**:
1. Check Network tab in DevTools
2. Verify API calls are being made
3. Check response status (should be 200)
4. Look for error messages in response

### Issue: Data Not Updating
**Solution**:
1. Wait a few seconds for API response
2. Check if loading spinner appears
3. Verify API endpoint is responding
4. Check backend logs for errors

### Issue: "No Data" Message
**Solution**:
1. This is normal if filters return no results
2. Try different filter values
3. Click "Clear All" to reset filters
4. Verify data exists in database

---

## Browser DevTools Inspection

### Console Tab
- Should show no red errors
- May show info/warning messages (normal)
- Should see successful API responses

### Network Tab
- Filter by "Fetch/XHR"
- Look for `/api/v1/branches` requests
- Check response status (200 = success)
- Verify query parameters in URL

### Application Tab
- Check Local Storage for API configuration
- Verify VITE_API_URL is set correctly
- Check for any storage errors

---

## Performance Expectations

- **Page Load**: 2-3 seconds
- **Filter Selection**: 0.5-1 second
- **API Response**: 200-500ms
- **Data Display**: Instant after API response

---

## Success Criteria

✅ All 5 filters visible and functional
✅ Filters update data when selected
✅ Multiple filters work together
✅ Loading indicator appears during data fetch
✅ No JavaScript errors in console
✅ API calls include filter parameters
✅ Data updates correctly based on filters
✅ Clear All button resets all filters

---

## Production Website Details

- **URL**: http://metrics.seedsandpennies.com
- **Backend API**: http://localhost:8080/api/v1 (proxied via NGINX)
- **Frontend**: Served from /home/seeds-metrics-frontend/dist/
- **Database**: DigitalOcean PostgreSQL
- **Server**: 143.198.146.44

---

## Support Resources

1. **API Documentation**: `/backend/API_ENDPOINTS.md`
2. **Swagger Docs**: `/backend/docs/swagger.yaml`
3. **Component Code**: `/metrics-dashboard/src/components/CreditHealthByBranch.jsx`
4. **Handler Code**: `/backend/internal/handlers/dashboard_handler.go`
5. **Repository Code**: `/backend/internal/repository/dashboard_repository.go`

---

## Deployment Information

- **Commit Hash**: a5c3597
- **Deployment Date**: 2025-11-06
- **Status**: ✅ LIVE IN PRODUCTION
- **Ready for Testing**: YES

---

**Note**: If you encounter any issues, please check the troubleshooting section above or review the browser console for error messages.

