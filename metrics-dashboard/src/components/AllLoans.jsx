import React, { useState, useEffect, useMemo, useRef } from 'react';
import { Download, Filter, FileText, Eye, RefreshCw, ChevronDown } from 'lucide-react';
import jsPDF from 'jspdf';
import 'jspdf-autotable';
import LoanRepaymentsModal from './LoanRepaymentsModal';
import Pagination from './Pagination';
import './AllLoans.css';

const AllLoans = ({ initialLoans = [], initialFilter = null }) => {
  const [loans, setLoans] = useState(initialLoans);
  const [loading, setLoading] = useState(false);
  const [sortConfig, setSortConfig] = useState({ key: 'disbursement_date', direction: 'desc' });
  const [filters, setFilters] = useState({
    officer_id: initialFilter?.officer_id || '',
    branch: '',
    regions: [], // Multi-select region filter
    wave: '', // Single-select wave filter
    channel: '',
    status: '',
    customer_phone: '',
    vertical_lead_email: '',
    loan_type: initialFilter?.loan_type || '', // 'active' or 'inactive'
    rot_type: initialFilter?.rot_type || '', // 'early' or 'late'
    delay_type: initialFilter?.delay_type || '', // 'risky' for high delay loans
  });
  const [allBranches, setAllBranches] = useState([]);
  const [allRegions, setAllRegions] = useState([]);
  const [allChannels, setAllChannels] = useState([]);
  const [allWaves, setAllWaves] = useState([]);
  const [allVerticalLeads, setAllVerticalLeads] = useState([]);
  const [isRegionDropdownOpen, setIsRegionDropdownOpen] = useState(false);
  const regionDropdownRef = useRef(null);
  const [filterLabel, setFilterLabel] = useState(
    initialFilter?.officer_name ? `Officer: ${initialFilter.officer_name}` :
    initialFilter?.label ? initialFilter.label : ''
  );
  const [showFilters, setShowFilters] = useState(false);
  const [pagination, setPagination] = useState({
    page: 1,
    limit: 50,
    total: 0,
    pages: 0,
  });
  const [repaymentsModalOpen, setRepaymentsModalOpen] = useState(false);
  const [selectedLoan, setSelectedLoan] = useState(null);
  const [recalculating, setRecalculating] = useState(false);
  const [recalculateMessage, setRecalculateMessage] = useState('');

  // Close dropdown when clicking outside
  useEffect(() => {
    const handleClickOutside = (event) => {
      if (regionDropdownRef.current && !regionDropdownRef.current.contains(event.target)) {
        setIsRegionDropdownOpen(false);
      }
    };
    document.addEventListener('mousedown', handleClickOutside);
    return () => document.removeEventListener('mousedown', handleClickOutside);
  }, []);

  // Fetch filter options from API
  const fetchFilterOptions = async () => {
    try {
      const API_BASE_URL = import.meta.env.VITE_API_URL || 'http://localhost:8081/api/v1';

      // Fetch branches, regions, channels, and waves from API
      const [branchesRes, regionsRes, channelsRes, wavesRes] = await Promise.all([
        fetch(`${API_BASE_URL}/filters/branches`),
        fetch(`${API_BASE_URL}/filters/regions`),
        fetch(`${API_BASE_URL}/filters/channels`),
        fetch(`${API_BASE_URL}/filters/waves`),
      ]);

      const [branchesData, regionsData, channelsData, wavesData] = await Promise.all([
        branchesRes.json(),
        regionsRes.json(),
        channelsRes.json(),
        wavesRes.json(),
      ]);

      if (branchesData.status === 'success') {
        setAllBranches(branchesData.data.branches || []);
      }
      if (regionsData.status === 'success') {
        setAllRegions(regionsData.data.regions || []);
      }
      if (channelsData.status === 'success') {
        setAllChannels(channelsData.data.channels || []);
      }
      if (wavesData.status === 'success') {
        setAllWaves(wavesData.data.waves || []);
      }
    } catch (error) {
      console.error('Error fetching filter options:', error);
    }
  };

  // Fetch loans from API
  const fetchLoans = async () => {
    setLoading(true);
    try {
      console.log('🔍 AllLoans: fetchLoans called with filters:', filters);

      // Exclude loan_type, rot_type, delay_type, and regions from API params (will handle regions separately)
      // Include customer_phone for server-side filtering
      const apiFilters = Object.fromEntries(
        Object.entries(filters).filter(([k, v]) => v !== '' && k !== 'loan_type' && k !== 'rot_type' && k !== 'delay_type' && k !== 'regions')
      );

      // Convert regions array to comma-separated string
      if (filters.regions && filters.regions.length > 0) {
        apiFilters.region = filters.regions.join(',');
      }

      const params = new URLSearchParams({
        page: pagination.page,
        limit: pagination.limit,
        sort_by: sortConfig.key,
        sort_dir: sortConfig.direction.toUpperCase(),
        ...apiFilters,
      });

      const API_BASE_URL = import.meta.env.VITE_API_URL || 'http://localhost:8081/api/v1';
      const response = await fetch(`${API_BASE_URL}/loans?${params}`);
      const data = await response.json();

      if (data.status === 'success') {
        let fetchedLoans = data.data.loans || [];
        console.log(`📦 AllLoans: Fetched ${fetchedLoans.length} loans from API`);

        // Apply client-side filtering for loan_type and rot_type
        if (filters.loan_type === 'active') {
          console.log('🔵 Filtering for ACTIVE loans');
          fetchedLoans = fetchedLoans.filter(loan =>
            loan.total_outstanding > 2000 && loan.days_since_last_repayment < 6
          );
          console.log(`✅ Active loans filtered: ${fetchedLoans.length} loans`);
        } else if (filters.loan_type === 'inactive') {
          console.log('🔵 Filtering for INACTIVE loans');
          fetchedLoans = fetchedLoans.filter(loan =>
            loan.total_outstanding <= 2000 || loan.days_since_last_repayment > 5
          );
          console.log(`✅ Inactive loans filtered: ${fetchedLoans.length} loans`);
        } else if (filters.loan_type === 'overdue_15d') {
          console.log('🔵 Filtering for OVERDUE >15 Days loans');
          fetchedLoans = fetchedLoans.filter(loan =>
            loan.current_dpd > 15
          );
          console.log(`✅ Overdue >15 Days loans filtered: ${fetchedLoans.length} loans`);
        }

        if (filters.rot_type === 'early') {
          console.log('🔵 Filtering for EARLY ROT loans');
          fetchedLoans = fetchedLoans.filter(loan => {
            const loanAge = Math.floor((new Date() - new Date(loan.disbursement_date)) / (1000 * 60 * 60 * 24));
            return loanAge < 7 && loan.current_dpd > 4;
          });
          console.log(`✅ Early ROT loans filtered: ${fetchedLoans.length} loans`);
        } else if (filters.rot_type === 'late') {
          console.log('🔵 Filtering for LATE ROT loans');
          fetchedLoans = fetchedLoans.filter(loan => {
            const loanAge = Math.floor((new Date() - new Date(loan.disbursement_date)) / (1000 * 60 * 60 * 24));
            return loanAge >= 7 && loan.current_dpd > 4;
          });
          console.log(`✅ Late ROT loans filtered: ${fetchedLoans.length} loans`);
        }

        // Filter for risky delay loans (repayment_delay_rate < 60%)
        if (filters.delay_type === 'risky') {
          console.log('🔵 Filtering for RISKY DELAY loans (repayment_delay_rate < 60%)');
          fetchedLoans = fetchedLoans.filter(loan => {
            // Only consider active loans with outstanding balance > 2000
            if (loan.status !== 'Active' || loan.total_outstanding <= 2000) {
              return false;
            }

            // Filter by repayment_delay_rate < 60% (strict less-than)
            if (loan.repayment_delay_rate != null) {
              return loan.repayment_delay_rate < 60;
            }

            return false;
          });
          console.log(`✅ Risky delay loans filtered: ${fetchedLoans.length} loans`);
        }

        setLoans(fetchedLoans);

        // Extract unique vertical leads from fetched loans
        const uniqueVerticalLeads = [...new Set(
          fetchedLoans
            .map(loan => loan.vertical_lead_email)
            .filter(email => email != null && email !== '')
        )].sort();
        setAllVerticalLeads(uniqueVerticalLeads);

        // Use backend total for pagination, not client-side filtered count
        // The backend total represents the actual number of records matching server-side filters
        // Client-side filtering (loan_type, rot_type, delay_type) is for display only
        setPagination({
          page: data.data.page,
          limit: data.data.limit,
          total: data.data.total, // Use backend total, not fetchedLoans.length
          pages: data.data.pages, // Use backend calculated pages
        });
      }
    } catch (error) {
      console.error('Error fetching loans:', error);
    } finally {
      setLoading(false);
    }
  };

  // Update filters when initialFilter prop changes
  useEffect(() => {
    if (initialFilter) {
      console.log('🔄 AllLoans: initialFilter changed:', initialFilter);
      setFilters({
        officer_id: initialFilter.officer_id || '',
        branch: '',
        regions: [],
        wave: '',
        channel: '',
        status: '',
        customer_phone: '',
        vertical_lead_email: '',
        loan_type: initialFilter.loan_type || '',
        rot_type: initialFilter.rot_type || '',
        delay_type: initialFilter.delay_type || '',
      });
      setFilterLabel(
        initialFilter.officer_name ? `Officer: ${initialFilter.officer_name}` :
        initialFilter.label ? initialFilter.label : ''
      );
    }
  }, [initialFilter]);

  // Fetch filter options on mount
  useEffect(() => {
    fetchFilterOptions();
  }, []);

  useEffect(() => {
    fetchLoans();
  }, [pagination.page, pagination.limit, sortConfig, filters]);

  // Get unique values for filter dropdowns
  const filterOptions = useMemo(() => {
    return {
      officers: [...new Set(loans.map(l => l.officer_name))].filter(Boolean).sort(),
      branches: allBranches.length > 0 ? allBranches.sort() : [...new Set(loans.map(l => l.branch))].filter(Boolean).sort(),
      regions: allRegions.length > 0 ? allRegions.sort() : [...new Set(loans.map(l => l.region))].filter(Boolean).sort(),
      waves: allWaves.length > 0 ? allWaves.sort() : [...new Set(loans.map(l => l.wave))].filter(Boolean).sort(),
      channels: allChannels.length > 0 ? allChannels.sort() : [...new Set(loans.map(l => l.channel))].filter(Boolean).sort(),
      statuses: [...new Set(loans.map(l => l.status))].filter(Boolean).sort(),
      verticalLeads: allVerticalLeads.length > 0 ? allVerticalLeads : [...new Set(loans.map(l => l.vertical_lead_email))].filter(Boolean).sort(),
    };
  }, [loans, allBranches, allRegions, allWaves, allChannels, allVerticalLeads]);

  const handleSort = (key) => {
    setSortConfig(prev => ({
      key,
      direction: prev.key === key && prev.direction === 'asc' ? 'desc' : 'asc',
    }));
  };

  const handleFilterChange = (filterKey, value) => {
    setFilters(prev => ({ ...prev, [filterKey]: value }));
    setPagination(prev => ({ ...prev, page: 1 })); // Reset to first page
  };

  const handleRegionToggle = (region) => {
    const currentRegions = filters.regions || [];
    const newRegions = currentRegions.includes(region)
      ? currentRegions.filter(r => r !== region)
      : [...currentRegions, region];
    setFilters(prev => ({ ...prev, regions: newRegions }));
    setPagination(prev => ({ ...prev, page: 1 })); // Reset to first page
  };

  const clearFilters = () => {
    setFilters({
      officer_id: '',
      branch: '',
      regions: [],
      wave: '',
      channel: '',
      status: '',
      customer_phone: '',
      vertical_lead_email: '',
      loan_type: '',
      rot_type: '',
      delay_type: '',
    });
    setFilterLabel('');
  };

  const handleLimitChange = (newLimit) => {
    setPagination(prev => ({ ...prev, limit: parseInt(newLimit), page: 1 }));
  };

  const handlePageChange = (newPage) => {
    setPagination(prev => ({ ...prev, page: newPage }));
  };

  const handleExportCSV = () => {
    const headers = [
      'Loan ID', 'Customer Name', 'Customer Phone', 'Officer Name', 'Region', 'Branch',
      'Vertical Lead Name', 'Vertical Lead Email',
      'Channel', 'Loan Amount', 'Repayment Amount', 'Disbursement Date', 'Loan Tenure', 'Maturity Date',
      'Daily Repayment Amount', 'Repayment Days Due Today', 'Repayment Days Paid', 'Business Days Since Disbursement',
      'Timeliness Score', 'Repayment Health', 'Repayment Delay Rate %', 'Wave', 'Days Since Last Repayment', 'Current DPD',
      'Principal Outstanding', 'Interest Outstanding', 'Fees Outstanding', 'Total Outstanding',
      'Actual Outstanding', 'Total Repayments', 'Status', 'FIMR Tagged'
    ];

    const rows = loans.map(loan => [
      loan.loan_id,
      loan.customer_name,
      loan.customer_phone || '',
      loan.officer_name,
      loan.region,
      loan.branch,
      loan.vertical_lead_name || 'N/A',
      loan.vertical_lead_email || 'N/A',
      loan.channel,
      loan.loan_amount,
      loan.repayment_amount || 'N/A',
      loan.disbursement_date,
      formatTenure(loan.loan_term_days),
      loan.maturity_date,
      loan.daily_repayment_amount != null ? loan.daily_repayment_amount.toFixed(2) : 'N/A',
      loan.repayment_days_due_today != null ? loan.repayment_days_due_today : 'N/A',
      loan.repayment_days_paid != null ? loan.repayment_days_paid.toFixed(2) : 'N/A',
      loan.business_days_since_disbursement != null ? loan.business_days_since_disbursement : 'N/A',
      loan.timeliness_score != null ? loan.timeliness_score.toFixed(2) : 'N/A',
      loan.repayment_health != null ? loan.repayment_health.toFixed(2) : 'N/A',
      loan.repayment_delay_rate != null ? loan.repayment_delay_rate.toFixed(2) : 'N/A',
      loan.wave || 'N/A',
      loan.days_since_last_repayment != null ? loan.days_since_last_repayment : 'N/A',
      loan.current_dpd,
      loan.principal_outstanding,
      loan.interest_outstanding,
      loan.fees_outstanding,
      loan.total_outstanding,
      loan.actual_outstanding || 0,
      loan.total_repayments || 0,
      loan.status,
      loan.fimr_tagged ? 'Yes' : 'No',
    ]);

    const csvContent = [
      headers.join(','),
      ...rows.map(row => row.map(cell => `"${cell}"`).join(','))
    ].join('\n');

    const blob = new Blob([csvContent], { type: 'text/csv;charset=utf-8;' });
    const link = document.createElement('a');
    const url = URL.createObjectURL(blob);
    link.setAttribute('href', url);
    link.setAttribute('download', `All_Loans_${new Date().toISOString().split('T')[0]}.csv`);
    link.style.visibility = 'hidden';
    document.body.appendChild(link);
    link.click();
    document.body.removeChild(link);
  };

  const handleExportPDF = () => {
    const doc = new jsPDF('landscape');

    doc.setFontSize(16);
    doc.text('All Loans Report', 14, 15);
    doc.setFontSize(10);
    doc.text(`Generated: ${new Date().toLocaleString()}`, 14, 22);
    doc.text(`Total Loans: ${pagination.total}`, 14, 27);

    const tableData = loans.map(loan => [
      loan.loan_id,
      loan.customer_name,
      loan.customer_phone || 'N/A',
      loan.officer_name,
      loan.branch,
      `₦${(loan.loan_amount / 1000000).toFixed(2)}M`,
      loan.repayment_amount ? `₦${(loan.repayment_amount / 1000000).toFixed(2)}M` : 'N/A',
      loan.disbursement_date,
      formatTenure(loan.loan_term_days),
      loan.daily_repayment_amount != null ? `₦${(loan.daily_repayment_amount / 1000).toFixed(1)}K` : 'N/A',
      loan.repayment_days_due_today != null ? loan.repayment_days_due_today : 'N/A',
      loan.repayment_days_paid != null ? loan.repayment_days_paid.toFixed(1) : 'N/A',
      loan.business_days_since_disbursement != null ? loan.business_days_since_disbursement : 'N/A',
      loan.timeliness_score != null ? loan.timeliness_score.toFixed(1) : 'N/A',
      loan.repayment_health != null ? loan.repayment_health.toFixed(1) : 'N/A',
      loan.repayment_delay_rate != null ? loan.repayment_delay_rate.toFixed(1) + '%' : 'N/A',
      loan.wave || 'N/A',
      loan.days_since_last_repayment != null ? loan.days_since_last_repayment : 'N/A',
      loan.current_dpd,
      `₦${(loan.total_outstanding / 1000000).toFixed(2)}M`,
      `₦${((loan.actual_outstanding || 0) / 1000000).toFixed(2)}M`,
      loan.status,
      loan.fimr_tagged ? 'Yes' : 'No',
    ]);

    doc.autoTable({
      startY: 32,
      head: [['Loan ID', 'Customer', 'Phone', 'Officer', 'Branch', 'Amount', 'Repay. Amt', 'Disbursed', 'Tenure', 'Daily Repay', 'Days Due', 'Days Paid', 'Biz Days', 'T.Score', 'R.Health', 'Delay %', 'Wave', 'Days Since', 'DPD', 'Total Out.', 'Actual Out.', 'Status', 'FIMR']],
      body: tableData,
      styles: { fontSize: 5 },
      headStyles: { fillColor: [41, 128, 185] },
    });

    doc.save(`All_Loans_${new Date().toISOString().split('T')[0]}.pdf`);
  };

  const formatCurrency = (value) => {
    return new Intl.NumberFormat('en-NG', {
      style: 'currency',
      currency: 'NGN',
      minimumFractionDigits: 0,
    }).format(value);
  };

  const formatDate = (dateString) => {
    if (!dateString) return '';
    const date = new Date(dateString);
    return date.toLocaleDateString('en-GB', {
      day: '2-digit',
      month: 'short',
      year: 'numeric'
    });
  };

  const formatTenure = (days) => {
    if (!days) return '';
    const months = Math.round(days / 30);
    return `${months} month${months !== 1 ? 's' : ''}`;
  };

  const activeFilterCount = Object.values(filters).filter(v => v !== '').length;

  const handleViewRepayments = (loan) => {
    setSelectedLoan(loan);
    setRepaymentsModalOpen(true);
  };

  const handleRecalculateFields = async () => {
    setRecalculating(true);
    setRecalculateMessage('');

    try {
      const API_BASE_URL = import.meta.env.VITE_API_URL || 'http://localhost:8081/api/v1';
      const response = await fetch(`${API_BASE_URL}/loans/recalculate-fields`, {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
        },
      });

      const result = await response.json();

      if (response.ok && result.status === 'success') {
        setRecalculateMessage(`✓ Successfully recalculated fields for ${result.data.loans_updated} loans`);
        // Refresh the loans table after successful recalculation
        await fetchLoans();

        // Clear success message after 5 seconds
        setTimeout(() => {
          setRecalculateMessage('');
        }, 5000);
      } else {
        setRecalculateMessage(`✗ Error: ${result.error?.message || 'Failed to recalculate fields'}`);

        // Clear error message after 5 seconds
        setTimeout(() => {
          setRecalculateMessage('');
        }, 5000);
      }
    } catch (error) {
      console.error('Error recalculating loan fields:', error);
      setRecalculateMessage(`✗ Error: ${error.message}`);

      // Clear error message after 5 seconds
      setTimeout(() => {
        setRecalculateMessage('');
      }, 5000);
    } finally {
      setRecalculating(false);
    }
  };

  return (
    <div className="all-loans">
      <div className="all-loans-header">
        <div className="all-loans-title">
          <h2>All Loans</h2>
          <span className="loan-count">{pagination.total} Total Loans</span>
          {filterLabel && (
            <span className="filter-label" style={{
              background: '#e3f2fd',
              color: '#1976d2',
              padding: '4px 12px',
              borderRadius: '12px',
              fontSize: '14px',
              fontWeight: '500'
            }}>
              {filterLabel}
            </span>
          )}
        </div>
        <div className="all-loans-actions">
          <button
            className={`filter-toggle ${showFilters ? 'active' : ''}`}
            onClick={() => setShowFilters(!showFilters)}
          >
            <Filter size={16} />
            Filters
            {activeFilterCount > 0 && (
              <span className="filter-badge">{activeFilterCount}</span>
            )}
          </button>
          <button
            className={`recalculate-button ${recalculating ? 'loading' : ''}`}
            onClick={handleRecalculateFields}
            disabled={recalculating}
            title="Recalculate all computed fields (actual_outstanding, total_outstanding, current_dpd, etc.)"
          >
            <RefreshCw size={16} className={recalculating ? 'spinning' : ''} />
            {recalculating ? 'Recalculating...' : 'Refresh Fields'}
          </button>
          <button className="export-button" onClick={handleExportCSV}>
            <Download size={16} />
            Export CSV
          </button>
          <button className="export-button" onClick={handleExportPDF}>
            <FileText size={16} />
            Export PDF
          </button>
        </div>
      </div>

      {recalculateMessage && (
        <div className={`recalculate-message ${recalculateMessage.startsWith('✓') ? 'success' : 'error'}`}>
          {recalculateMessage}
        </div>
      )}

      {showFilters && (
        <div className="filter-panel">
          <div className="filter-row">
            <div className="filter-group multi-select-wrapper" ref={regionDropdownRef}>
              <button
                className="multi-select-button"
                onClick={() => setIsRegionDropdownOpen(!isRegionDropdownOpen)}
              >
                <span>
                  {filters.regions && filters.regions.length > 0
                    ? `${filters.regions.length} Region${filters.regions.length > 1 ? 's' : ''} Selected`
                    : 'All Regions'}
                </span>
                <ChevronDown size={16} />
              </button>
              {isRegionDropdownOpen && (
                <div className="multi-select-dropdown">
                  <div className="multi-select-option" onClick={() => setFilters(prev => ({ ...prev, regions: [] }))}>
                    <input
                      type="checkbox"
                      checked={!filters.regions || filters.regions.length === 0}
                      readOnly
                    />
                    <span>All Regions</span>
                  </div>
                  {filterOptions.regions.map(region => (
                    <div
                      key={region}
                      className="multi-select-option"
                      onClick={() => handleRegionToggle(region)}
                    >
                      <input
                        type="checkbox"
                        checked={filters.regions && filters.regions.includes(region)}
                        readOnly
                      />
                      <span>{region}</span>
                    </div>
                  ))}
                </div>
              )}
            </div>
            <div className="filter-group">
              <select
                value={filters.branch}
                onChange={(e) => handleFilterChange('branch', e.target.value)}
              >
                <option value="">All Branches</option>
                {filterOptions.branches.map(branch => (
                  <option key={branch} value={branch}>{branch}</option>
                ))}
              </select>
            </div>
            <div className="filter-group">
              <select
                value={filters.wave}
                onChange={(e) => handleFilterChange('wave', e.target.value)}
              >
                <option value="">All Waves</option>
                {filterOptions.waves.map(wave => (
                  <option key={wave} value={wave}>{wave}</option>
                ))}
              </select>
            </div>
            <div className="filter-group">
              <select
                value={filters.channel}
                onChange={(e) => handleFilterChange('channel', e.target.value)}
              >
                <option value="">All Channels</option>
                {filterOptions.channels.map(channel => (
                  <option key={channel} value={channel}>{channel}</option>
                ))}
              </select>
            </div>
            <div className="filter-group">
              <select
                value={filters.vertical_lead_email}
                onChange={(e) => handleFilterChange('vertical_lead_email', e.target.value)}
              >
                <option value="">All Vertical Leads</option>
                {filterOptions.verticalLeads.map(email => (
                  <option key={email} value={email}>{email}</option>
                ))}
              </select>
            </div>
            <div className="filter-group">
              <select
                value={filters.status}
                onChange={(e) => handleFilterChange('status', e.target.value)}
              >
                <option value="">All Statuses</option>
                {filterOptions.statuses.map(status => (
                  <option key={status} value={status}>{status}</option>
                ))}
              </select>
            </div>
            <div className="filter-group">
              <input
                type="text"
                placeholder="Search by phone number..."
                value={filters.customer_phone}
                onChange={(e) => handleFilterChange('customer_phone', e.target.value)}
                className="phone-filter-input"
              />
            </div>
            <div className="filter-group">
              <button className="clear-filters" onClick={clearFilters}>Clear All</button>
            </div>
          </div>
        </div>
      )}

      <Pagination
        currentPage={pagination.page}
        totalPages={pagination.pages}
        totalRecords={pagination.total}
        pageSize={pagination.limit}
        onPageChange={handlePageChange}
        onPageSizeChange={handleLimitChange}
        pageSizeOptions={[10, 25, 50, 100, 200]}
        loading={loading}
        position="top"
      />

      {/* Show info message when client-side filtering is active */}
      {(filters.loan_type || filters.rot_type || filters.delay_type) && loans.length < pagination.limit && (
        <div className="client-filter-info">
          ℹ️ Showing {loans.length} loans after applying display filters.
          Total matching records: {pagination.total}
        </div>
      )}

      <div className="all-loans-table-container">
        {loading ? (
          <div className="loading">Loading...</div>
        ) : (
          <table className="all-loans-table">
            <thead>
              <tr>
                <th onClick={() => handleSort('loan_id')}>Loan ID</th>
                <th onClick={() => handleSort('customer_name')}>Customer Name</th>
                <th onClick={() => handleSort('customer_phone')}>Customer Phone</th>
                <th onClick={() => handleSort('officer_name')}>Officer Name</th>
                <th onClick={() => handleSort('region')}>Region</th>
                <th onClick={() => handleSort('branch')}>Branch</th>
                <th onClick={() => handleSort('vertical_lead_name')}>Vertical Lead Name</th>
                <th onClick={() => handleSort('vertical_lead_email')}>Vertical Lead Email</th>
                <th onClick={() => handleSort('channel')}>Channel</th>
                <th onClick={() => handleSort('loan_amount')}>Loan Amount</th>
                <th onClick={() => handleSort('repayment_amount')}>Repayment Amount</th>
                <th onClick={() => handleSort('disbursement_date')}>Disbursement Date</th>
                <th onClick={() => handleSort('first_payment_due_date')}>First Payment Due</th>
                <th onClick={() => handleSort('loan_term_days')}>Loan Tenure</th>
                <th onClick={() => handleSort('daily_repayment_amount')}>Daily Repayment Amount</th>
                <th onClick={() => handleSort('repayment_days_due_today')}>Repayment Days Due Today</th>
                <th onClick={() => handleSort('repayment_days_paid')}>Repayment Days Paid</th>
                <th onClick={() => handleSort('business_days_since_disbursement')}>Business Days Since Disbursement</th>
                <th onClick={() => handleSort('timeliness_score')}>Timeliness Score</th>
                <th onClick={() => handleSort('repayment_health')}>Repayment Health</th>
                <th onClick={() => handleSort('repayment_delay_rate')}>Repayment Delay Rate %</th>
                <th onClick={() => handleSort('wave')}>Wave</th>
                <th onClick={() => handleSort('days_since_last_repayment')}>Days Since Last Repayment</th>
                <th onClick={() => handleSort('current_dpd')}>Current DPD</th>
                <th onClick={() => handleSort('total_outstanding')}>Total Outstanding</th>
                <th onClick={() => handleSort('actual_outstanding')}>Actual Outstanding</th>
              <th onClick={() => handleSort('total_repayments')}>Total Repayments</th>
                <th onClick={() => handleSort('status')}>Status</th>
                <th>FIMR Tagged</th>
                <th>Actions</th>
              </tr>
            </thead>
            <tbody>
              {loans.map((loan) => (
                <tr key={loan.loan_id}>
                  <td className="loan-id">{loan.loan_id}</td>
                  <td>{loan.customer_name}</td>
                  <td>{loan.customer_phone || 'N/A'}</td>
                  <td>{loan.officer_name}</td>
                  <td>{loan.region}</td>
                  <td>{loan.branch}</td>
                  <td>{loan.vertical_lead_name || 'N/A'}</td>
                  <td>{loan.vertical_lead_email || 'N/A'}</td>
                  <td>{loan.channel}</td>
                  <td className="amount">{formatCurrency(loan.loan_amount)}</td>
                  <td className="amount">{loan.repayment_amount ? formatCurrency(loan.repayment_amount) : 'N/A'}</td>
                  <td>{formatDate(loan.disbursement_date)}</td>
                  <td>{loan.first_payment_due_date ? formatDate(loan.first_payment_due_date) : 'N/A'}</td>
                  <td className="tenure">{formatTenure(loan.loan_term_days)}</td>
                  <td className="amount">{loan.daily_repayment_amount != null ? formatCurrency(loan.daily_repayment_amount) : 'N/A'}</td>
                  <td className="days-since">{loan.repayment_days_due_today != null ? loan.repayment_days_due_today + ' days' : 'N/A'}</td>
                  <td className="days-since">{loan.repayment_days_paid != null ? loan.repayment_days_paid.toFixed(2) + ' days' : 'N/A'}</td>
                  <td className="days-since">{loan.business_days_since_disbursement != null ? loan.business_days_since_disbursement + ' days' : 'N/A'}</td>
                  <td className="score">{loan.timeliness_score != null ? loan.timeliness_score.toFixed(2) : 'N/A'}</td>
                  <td className="score">{loan.repayment_health != null ? loan.repayment_health.toFixed(2) : 'N/A'}</td>
                  <td className="delay-rate" style={{
                    color: loan.repayment_delay_rate != null
                      ? (loan.repayment_delay_rate >= 60 ? '#2e7d32'
                        : loan.repayment_delay_rate >= 30 ? '#f57c00'
                        : '#c62828')
                      : 'inherit',
                    fontWeight: loan.repayment_delay_rate != null ? '600' : 'normal'
                  }}>
                    {loan.repayment_delay_rate != null ? loan.repayment_delay_rate.toFixed(2) + '%' : 'N/A'}
                  </td>
                  <td className="wave">
                    <span className={`wave-badge wave-${loan.wave?.replace(' ', '-').toLowerCase()}`}>
                      {loan.wave || 'N/A'}
                    </span>
                  </td>
                  <td className="days-since">{loan.days_since_last_repayment != null ? loan.days_since_last_repayment : 'N/A'}</td>
                  <td className="dpd">{loan.current_dpd}</td>
                  <td className="amount">{formatCurrency(loan.total_outstanding)}</td>
                  <td className="amount">{formatCurrency(loan.actual_outstanding || 0)}</td>
                  <td className="amount">{formatCurrency(loan.total_repayments || 0)}</td>
                  <td>
                    <span className={`status-badge status-${loan.status.toLowerCase()}`}>
                      {loan.status}
                    </span>
                  </td>
                  <td className="fimr-tagged">
                    {loan.fimr_tagged ? (
                      <span className="badge-yes">Yes</span>
                    ) : (
                      <span className="badge-no">No</span>
                    )}
                  </td>
                  <td className="action-cell">
                    <button
                      className="view-repayments-btn"
                      onClick={() => handleViewRepayments(loan)}
                      title="View Repayment History"
                    >
                      <Eye size={16} />
                      Repayments
                    </button>
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        )}
      </div>

      <Pagination
        currentPage={pagination.page}
        totalPages={pagination.pages}
        totalRecords={pagination.total}
        pageSize={pagination.limit}
        onPageChange={handlePageChange}
        onPageSizeChange={handleLimitChange}
        pageSizeOptions={[10, 25, 50, 100, 200]}
        loading={loading}
        position="bottom"
      />

      {/* Repayments Modal */}
      <LoanRepaymentsModal
        isOpen={repaymentsModalOpen}
        onClose={() => setRepaymentsModalOpen(false)}
        loanId={selectedLoan?.loan_id || ''}
        customerName={selectedLoan?.customer_name || ''}
      />
    </div>
  );
};

export default AllLoans;

