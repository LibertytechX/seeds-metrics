import React, { useState, useEffect, useMemo, useRef } from 'react';
import { Download, Filter, FileText, Eye, RefreshCw, ChevronDown } from 'lucide-react';
import jsPDF from 'jspdf';
import 'jspdf-autotable';
import LoanRepaymentsModal from './LoanRepaymentsModal';
import Pagination from './Pagination';
import SearchableSelect from './SearchableSelect';
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
    statuses: [], // Multi-select status filter
    performance_statuses: [], // Multi-select performance status filter
    customer_phone: '',
    vertical_lead_email: '',
    loan_type: initialFilter?.loan_type || '', // 'active' or 'inactive'
    rot_type: initialFilter?.rot_type || '', // 'early' or 'late'
    delay_type: initialFilter?.delay_type || '', // 'risky' for high delay loans
    dpd_min: '', // DPD minimum value
    dpd_max: '', // DPD maximum value
    django_loan_types: [], // Multi-select loan types from Django (AJO, BNPL, PROSPER, DMO)
    django_verification_statuses: [], // Multi-select verification statuses from Django
  });
  const [allBranches, setAllBranches] = useState([]);
  const [allRegions, setAllRegions] = useState([]);
  const [allChannels, setAllChannels] = useState([]);
  const [allWaves, setAllWaves] = useState([]);
  const [allStatuses, setAllStatuses] = useState([]);
  const [allPerformanceStatuses, setAllPerformanceStatuses] = useState([]);
  const [allVerticalLeads, setAllVerticalLeads] = useState([]);
  const [allOfficers, setAllOfficers] = useState([]);
  const [allLoanTypes, setAllLoanTypes] = useState([]);
  const [allVerificationStatuses, setAllVerificationStatuses] = useState([]);
  const [isRegionDropdownOpen, setIsRegionDropdownOpen] = useState(false);
  const regionDropdownRef = useRef(null);
  const [isStatusDropdownOpen, setIsStatusDropdownOpen] = useState(false);
  const statusDropdownRef = useRef(null);
  const [isPerformanceStatusDropdownOpen, setIsPerformanceStatusDropdownOpen] = useState(false);
  const performanceStatusDropdownRef = useRef(null);
  const [isLoanTypeDropdownOpen, setIsLoanTypeDropdownOpen] = useState(false);
  const loanTypeDropdownRef = useRef(null);
  const [isVerificationStatusDropdownOpen, setIsVerificationStatusDropdownOpen] = useState(false);
  const verificationStatusDropdownRef = useRef(null);
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
  const [exporting, setExporting] = useState(false);
  const [refreshingLoans, setRefreshingLoans] = useState(new Set()); // Track which loans are being refreshed
  const [refreshMessage, setRefreshMessage] = useState(''); // Success/error message for refresh
  const [summaryMetrics, setSummaryMetrics] = useState({
    totalLoans: 0,
    totalPortfolioAmount: 0,
    atRiskLoans: { count: 0, amount: 0, actualOutstanding: 0, percentage: 0 },
    totalAmountInDPD: 0,
    criticalLoans: { count: 0, percentage: 0 },
    repaymentDelayCategories: { excellent: 0, okay: 0, critical: 0 },
    totalDueForToday: 0,
    totalRepaymentsToday: 0,
    percentageOfDueCollected: 0
  });

  // Close dropdown when clicking outside
  useEffect(() => {
    const handleClickOutside = (event) => {
      if (regionDropdownRef.current && !regionDropdownRef.current.contains(event.target)) {
        setIsRegionDropdownOpen(false);
      }
      if (statusDropdownRef.current && !statusDropdownRef.current.contains(event.target)) {
        setIsStatusDropdownOpen(false);
      }
      if (performanceStatusDropdownRef.current && !performanceStatusDropdownRef.current.contains(event.target)) {
        setIsPerformanceStatusDropdownOpen(false);
      }
      if (loanTypeDropdownRef.current && !loanTypeDropdownRef.current.contains(event.target)) {
        setIsLoanTypeDropdownOpen(false);
      }
      if (verificationStatusDropdownRef.current && !verificationStatusDropdownRef.current.contains(event.target)) {
        setIsVerificationStatusDropdownOpen(false);
      }
    };
    document.addEventListener('mousedown', handleClickOutside);
    return () => document.removeEventListener('mousedown', handleClickOutside);
  }, []);

  // Fetch filter options from API
  const fetchFilterOptions = async () => {
    try {
      const API_BASE_URL = import.meta.env.VITE_API_URL || 'http://localhost:8081/api/v1';

      // Fetch branches, regions, channels, waves, statuses, officers, loan types, and verification statuses from API
      const [branchesRes, regionsRes, channelsRes, wavesRes, statusesRes, officersRes, loanTypesRes, verificationStatusesRes] = await Promise.all([
        fetch(`${API_BASE_URL}/filters/branches`),
        fetch(`${API_BASE_URL}/filters/regions`),
        fetch(`${API_BASE_URL}/filters/channels`),
        fetch(`${API_BASE_URL}/filters/waves`),
        fetch(`${API_BASE_URL}/filters/statuses`),
        fetch(`${API_BASE_URL}/filters/officers`),
        fetch(`${API_BASE_URL}/filters/loan-types`),
        fetch(`${API_BASE_URL}/filters/verification-statuses`),
      ]);

      const [branchesData, regionsData, channelsData, wavesData, statusesData, officersData, loanTypesData, verificationStatusesData] = await Promise.all([
        branchesRes.json(),
        regionsRes.json(),
        channelsRes.json(),
        wavesRes.json(),
        statusesRes.json(),
        officersRes.json(),
        loanTypesRes.json(),
        verificationStatusesRes.json(),
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
      if (statusesData.status === 'success') {
        setAllStatuses(statusesData.data.statuses || []);
      }
      if (officersData.status === 'success') {
        setAllOfficers(officersData.data.officers || []);
      }
      if (loanTypesData.status === 'success') {
        setAllLoanTypes(loanTypesData.data['loan-types'] || []);
      }
      if (verificationStatusesData.status === 'success') {
        setAllVerificationStatuses(verificationStatusesData.data['verification-statuses'] || []);
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

      // Exclude loan_type, rot_type, delay_type, regions, statuses, performance_statuses, django_loan_types, and django_verification_statuses from API params (will handle arrays separately)
      // Include customer_phone and DPD filters for server-side filtering
      const apiFilters = Object.fromEntries(
        Object.entries(filters)
          .filter(([k, v]) => v !== '' && k !== 'loan_type' && k !== 'rot_type' && k !== 'delay_type' && k !== 'regions' && k !== 'statuses' && k !== 'performance_statuses' && k !== 'django_loan_types' && k !== 'django_verification_statuses')
      );

      // Add django_loan_types as loan_type for API (comma-separated)
      if (filters.django_loan_types && filters.django_loan_types.length > 0) {
        apiFilters.loan_type = filters.django_loan_types.join(',');
      }

      // Add django_verification_statuses as verification_status for API (comma-separated)
      if (filters.django_verification_statuses && filters.django_verification_statuses.length > 0) {
        apiFilters.verification_status = filters.django_verification_statuses.join(',');
      }

      // Map behaviour-based filters to backend params so server, table, and exports match
      if (filters.loan_type) {
        apiFilters.behavior_loan_type = filters.loan_type;
      }
      if (filters.rot_type) {
        apiFilters.rot_type = filters.rot_type;
      }
      if (filters.delay_type) {
        apiFilters.delay_type = filters.delay_type;
      }

      // Convert regions array to comma-separated string
      if (filters.regions && filters.regions.length > 0) {
        apiFilters.region = filters.regions.join(',');
      }

      // Convert statuses array to comma-separated string
      if (filters.statuses && filters.statuses.length > 0) {
        apiFilters.status = filters.statuses.join(',');
      }

      // Convert performance_statuses array to comma-separated string
      if (filters.performance_statuses && filters.performance_statuses.length > 0) {
        apiFilters.performance_status = filters.performance_statuses.join(',');
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
        const fetchedLoans = data.data.loans || [];
        console.log(`📦 AllLoans: Fetched ${fetchedLoans.length} loans from API (already fully filtered server-side)`);

        // All behavior-based filters (active/inactive/overdue_15d, early/late ROT,
        // risky delay) are now handled on the backend so that the table,
        // pagination totals, summary metrics, and CSV exports all use the same
        // set of loans.

        setLoans(fetchedLoans);

        // Extract unique vertical leads from fetched loans
        const uniqueVerticalLeads = [...new Set(
          fetchedLoans
            .map(loan => loan.vertical_lead_email)
            .filter(email => email != null && email !== '')
        )].sort();
        setAllVerticalLeads(uniqueVerticalLeads);

        // Note: loan types and verification statuses are now fetched from API in fetchFilterOptions()
        // No need to extract them from fetched loans

        // Use backend total for pagination, not client-side filtered count
        // The backend total represents the actual number of records matching server-side filters
        // Client-side filtering (loan_type, rot_type, delay_type) is for display only
        setPagination({
          page: data.data.page,
          limit: data.data.limit,
          total: data.data.total, // Use backend total, not fetchedLoans.length
          pages: data.data.pages, // Use backend calculated pages
        });

        // Use summary metrics from backend (calculated from ALL filtered loans, not just current page)
        if (data.data.summary_metrics) {
          setSummaryMetrics({
            totalLoans: data.data.summary_metrics.total_loans,
            totalPortfolioAmount: data.data.summary_metrics.total_portfolio_amount,
            atRiskLoans: {
              count: data.data.summary_metrics.at_risk_loans.count,
              amount: data.data.summary_metrics.at_risk_loans.amount,
              actualOutstanding: data.data.summary_metrics.at_risk_loans.actual_outstanding,
              percentage: data.data.summary_metrics.at_risk_loans.percentage
            },
            totalAmountInDPD: data.data.summary_metrics.total_amount_in_dpd,
            criticalLoans: {
              count: data.data.summary_metrics.critical_loans.count,
              percentage: data.data.summary_metrics.critical_loans.percentage
            },
            repaymentDelayCategories: {
              excellent: data.data.summary_metrics.repayment_delay_categories.excellent,
              okay: data.data.summary_metrics.repayment_delay_categories.okay,
              critical: data.data.summary_metrics.repayment_delay_categories.critical
            },
            totalDueForToday: data.data.summary_metrics.total_due_for_today || 0,
            totalRepaymentsToday: data.data.summary_metrics.total_repayments_today || 0,
            percentageOfDueCollected: data.data.summary_metrics.percentage_of_due_collected || 0
          });
        }
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
        statuses: [],
        performance_statuses: [],
        customer_phone: '',
        vertical_lead_email: '',
        loan_type: initialFilter.loan_type || '',
        rot_type: initialFilter.rot_type || '',
        delay_type: initialFilter.delay_type || '',
        dpd_min: '',
        dpd_max: '',
        django_loan_types: [],
        django_verification_statuses: [],
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
    // Define performance status options
    const performanceStatusOptions = ['PERFORMING', 'DEFAULTED', 'LOST', 'PAST_MATURITY', 'OWED_BALANCE'];

    return {
      officers: allOfficers.length > 0 ? allOfficers : [...new Set(loans.map(l => l.officer_name))].filter(Boolean).sort(),
      branches: allBranches.length > 0 ? allBranches.sort() : [...new Set(loans.map(l => l.branch))].filter(Boolean).sort(),
      regions: allRegions.length > 0 ? allRegions.sort() : [...new Set(loans.map(l => l.region))].filter(Boolean).sort(),
      waves: allWaves.length > 0 ? allWaves.sort() : [...new Set(loans.map(l => l.wave))].filter(Boolean).sort(),
      channels: allChannels.length > 0 ? allChannels.sort() : [...new Set(loans.map(l => l.channel))].filter(Boolean).sort(),
      statuses: allStatuses.length > 0 ? allStatuses.sort() : [...new Set(loans.map(l => l.status))].filter(Boolean).sort(),
      performanceStatuses: performanceStatusOptions,
      verticalLeads: allVerticalLeads.length > 0 ? allVerticalLeads : [...new Set(loans.map(l => l.vertical_lead_email))].filter(Boolean).sort(),
    };
  }, [loans, allBranches, allRegions, allWaves, allChannels, allStatuses, allPerformanceStatuses, allVerticalLeads, allOfficers]);

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

  const handleStatusToggle = (status) => {
    const currentStatuses = filters.statuses || [];
    const newStatuses = currentStatuses.includes(status)
      ? currentStatuses.filter(s => s !== status)
      : [...currentStatuses, status];
    setFilters(prev => ({ ...prev, statuses: newStatuses }));
    setPagination(prev => ({ ...prev, page: 1 })); // Reset to first page
  };

  const handlePerformanceStatusToggle = (performanceStatus) => {
    const currentPerformanceStatuses = filters.performance_statuses || [];
    const newPerformanceStatuses = currentPerformanceStatuses.includes(performanceStatus)
      ? currentPerformanceStatuses.filter(ps => ps !== performanceStatus)
      : [...currentPerformanceStatuses, performanceStatus];
    setFilters(prev => ({ ...prev, performance_statuses: newPerformanceStatuses }));
    setPagination(prev => ({ ...prev, page: 1 })); // Reset to first page
  };

  const handleLoanTypeToggle = (loanType) => {
    const currentLoanTypes = filters.django_loan_types || [];
    const newLoanTypes = currentLoanTypes.includes(loanType)
      ? currentLoanTypes.filter(lt => lt !== loanType)
      : [...currentLoanTypes, loanType];
    setFilters(prev => ({ ...prev, django_loan_types: newLoanTypes }));
    setPagination(prev => ({ ...prev, page: 1 })); // Reset to first page
  };

  const handleVerificationStatusToggle = (verificationStatus) => {
    const currentVerificationStatuses = filters.django_verification_statuses || [];
    const newVerificationStatuses = currentVerificationStatuses.includes(verificationStatus)
      ? currentVerificationStatuses.filter(vs => vs !== verificationStatus)
      : [...currentVerificationStatuses, verificationStatus];
    setFilters(prev => ({ ...prev, django_verification_statuses: newVerificationStatuses }));
    setPagination(prev => ({ ...prev, page: 1 })); // Reset to first page
  };

  const clearFilters = () => {
    setFilters({
      officer_id: '',
      branch: '',
      regions: [],
      wave: '',
      channel: '',
      statuses: [],
      performance_statuses: [],
      customer_phone: '',
      vertical_lead_email: '',
      loan_type: '',
      rot_type: '',
      delay_type: '',
      dpd_min: '',
      dpd_max: '',
      django_loan_types: [],
      django_verification_statuses: [],
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
      'Channel', 'Loan Type', 'Verification Status', 'Loan Amount', 'Repayment Amount', 'Disbursement Date',
      'First Repayment Due Date', 'Loan Tenure', 'Maturity Date',
      'Daily Repayment Amount', 'Repayment Days Due Today', 'Repayment Days Paid', 'Business Days Since Disbursement',
      'Timeliness Score', 'Repayment Health', 'Repayment Delay Rate %', 'Wave', 'Days Since Last Repayment', 'Current DPD',
      'Principal Outstanding', 'Interest Outstanding', 'Fees Outstanding', 'Total Outstanding',
      'Actual Outstanding', 'Total Repayments', 'Status', 'Performance Status', 'FIMR Tagged'
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
      loan.loan_type || 'N/A',
      loan.verification_status || 'N/A',
      loan.loan_amount,
      loan.repayment_amount || 'N/A',
      loan.disbursement_date,
      loan.first_payment_due_date || 'N/A',
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
      loan.performance_status || 'N/A',
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

  // Export all filtered rows (no limit)
  const handleExportLargeCSV = async () => {
    // First, check the total count and confirm if it's very large
    const totalCount = pagination.total;

    if (totalCount > 10000) {
      const confirmExport = window.confirm(
        `You are about to export ${totalCount.toLocaleString()} loans. This may take some time.\n\nDo you want to continue?`
      );
      if (!confirmExport) {
        return;
      }
    }

    setExporting(true);
    try {
      console.log('🔍 Exporting all filtered loans. Total count:', totalCount);

      // Prepare API filters (same logic as fetchLoans)
      const apiFilters = Object.fromEntries(
        Object.entries(filters).filter(([k, v]) => v !== '' && k !== 'loan_type' && k !== 'rot_type' && k !== 'delay_type' && k !== 'regions' && k !== 'statuses' && k !== 'performance_statuses' && k !== 'django_loan_types' && k !== 'django_verification_statuses')
      );

      // Convert regions array to comma-separated string
      if (filters.regions && filters.regions.length > 0) {
        apiFilters.region = filters.regions.join(',');
      }

      // Convert statuses array to comma-separated string
      if (filters.statuses && filters.statuses.length > 0) {
        apiFilters.status = filters.statuses.join(',');
      }

      // Convert performance_statuses array to comma-separated string
      if (filters.performance_statuses && filters.performance_statuses.length > 0) {
        apiFilters.performance_status = filters.performance_statuses.join(',');
      }

      // Convert django_loan_types array to comma-separated string
      if (filters.django_loan_types && filters.django_loan_types.length > 0) {
        apiFilters.loan_type = filters.django_loan_types.join(',');
      }

      // Convert django_verification_statuses array to comma-separated string
      if (filters.django_verification_statuses && filters.django_verification_statuses.length > 0) {
        apiFilters.verification_status = filters.django_verification_statuses.join(',');
      }

      // Pass behaviour-based filters through to backend so export uses identical logic
      if (filters.loan_type) {
        apiFilters.behavior_loan_type = filters.loan_type;
      }
      if (filters.rot_type) {
        apiFilters.rot_type = filters.rot_type;
      }
      if (filters.delay_type) {
        apiFilters.delay_type = filters.delay_type;
      }

      const params = new URLSearchParams({
        page: 1,
        limit: 999999, // Very high limit to fetch all results
        sort_by: sortConfig.key,
        sort_dir: sortConfig.direction.toUpperCase(),
        ...apiFilters,
      });

      const API_BASE_URL = import.meta.env.VITE_API_URL || 'http://localhost:8081/api/v1';
      const response = await fetch(`${API_BASE_URL}/loans?${params}`);
      const data = await response.json();

      if (data.status === 'success') {
        const exportLoans = data.data.loans || [];
        console.log(`📦 Fetched ${exportLoans.length} loans for export (fully filtered by backend)`);

        // No extra client-side filtering herebackend already applied
        // loan_type/rot_type/delay_type behaviour filters so that the CSV
        // exactly matches what the dashboard considers "filtered loans".

        // Generate CSV
        const headers = [
          'Loan ID', 'Customer Name', 'Customer Phone', 'Officer Name', 'Region', 'Branch',
          'Vertical Lead Name', 'Vertical Lead Email',
          'Channel', 'Loan Type', 'Verification Status', 'Loan Amount', 'Repayment Amount', 'Disbursement Date',
          'First Repayment Due Date', 'Loan Tenure', 'Maturity Date',
          'Daily Repayment Amount', 'Repayment Days Due Today', 'Repayment Days Paid', 'Business Days Since Disbursement',
          'Timeliness Score', 'Repayment Health', 'Repayment Delay Rate %', 'Wave', 'Days Since Last Repayment', 'Current DPD',
          'Principal Outstanding', 'Interest Outstanding', 'Fees Outstanding', 'Total Outstanding',
          'Actual Outstanding', 'Total Repayments', 'Status', 'Performance Status', 'FIMR Tagged'
        ];

        const rows = exportLoans.map(loan => [
          loan.loan_id,
          loan.customer_name,
          loan.customer_phone || '',
          loan.officer_name,
          loan.region,
          loan.branch,
          loan.vertical_lead_name || 'N/A',
          loan.vertical_lead_email || 'N/A',
          loan.channel,
          loan.loan_type || 'N/A',
          loan.verification_status || 'N/A',
          loan.loan_amount,
          loan.repayment_amount || 'N/A',
          loan.disbursement_date,
          loan.first_payment_due_date || 'N/A',
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
          loan.performance_status || 'N/A',
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
        link.setAttribute('download', `All_Loans_Export_${new Date().toISOString().split('T')[0]}.csv`);
        link.style.visibility = 'hidden';
        document.body.appendChild(link);
        link.click();
        document.body.removeChild(link);

        console.log(`✅ Exported ${exportLoans.length} loans to CSV`);

        // Show success message
        alert(`✅ Successfully exported ${exportLoans.length.toLocaleString()} loans to CSV!`);
      } else {
        throw new Error(data.message || 'Failed to fetch loans');
      }
    } catch (error) {
      console.error('Error exporting loans:', error);
      alert(`❌ Failed to export loans: ${error.message}\n\nPlease try again or contact support if the issue persists.`);
    } finally {
      setExporting(false);
    }
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

  const handleRefreshRepayments = async (loan) => {
    const loanId = loan.loan_id;

    // Add loan to refreshing set
    setRefreshingLoans(prev => new Set(prev).add(loanId));
    setRefreshMessage('');

    try {
      const API_BASE_URL = import.meta.env.VITE_API_URL || 'http://localhost:8081/api/v1';
      const response = await fetch(`${API_BASE_URL}/loans/${loanId}/sync-repayments`, {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
        },
      });

      const data = await response.json();

      if (!response.ok) {
        throw new Error(data.message || 'Failed to sync repayments');
      }

      // Update the loan in the loans array with the updated data
      if (data.data && data.data.updated_loan) {
        setLoans(prevLoans =>
          prevLoans.map(l =>
            l.loan_id === loanId ? data.data.updated_loan : l
          )
        );
      }

      setRefreshMessage(`✅ Successfully synced ${data.data.total_synced} repayment(s) for loan ${loanId}`);

      // Clear message after 5 seconds
      setTimeout(() => setRefreshMessage(''), 5000);
    } catch (error) {
      console.error('Error syncing repayments:', error);
      setRefreshMessage(`❌ Failed to sync repayments: ${error.message}`);

      // Clear error message after 5 seconds
      setTimeout(() => setRefreshMessage(''), 5000);
    } finally {
      // Remove loan from refreshing set
      setRefreshingLoans(prev => {
        const newSet = new Set(prev);
        newSet.delete(loanId);
        return newSet;
      });
    }
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

      // Handle 202 Accepted (async processing) or 200 OK (immediate success)
      if ((response.status === 202 || response.ok) && result.status === 'success') {
        setRecalculateMessage(`✓ ${result.message || 'Recalculation started successfully'}`);

        // If it's async (202), don't refresh immediately
        if (response.status !== 202) {
          // Refresh the loans table after successful recalculation
          await fetchLoans();
        }

        // Clear success message after 10 seconds
        setTimeout(() => {
          setRecalculateMessage('');
        }, 10000);
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
          <button
            className={`export-button ${exporting ? 'loading' : ''}`}
            onClick={handleExportLargeCSV}
            disabled={exporting}
            title={`Export all ${pagination.total.toLocaleString()} filtered loans to CSV`}
          >
            <Download size={16} />
            {exporting ? `Exporting ${pagination.total.toLocaleString()} loans...` : `Export All Filtered (${pagination.total.toLocaleString()})`}
          </button>
          <button className="export-button" onClick={handleExportCSV}>
            <Download size={16} />
            Export Current Page
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

      {refreshMessage && (
        <div className={`recalculate-message ${refreshMessage.startsWith('✅') ? 'success' : 'error'}`}>
          {refreshMessage}
        </div>
      )}

      {/* Summary Metrics Section */}
      <div className="summary-metrics">
        <div className="summary-card">
          <div className="summary-label">Total Portfolio Amount</div>
          <div className="summary-value">₦{(summaryMetrics.totalPortfolioAmount / 1000000).toFixed(2)}M</div>
          <div className="summary-detail">
            Due Today: ₦{summaryMetrics.totalDueForToday.toLocaleString('en-NG', { minimumFractionDigits: 2, maximumFractionDigits: 2 })} |
            Collected: ₦{summaryMetrics.totalRepaymentsToday.toLocaleString('en-NG', { minimumFractionDigits: 2, maximumFractionDigits: 2 })} |
            Collection Rate: {summaryMetrics.percentageOfDueCollected.toFixed(2)}%
          </div>
        </div>
        <div className="summary-card at-risk">
          <div className="summary-label">At Risk Loans (DPD &gt; 14)</div>
          <div className="summary-value">{summaryMetrics.atRiskLoans.count} loans ({summaryMetrics.atRiskLoans.percentage.toFixed(1)}%)</div>
          <div className="summary-detail">
            Amount: ₦{(summaryMetrics.atRiskLoans.amount / 1000000).toFixed(2)}M |
            Outstanding: ₦{(summaryMetrics.atRiskLoans.actualOutstanding / 1000000).toFixed(2)}M
          </div>
        </div>
        <div className="summary-card">
          <div className="summary-label">Total Amount in DPD</div>
          <div className="summary-value">₦{(summaryMetrics.totalAmountInDPD / 1000000).toFixed(2)}M</div>
        </div>
        <div className="summary-card critical">
          <div className="summary-label">Critical Loans (DPD &gt; 21)</div>
          <div className="summary-value">{summaryMetrics.criticalLoans.count} loans ({summaryMetrics.criticalLoans.percentage.toFixed(1)}%)</div>
        </div>
        <div className="summary-card delay-categories">
          <div className="summary-label">Repayment Delay Rate</div>
          <div className="summary-detail">
            <span className="delay-excellent">Excellent (≥80%): {summaryMetrics.repaymentDelayCategories.excellent}</span> |
            <span className="delay-okay">Okay (40-79%): {summaryMetrics.repaymentDelayCategories.okay}</span> |
            <span className="delay-critical">Critical (≤39%): {summaryMetrics.repaymentDelayCategories.critical}</span>
          </div>
        </div>
      </div>

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
              <SearchableSelect
                options={filterOptions.officers}
                selectedValue={filters.officer_id}
                onChange={(value) => handleFilterChange('officer_id', value)}
                placeholder="All Loan Officers"
                getOptionLabel={(officer) => {
                  // Handle both API response format (object) and fallback format (string)
                  if (typeof officer === 'object') {
                    return officer.email
                      ? `${officer.name} (${officer.email})`
                      : officer.name;
                  } else {
                    return officer;
                  }
                }}
                getOptionValue={(officer) => {
                  // Handle both API response format (object) and fallback format (string)
                  if (typeof officer === 'object') {
                    return officer.officer_id;
                  } else {
                    return officer;
                  }
                }}
              />
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
            <div className="filter-group multi-select-wrapper" ref={loanTypeDropdownRef}>
              <button
                className="multi-select-button"
                onClick={() => setIsLoanTypeDropdownOpen(!isLoanTypeDropdownOpen)}
              >
                <span>
                  {filters.django_loan_types && filters.django_loan_types.length > 0
                    ? `${filters.django_loan_types.length} Loan Type${filters.django_loan_types.length > 1 ? 's' : ''} Selected`
                    : 'All Loan Types'}
                </span>
                <ChevronDown size={16} />
              </button>
              {isLoanTypeDropdownOpen && (
                <div className="multi-select-dropdown">
                  <div className="multi-select-option" onClick={() => setFilters(prev => ({ ...prev, django_loan_types: [] }))}>
                    <input
                      type="checkbox"
                      checked={!filters.django_loan_types || filters.django_loan_types.length === 0}
                      readOnly
                    />
                    <span>All Loan Types</span>
                  </div>
                  {allLoanTypes.map(type => (
                    <div
                      key={type}
                      className="multi-select-option"
                      onClick={() => handleLoanTypeToggle(type)}
                    >
                      <input
                        type="checkbox"
                        checked={filters.django_loan_types && filters.django_loan_types.includes(type)}
                        readOnly
                      />
                      <span>{type}</span>
                    </div>
                  ))}
                </div>
              )}
            </div>
            <div className="filter-group multi-select-wrapper" ref={verificationStatusDropdownRef}>
              <button
                className="multi-select-button"
                onClick={() => setIsVerificationStatusDropdownOpen(!isVerificationStatusDropdownOpen)}
              >
                <span>
                  {filters.django_verification_statuses && filters.django_verification_statuses.length > 0
                    ? `${filters.django_verification_statuses.length} Verification Status${filters.django_verification_statuses.length > 1 ? 'es' : ''} Selected`
                    : 'All Verification Statuses'}
                </span>
                <ChevronDown size={16} />
              </button>
              {isVerificationStatusDropdownOpen && (
                <div className="multi-select-dropdown">
                  <div className="multi-select-option" onClick={() => setFilters(prev => ({ ...prev, django_verification_statuses: [] }))}>
                    <input
                      type="checkbox"
                      checked={!filters.django_verification_statuses || filters.django_verification_statuses.length === 0}
                      readOnly
                    />
                    <span>All Verification Statuses</span>
                  </div>
                  {allVerificationStatuses.map(status => (
                    <div
                      key={status}
                      className="multi-select-option"
                      onClick={() => handleVerificationStatusToggle(status)}
                    >
                      <input
                        type="checkbox"
                        checked={filters.django_verification_statuses && filters.django_verification_statuses.includes(status)}
                        readOnly
                      />
                      <span>{status}</span>
                    </div>
                  ))}
                </div>
              )}
            </div>
            <div className="filter-group multi-select-wrapper" ref={statusDropdownRef}>
              <button
                className="multi-select-button"
                onClick={() => setIsStatusDropdownOpen(!isStatusDropdownOpen)}
              >
                <span>
                  {filters.statuses && filters.statuses.length > 0
                    ? `${filters.statuses.length} Status${filters.statuses.length > 1 ? 'es' : ''} Selected`
                    : 'All Statuses'}
                </span>
                <ChevronDown size={16} />
              </button>
              {isStatusDropdownOpen && (
                <div className="multi-select-dropdown">
                  <div className="multi-select-option" onClick={() => setFilters(prev => ({ ...prev, statuses: [] }))}>
                    <input
                      type="checkbox"
                      checked={!filters.statuses || filters.statuses.length === 0}
                      readOnly
                    />
                    <span>All Statuses</span>
                  </div>
                  {filterOptions.statuses.map(status => (
                    <div
                      key={status}
                      className="multi-select-option"
                      onClick={() => handleStatusToggle(status)}
                    >
                      <input
                        type="checkbox"
                        checked={filters.statuses && filters.statuses.includes(status)}
                        readOnly
                      />
                      <span>{status}</span>
                    </div>
                  ))}
                </div>
              )}
            </div>
            <div className="filter-group multi-select-wrapper" ref={performanceStatusDropdownRef}>
              <button
                className="multi-select-button"
                onClick={() => setIsPerformanceStatusDropdownOpen(!isPerformanceStatusDropdownOpen)}
              >
                <span>
                  {filters.performance_statuses && filters.performance_statuses.length > 0
                    ? `${filters.performance_statuses.length} Performance Status${filters.performance_statuses.length > 1 ? 'es' : ''} Selected`
                    : 'All Performance Statuses'}
                </span>
                <ChevronDown size={16} />
              </button>
              {isPerformanceStatusDropdownOpen && (
                <div className="multi-select-dropdown">
                  <div className="multi-select-option" onClick={() => setFilters(prev => ({ ...prev, performance_statuses: [] }))}>
                    <input
                      type="checkbox"
                      checked={!filters.performance_statuses || filters.performance_statuses.length === 0}
                      readOnly
                    />
                    <span>All Performance Statuses</span>
                  </div>
                  {filterOptions.performanceStatuses.map(performanceStatus => (
                    <div
                      key={performanceStatus}
                      className="multi-select-option"
                      onClick={() => handlePerformanceStatusToggle(performanceStatus)}
                    >
                      <input
                        type="checkbox"
                        checked={filters.performance_statuses && filters.performance_statuses.includes(performanceStatus)}
                        readOnly
                      />
                      <span>{performanceStatus}</span>
                    </div>
                  ))}
                </div>
              )}
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
              <input
                type="number"
                placeholder="DPD Min (e.g., 0)"
                value={filters.dpd_min}
                onChange={(e) => handleFilterChange('dpd_min', e.target.value)}
                className="dpd-filter-input"
                min="0"
              />
            </div>
            <div className="filter-group">
              <input
                type="number"
                placeholder="DPD Max (e.g., 30)"
                value={filters.dpd_max}
                onChange={(e) => handleFilterChange('dpd_max', e.target.value)}
                className="dpd-filter-input"
                min="0"
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

      {/* Info message when behaviour-based filters are active */}
      {(filters.loan_type || filters.rot_type || filters.delay_type) && (
        <div className="client-filter-info">
          ℹ️ Showing {loans.length} loans on this page.
          Total matching records (all pages): {pagination.total}
        </div>
      )}

      <div className="all-loans-table-container">
        {loading ? (
          <div className="loading">Loading...</div>
        ) : (
          <table className="all-loans-table">
            <thead>
              <tr>
                <th>Actions</th>
                <th onClick={() => handleSort('loan_id')}>Loan ID</th>
                <th onClick={() => handleSort('customer_name')}>Customer Name</th>
                <th onClick={() => handleSort('customer_phone')}>Customer Phone</th>
                <th onClick={() => handleSort('officer_name')}>Officer Name</th>
                <th onClick={() => handleSort('region')}>Region</th>
                <th onClick={() => handleSort('branch')}>Branch</th>
                <th onClick={() => handleSort('vertical_lead_name')}>Vertical Lead Name</th>
                <th onClick={() => handleSort('vertical_lead_email')}>Vertical Lead Email</th>
                <th onClick={() => handleSort('channel')}>Channel</th>
                <th onClick={() => handleSort('loan_type')}>Loan Type</th>
                <th onClick={() => handleSort('verification_status')}>Verification Status</th>
                <th onClick={() => handleSort('loan_amount')}>Loan Amount</th>
                <th onClick={() => handleSort('current_dpd')}>Current DPD</th>
                <th onClick={() => handleSort('days_since_last_repayment')}>Days Since Last Repayment</th>
                <th onClick={() => handleSort('repayment_delay_rate')}>Repayment Delay Rate %</th>
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
                <th onClick={() => handleSort('wave')}>Wave</th>
                <th onClick={() => handleSort('total_outstanding')}>Total Outstanding</th>
                <th onClick={() => handleSort('actual_outstanding')}>Actual Outstanding</th>
                <th onClick={() => handleSort('total_repayments')}>Total Repayments</th>
                <th onClick={() => handleSort('status')}>Status</th>
                <th onClick={() => handleSort('performance_status')}>Performance Status</th>
                <th>FIMR Tagged</th>
              </tr>
            </thead>
            <tbody>
              {loans.map((loan) => (
                <tr key={loan.loan_id}>
                  <td className="action-cell">
                    <div style={{ display: 'flex', gap: '8px', alignItems: 'center' }}>
                      <button
                        className="view-repayments-btn"
                        onClick={() => handleViewRepayments(loan)}
                        title="View Repayment History"
                      >
                        <Eye size={16} />
                        Repayments
                      </button>
                      <button
                        className={`refresh-repayments-btn ${refreshingLoans.has(loan.loan_id) ? 'refreshing' : ''}`}
                        onClick={() => handleRefreshRepayments(loan)}
                        disabled={refreshingLoans.has(loan.loan_id)}
                        title="Refresh repayments from source database"
                      >
                        <RefreshCw size={16} className={refreshingLoans.has(loan.loan_id) ? 'spinning' : ''} />
                      </button>
                    </div>
                  </td>
                  <td className="loan-id">{loan.loan_id}</td>
                  <td>{loan.customer_name}</td>
                  <td>{loan.customer_phone || 'N/A'}</td>
                  <td>{loan.officer_name}</td>
                  <td>{loan.region}</td>
                  <td>{loan.branch}</td>
                  <td>{loan.vertical_lead_name || 'N/A'}</td>
                  <td>{loan.vertical_lead_email || 'N/A'}</td>
                  <td>{loan.channel}</td>
                  <td>{loan.loan_type || 'N/A'}</td>
                  <td>{loan.verification_status || 'N/A'}</td>
                  <td className="amount">{formatCurrency(loan.loan_amount)}</td>
                  <td className="dpd">{loan.current_dpd}</td>
                  <td className="days-since">{loan.days_since_last_repayment != null ? loan.days_since_last_repayment : 'N/A'}</td>
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
                  <td className="wave">
                    <span className={`wave-badge wave-${loan.wave?.replace(' ', '-').toLowerCase()}`}>
                      {loan.wave || 'N/A'}
                    </span>
                  </td>
                  <td className="amount">{formatCurrency(loan.total_outstanding)}</td>
                  <td className="amount">{formatCurrency(loan.actual_outstanding || 0)}</td>
                  <td className="amount">{formatCurrency(loan.total_repayments || 0)}</td>
                  <td>
                    <span className={`status-badge status-${loan.status.toLowerCase()}`}>
                      {loan.status}
                    </span>
                  </td>
                  <td>
                    {loan.performance_status ? (
                      <span className={`status-badge performance-status-${loan.performance_status.toLowerCase().replace('_', '-')}`}>
                        {loan.performance_status}
                      </span>
                    ) : (
                      <span className="status-badge status-unknown">N/A</span>
                    )}
                  </td>
                  <td className="fimr-tagged">
                    {loan.fimr_tagged ? (
                      <span className="badge-yes">Yes</span>
                    ) : (
                      <span className="badge-no">No</span>
                    )}
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

