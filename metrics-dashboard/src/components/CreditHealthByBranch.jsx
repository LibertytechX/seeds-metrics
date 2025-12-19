import React, { useState, useMemo, useEffect } from 'react';
import { Download, Filter } from 'lucide-react';
import './CreditHealthByBranch.css';
import apiService from '../services/api';

// Temporary dummy data for Vertical Lead summary view
const VERTICAL_LEAD_SUMMARY = [
  {
    name: 'Adejare Adekemi',
    branches: 6,
    activeLOs: 25,
    loans: 147,
    outstanding: '\u20a647,858,405',
    dailyTarget: '\u20a6904,822',
    avgDPD: 4.2,
    maxDPD: 26,
    dpd0: 51,
    dpd1_6: 58,
    dpd7_14: 33,
    dpd14_21: 3,
    dpd21_plus: 2,
    quiet: 8,
    quietValue: '\u20a63,500,556',
    riskScore: 25,
    status: '\ud83d\udfe2 OK',
  },
  {
    name: 'Akeem Adebayo',
    branches: 7,
    activeLOs: 19,
    loans: 156,
    outstanding: '\u20a630,010,498',
    dailyTarget: '\u20a6703,094',
    avgDPD: 7.3,
    maxDPD: 31,
    dpd0: 35,
    dpd1_6: 54,
    dpd7_14: 40,
    dpd14_21: 15,
    dpd21_plus: 12,
    quiet: 17,
    quietValue: '\u20a64,376,780',
    riskScore: 85,
    status: '\ud83d\udd34 CRITICAL',
  },
  {
    name: 'Basirat Okanlawon',
    branches: 5,
    activeLOs: 20,
    loans: 141,
    outstanding: '\u20a632,192,229',
    dailyTarget: '\u20a6734,506',
    avgDPD: 5.0,
    maxDPD: 33,
    dpd0: 49,
    dpd1_6: 53,
    dpd7_14: 28,
    dpd14_21: 8,
    dpd21_plus: 3,
    quiet: 4,
    quietValue: '\u20a61,163,340',
    riskScore: 25,
    status: '\ud83d\udfe2 OK',
  },
  {
    name: 'ELIZABETH NWAENIE',
    branches: 7,
    activeLOs: 33,
    loans: 180,
    outstanding: '\u20a6110,256,488',
    dailyTarget: '\u20a61,792,524',
    avgDPD: 6.3,
    maxDPD: 30,
    dpd0: 60,
    dpd1_6: 45,
    dpd7_14: 51,
    dpd14_21: 17,
    dpd21_plus: 7,
    quiet: 24,
    quietValue: '\u20a619,984,317',
    riskScore: 86,
    status: '\ud83d\udd34 CRITICAL',
  },
  {
    name: 'Nwaneti Charles',
    branches: 5,
    activeLOs: 15,
    loans: 99,
    outstanding: '\u20a627,611,900',
    dailyTarget: '\u20a6541,026',
    avgDPD: 7.5,
    maxDPD: 33,
    dpd0: 27,
    dpd1_6: 34,
    dpd7_14: 15,
    dpd14_21: 13,
    dpd21_plus: 10,
    quiet: 16,
    quietValue: '\u20a65,955,468',
    riskScore: 75,
    status: '\ud83d\udd34 CRITICAL',
  },
  {
    name: 'Oluseye Awoyemi',
    branches: 5,
    activeLOs: 11,
    loans: 47,
    outstanding: '\u20a612,504,415',
    dailyTarget: '\u20a6238,597',
    avgDPD: 5.9,
    maxDPD: 36,
    dpd0: 11,
    dpd1_6: 16,
    dpd7_14: 16,
    dpd14_21: 2,
    dpd21_plus: 2,
    quiet: 4,
    quietValue: '\u20a6851,288',
    riskScore: 16,
    status: '\ud83d\udfe2 OK',
  },
  {
    name: 'Oluwafemi Kehinde',
    branches: 1,
    activeLOs: 6,
    loans: 24,
    outstanding: '\u20a69,330,913',
    dailyTarget: '\u20a6177,158',
    avgDPD: 7.2,
    maxDPD: 21,
    dpd0: 6,
    dpd1_6: 10,
    dpd7_14: 2,
    dpd14_21: 6,
    dpd21_plus: 0,
    quiet: 7,
    quietValue: '\u20a64,072,300',
    riskScore: 20,
    status: '\ud83d\udfe2 OK',
  },
  {
    name: 'Oyebola Oyediran',
    branches: 2,
    activeLOs: 5,
    loans: 51,
    outstanding: '\u20a69,296,684',
    dailyTarget: '\u20a6203,654',
    avgDPD: 4.6,
    maxDPD: 22,
    dpd0: 21,
    dpd1_6: 12,
    dpd7_14: 15,
    dpd14_21: 2,
    dpd21_plus: 1,
    quiet: 2,
    quietValue: '\u20a6683,000',
    riskScore: 9,
    status: '\ud83d\udfe2 OK',
  },
  {
    name: 'Precious Emere',
    branches: 1,
    activeLOs: 2,
    loans: 5,
    outstanding: '\u20a63,329,910',
    dailyTarget: '\u20a657,623',
    avgDPD: 12.0,
    maxDPD: 29,
    dpd0: 1,
    dpd1_6: 0,
    dpd7_14: 2,
    dpd14_21: 1,
    dpd21_plus: 1,
    quiet: 1,
    quietValue: '\u20a61,402,400',
    riskScore: 6,
    status: '\ud83d\udfe2 OK',
  },
  {
    name: 'TAIWO LAWAL',
    branches: 5,
    activeLOs: 16,
    loans: 76,
    outstanding: '\u20a620,959,784',
    dailyTarget: '\u20a6448,084',
    avgDPD: 5.6,
    maxDPD: 27,
    dpd0: 25,
    dpd1_6: 25,
    dpd7_14: 16,
    dpd14_21: 9,
    dpd21_plus: 1,
    quiet: 8,
    quietValue: '\u20a62,725,700',
    riskScore: 28,
    status: '\ud83d\dfE1 WARNING',
  },
  {
    name: 'bekee Cynthia',
    branches: 4,
    activeLOs: 10,
    loans: 51,
    outstanding: '\u20a610,126,941',
    dailyTarget: '\u20a6250,031',
    avgDPD: 8.8,
    maxDPD: 27,
    dpd0: 7,
    dpd1_6: 9,
    dpd7_14: 28,
    dpd14_21: 2,
    dpd21_plus: 5,
    quiet: 4,
    quietValue: '\u20a61,396,400',
    riskScore: 25,
    status: '\ud83d\udfe2 OK',
  },
  {
    name: 'TOTAL',
    branches: 48,
    activeLOs: 162,
    loans: 977,
    outstanding: '\u20a6313,478,167',
    dailyTarget: '\u20a66,051,118',
    avgDPD: 6.763636364,
    maxDPD: 36,
    dpd0: 293,
    dpd1_6: 316,
    dpd7_14: 246,
    dpd14_21: 78,
    dpd21_plus: 44,
    quiet: 95,
    quietValue: '\u20a646,111,549',
    riskScore: 400,
    status: '',
    isTotal: true,
  },
];

// Dummy metrics template for each vertical lead row (other than TOTAL).
const VERTICAL_LEAD_DUMMY_METRICS = {
  branches: 4,
  activeLOs: 10,
  loans: 50,
  outstanding: '₦10,126,941',
  dailyTarget: '₦250,031',
  avgDPD: 6.8,
  maxDPD: 27,
  dpd0: 30,
  dpd1_6: 20,
  dpd7_14: 15,
  dpd14_21: 5,
  dpd21_plus: 2,
  quiet: 4,
  quietValue: '₦1,396,400',
  riskScore: 25,
  status: '🟢 OK',
};

// Dummy TOTAL row with aggregated placeholder values.
const VERTICAL_LEAD_TOTAL_ROW = {
  name: 'TOTAL',
  branches: 48,
  activeLOs: 162,
  loans: 977,
  outstanding: '₦313,478,167',
  dailyTarget: '₦6,051,118',
  avgDPD: 6.763636364,
  maxDPD: 36,
  dpd0: 293,
  dpd1_6: 316,
  dpd7_14: 246,
  dpd14_21: 78,
  dpd21_plus: 44,
  quiet: 95,
  quietValue: '₦46,111,549',
  riskScore: 400,
  status: '',
  isTotal: true,
};

const CreditHealthByBranch = ({ branches: initialBranches, onFilterChange }) => {
  const [sortConfig, setSortConfig] = useState({ key: 'branch', direction: 'asc' });
  const [activeView, setActiveView] = useState('branch'); // 'branch' | 'verticalLead'
  const [filters, setFilters] = useState({
    branch: '',
    region: '',
    channel: '',
    user_type: '',
    wave: '',
  });
	  const [showFilters, setShowFilters] = useState(false);
		  const [branches, setBranches] = useState(initialBranches || []);
		  const [loading, setLoading] = useState(false);
		  const [verticalLeadMetrics, setVerticalLeadMetrics] = useState([]);
		  const [verticalLeadLoading, setVerticalLeadLoading] = useState(false);
		  const [filterOptions, setFilterOptions] = useState({
	    branches: [],
	    regions: [],
	    channels: [],
	    userTypes: [],
	    waves: [],
	  });

  // Fetch filter options from API
  useEffect(() => {
    const fetchFilterOptions = async () => {
      try {
        const API_BASE_URL = import.meta.env.VITE_API_URL || 'http://localhost:8081/api/v1';

        const [branchesRes, regionsRes, channelsRes, userTypesRes, wavesRes] = await Promise.all([
          fetch(`${API_BASE_URL}/filters/branches`),
          fetch(`${API_BASE_URL}/filters/regions`),
          fetch(`${API_BASE_URL}/filters/channels`),
          fetch(`${API_BASE_URL}/filters/user-types`),
          fetch(`${API_BASE_URL}/filters/waves`),
        ]);

        const [branchesData, regionsData, channelsData, userTypesData, wavesData] = await Promise.all([
          branchesRes.json(),
          regionsRes.json(),
          channelsRes.json(),
          userTypesRes.json(),
          wavesRes.json(),
        ]);

        setFilterOptions({
          branches: branchesData.data?.branches || [],
          regions: regionsData.data?.regions || [],
          channels: channelsData.data?.channels || [],
          userTypes: userTypesData.data?.user_types || [],
          waves: wavesData.data?.waves || [],
        });
      } catch (error) {
        console.error('Error fetching filter options:', error);
      }
    };

    fetchFilterOptions();
  }, []);

  // Fetch branches with applied filters
  useEffect(() => {
    const fetchBranches = async () => {
      setLoading(true);
      try {
        const params = {};
        if (filters.branch) params.branch = filters.branch;
        if (filters.region) params.region = filters.region;
        if (filters.channel) params.channel = filters.channel;
        if (filters.user_type) params.user_type = filters.user_type;
        if (filters.wave) params.wave = filters.wave;

        const branchesData = await apiService.fetchBranches(params);
        setBranches(branchesData.map(b => apiService.transformBranchData(b)));
      } catch (error) {
        console.error('Error fetching branches:', error);
        setBranches(initialBranches || []);
      } finally {
        setLoading(false);
      }
    };

    fetchBranches();
	  }, [filters, initialBranches]);

		  // Fetch vertical lead metrics for the Vertical Lead view.
		  useEffect(() => {
		    if (activeView !== 'verticalLead') {
		      return;
		    }

		    let isMounted = true;

		    const fetchVerticalLeadMetrics = async () => {
		      setVerticalLeadLoading(true);
		      try {
		        const params = {};
		        if (filters.branch) params.branch = filters.branch;
		        if (filters.region) params.region = filters.region;
		        if (filters.channel) params.channel = filters.channel;
		        if (filters.user_type) params.user_type = filters.user_type;
		        if (filters.wave) params.wave = filters.wave;

		        const metrics = await apiService.fetchVerticalLeadMetrics(params);
		        if (!isMounted) return;
		        setVerticalLeadMetrics(metrics || []);
		      } catch (error) {
		        console.error('Error fetching vertical lead metrics:', error);
		        if (!isMounted) return;
		        setVerticalLeadMetrics([]);
		      } finally {
		        if (!isMounted) return;
		        setVerticalLeadLoading(false);
		      }
		    };

		    fetchVerticalLeadMetrics();

		    return () => {
		      isMounted = false;
		    };
		  }, [activeView, filters]);

  // Get unique values for filter dropdowns (fallback if API fails)
  const fallbackFilterOptions = useMemo(() => {
    return {
      regions: [...new Set(branches.map(b => b.region))].sort(),
    };
  }, [branches]);

	  // Apply sorting
	  const sortedBranches = useMemo(() => {
    const sorted = [...branches];
    sorted.sort((a, b) => {
      let aVal = a[sortConfig.key];
      let bVal = b[sortConfig.key];

      // Handle numeric values
      if (typeof aVal === 'number' && typeof bVal === 'number') {
        return sortConfig.direction === 'asc' ? aVal - bVal : bVal - aVal;
      }

      // Handle string values
      aVal = String(aVal).toLowerCase();
      bVal = String(bVal).toLowerCase();
      if (aVal < bVal) return sortConfig.direction === 'asc' ? -1 : 1;
      if (aVal > bVal) return sortConfig.direction === 'asc' ? 1 : -1;
      return 0;
    });
	    return sorted;
	  }, [branches, sortConfig]);

		  // Build vertical lead rows using live metrics from the API plus dummy business fields.
		  const verticalLeadRows = useMemo(() => {
		    if (!verticalLeadMetrics || verticalLeadMetrics.length === 0) {
		      return [];
		    }

		    let totalBranches = 0;
		    let totalActiveLOs = 0;
		    let totalLoans = 0;
		    let totalOutstanding = 0;
		    let totalDPD0 = 0;
		    let totalDPD1to6 = 0;
		    let totalDPD7to14 = 0;
		    let totalDPD14to21 = 0;
		    let totalDPD21Plus = 0;
		    let totalQuiet = 0;
		    let totalQuietValue = 0;
		    let weightedDpdSum = 0;

		    const rows = (verticalLeadMetrics || []).map((row) => {
		      const loans = row.loans || 0;
		      const avgDpd = row.avg_dpd || 0;

		      totalBranches += row.branches || 0;
		      totalActiveLOs += row.active_los || 0;
		      totalLoans += loans;
		      totalOutstanding += row.outstanding || 0;
		      totalDPD0 += row.dpd0 || 0;
		      totalDPD1to6 += row.dpd1_6 || 0;
		      totalDPD7to14 += row.dpd7_14 || 0;
		      totalDPD14to21 += row.dpd14_21 || 0;
		      totalDPD21Plus += row.dpd21_plus || 0;
		      totalQuiet += row.quiet || 0;
		      totalQuietValue += row.quiet_value || 0;
		      weightedDpdSum += avgDpd * loans;

		      return {
		        name: row.vertical_lead_name || 'Unassigned Vertical Lead',
		        email: row.vertical_lead_email || '',
		        branches: row.branches || 0,
		        activeLOs: row.active_los || 0,
		        loans,
		        outstanding: row.outstanding || 0,
		        avgDPD: avgDpd,
		        maxDPD: row.max_dpd || 0,
		        dpd0: row.dpd0 || 0,
		        dpd1_6: row.dpd1_6 || 0,
		        dpd7_14: row.dpd7_14 || 0,
		        dpd14_21: row.dpd14_21 || 0,
		        dpd21_plus: row.dpd21_plus || 0,
		        quiet: row.quiet || 0,
		        quietValue: row.quiet_value || 0,
		        // Business fields still dummy for now
		        dailyTarget: VERTICAL_LEAD_DUMMY_METRICS.dailyTarget,
		        riskScore: VERTICAL_LEAD_DUMMY_METRICS.riskScore,
		        status: VERTICAL_LEAD_DUMMY_METRICS.status,
		      };
		    });

		    if (rows.length === 0) {
		      return rows;
		    }

		    const totalAvgDPD = totalLoans > 0 ? weightedDpdSum / totalLoans : 0;

		    const totalRow = {
		      name: 'TOTAL',
		      email: '',
		      branches: totalBranches,
		      activeLOs: totalActiveLOs,
		      loans: totalLoans,
		      outstanding: totalOutstanding,
		      avgDPD: totalAvgDPD,
		      maxDPD: rows.reduce((max, r) => (r.maxDPD > max ? r.maxDPD : max), 0),
		      dpd0: totalDPD0,
		      dpd1_6: totalDPD1to6,
		      dpd7_14: totalDPD7to14,
		      dpd14_21: totalDPD14to21,
		      dpd21_plus: totalDPD21Plus,
		      quiet: totalQuiet,
		      quietValue: totalQuietValue,
		      // Keep business fields dummy for now on the TOTAL row as well
		      dailyTarget: VERTICAL_LEAD_TOTAL_ROW.dailyTarget,
		      riskScore: VERTICAL_LEAD_TOTAL_ROW.riskScore,
		      status: VERTICAL_LEAD_TOTAL_ROW.status,
		      isTotal: true,
		    };

		    return [...rows, totalRow];
		  }, [verticalLeadMetrics]);

  const handleSort = (key) => {
    setSortConfig(prev => ({
      key,
      direction: prev.key === key && prev.direction === 'asc' ? 'desc' : 'asc',
    }));
  };

  const handleFilterChange = (filterKey, value) => {
    setFilters(prev => ({ ...prev, [filterKey]: value }));
    if (onFilterChange) {
      onFilterChange({ ...filters, [filterKey]: value });
    }
  };

  const clearFilters = () => {
    setFilters({
      branch: '',
      region: '',
      channel: '',
      user_type: '',
      wave: '',
    });
    if (onFilterChange) {
      onFilterChange({
        branch: '',
        region: '',
        channel: '',
        user_type: '',
        wave: '',
      });
    }
  };

  const handleExport = () => {
    // Create CSV content
    const headers = [
      'Branch', 'Region', 'Portfolio Total', 'Overdue >15D', 'PAR15 Ratio',
      'AYR', 'DQI', 'FIMR', 'Avg Repayment Delay Rate %', 'Active Loans', 'Total Officers'
    ];

    const rows = sortedBranches.map(branch => [
      branch.branch,
      branch.region,
      branch.portfolioTotal,
      branch.overdue15d,
      branch.par15Ratio,
      branch.ayr,
      branch.dqi,
      branch.fimr,
      branch.avgRepaymentDelayRate,
      branch.activeLoans,
      branch.totalOfficers,
    ]);

    const csvContent = [
      headers.join(','),
      ...rows.map(row => row.join(','))
    ].join('\n');

    // Download CSV
    const blob = new Blob([csvContent], { type: 'text/csv' });
    const url = window.URL.createObjectURL(blob);
    const link = document.createElement('a');
    link.href = url;
    link.download = `Credit_Health_By_Branch_${new Date().toISOString().split('T')[0]}.csv`;
    link.style.visibility = 'hidden';
    document.body.appendChild(link);
    link.click();
    document.body.removeChild(link);
  };

  const formatCurrency = (value) => {
    return new Intl.NumberFormat('en-NG', {
      style: 'currency',
      currency: 'NGN',
      minimumFractionDigits: 0,
    }).format(value);
  };

  const formatPercent = (value) => {
    return `${(value * 100).toFixed(2)}%`;
  };

  const formatDecimal = (value, decimals = 2) => {
    return value.toFixed(decimals);
  };

  const activeFilterCount = Object.values(filters).filter(v => v !== '').length;

  return (
    <div className="credit-health-branch">
      <div className="credit-health-tabs">
        <button
          type="button"
          className={`credit-health-tab ${activeView === 'branch' ? 'active' : ''}`}
          onClick={() => setActiveView('branch')}
        >
          By Branch
        </button>
        <button
          type="button"
          className={`credit-health-tab ${activeView === 'verticalLead' ? 'active' : ''}`}
          onClick={() => setActiveView('verticalLead')}
        >
          By Vertical Lead
        </button>
      </div>

      {activeView === 'branch' && (
        <>
      <div className="branch-header">
        <div className="branch-title">
          <h2>Credit Health Overview by Branch</h2>
          <span className="branch-count">{sortedBranches.length} Branches</span>
        </div>
        <div className="branch-actions">
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
          <button className="export-button" onClick={handleExport}>
            <Download size={16} />
            Export CSV
          </button>
        </div>
      </div>

      {showFilters && (
        <div className="filter-panel">
          <div className="filter-row">
            <div className="filter-group">
              <label>Branch</label>
              <select
                value={filters.branch}
                onChange={(e) => handleFilterChange('branch', e.target.value)}
              >
                <option value="">All Branches</option>
                {(filterOptions.branches || fallbackFilterOptions.branches || []).map(branch => (
                  <option key={branch} value={branch}>{branch}</option>
                ))}
              </select>
            </div>
            <div className="filter-group">
              <label>Region</label>
              <select
                value={filters.region}
                onChange={(e) => handleFilterChange('region', e.target.value)}
              >
                <option value="">All Regions</option>
                {(filterOptions.regions || fallbackFilterOptions.regions || []).map(region => (
                  <option key={region} value={region}>{region}</option>
                ))}
              </select>
            </div>
            <div className="filter-group">
              <label>Channel</label>
              <select
                value={filters.channel}
                onChange={(e) => handleFilterChange('channel', e.target.value)}
              >
                <option value="">All Channels</option>
                {(filterOptions.channels || []).map(channel => (
                  <option key={channel} value={channel}>{channel}</option>
                ))}
              </select>
            </div>
            <div className="filter-group">
              <label>User Type</label>
              <select
                value={filters.user_type}
                onChange={(e) => handleFilterChange('user_type', e.target.value)}
              >
                <option value="">All User Types</option>
                {(filterOptions.userTypes || []).map(userType => (
                  <option key={userType} value={userType}>{userType}</option>
                ))}
              </select>
            </div>
            <div className="filter-group">
              <label>Wave</label>
              <select
                value={filters.wave}
                onChange={(e) => handleFilterChange('wave', e.target.value)}
              >
                <option value="">All Waves</option>
                {(filterOptions.waves || []).map(wave => (
                  <option key={wave} value={wave}>{wave}</option>
                ))}
              </select>
            </div>
            <div className="filter-group">
              <button className="clear-filters" onClick={clearFilters}>Clear All</button>
            </div>
          </div>
        </div>
      )}

      <div className="branch-table-container">
        {loading && (
          <div className="loading-indicator">
            <div className="spinner"></div>
            <p>Loading branches...</p>
          </div>
        )}
        {!loading && (
          <table className="branch-table">
            <thead>
              <tr>
                <th onClick={() => handleSort('branch')}>Branch</th>
                <th onClick={() => handleSort('region')}>Region</th>
                <th onClick={() => handleSort('portfolioTotal')}>Portfolio Total</th>
                <th onClick={() => handleSort('overdue15d')}>Overdue &gt;15D</th>
                <th onClick={() => handleSort('par15Ratio')}>PAR15 Ratio</th>
                <th onClick={() => handleSort('ayr')}>AYR</th>
                <th onClick={() => handleSort('dqi')}>DQI</th>
                <th onClick={() => handleSort('fimr')}>FIMR</th>
                <th onClick={() => handleSort('avgRepaymentDelayRate')}>Avg Repayment Delay Rate</th>
                <th onClick={() => handleSort('activeLoans')}>Active Loans</th>
                <th onClick={() => handleSort('totalOfficers')}>Total Officers</th>
              </tr>
            </thead>
            <tbody>
              {sortedBranches.length === 0 ? (
                <tr>
                  <td colSpan="11" className="no-data">No branches found matching the selected filters</td>
                </tr>
              ) : (
                sortedBranches.map((branch, index) => (
                  <tr key={branch.branch}>
                    <td className="branch-name">{branch.branch}</td>
                    <td>{branch.region}</td>
                    <td className="amount">{formatCurrency(branch.portfolioTotal)}</td>
                    <td className="amount">{formatCurrency(branch.overdue15d)}</td>
                    <td className="metric">{formatPercent(branch.par15Ratio)}</td>
                    <td className="metric">{formatPercent(branch.ayr)}</td>
                    <td className="metric">{formatDecimal(branch.dqi, 0)}</td>
                    <td className="metric">{formatPercent(branch.fimr)}</td>
                    <td className="metric">{formatPercent(branch.avgRepaymentDelayRate / 100)}</td>
                    <td className="count">{branch.activeLoans}</td>
                    <td className="count">{branch.totalOfficers}</td>
                  </tr>
                ))
              )}
            </tbody>
          </table>
        )}
      </div>
        </>
	      )}

		      {activeView === 'verticalLead' && (
		        <div className="vertical-lead-section">
		          <div className="branch-header">
		            <div className="branch-title">
		              <h2>Credit Health Overview by Vertical Lead</h2>
		              <span className="branch-count">
		            {verticalLeadMetrics.length} Vertical Leads
		              </span>
		            </div>
		          </div>

		          <div className="branch-table-container">
		            <table className="branch-table vertical-lead-table">
		              <thead>
		                <tr>
		              <th>Vertical Lead</th>
		              <th>Email</th>
		                  <th>Branches</th>
		                  <th>Active LOs</th>
		                  <th>Loans</th>
		                  <th>Outstanding</th>
		                  <th>Daily Target</th>
		                  <th>Avg DPD</th>
		                  <th>Max DPD</th>
		                  <th>DPD 0</th>
		                  <th>DPD 1-6</th>
		                  <th>DPD 7-14</th>
		                  <th>DPD 14-21</th>
		                  <th>DPD 21+</th>
		                  <th>Quiet</th>
		                  <th>Quiet Value</th>
		                  <th>Risk Score</th>
		                  <th>Status</th>
		                </tr>
		              </thead>
		              <tbody>
		                {verticalLeadLoading ? (
		                  <tr>
		                <td colSpan="18" className="no-data">Loading vertical leads...</td>
		                  </tr>
		                ) : verticalLeadRows.length === 0 ? (
		                  <tr>
		                <td colSpan="18" className="no-data">No vertical leads found</td>
		                  </tr>
		                ) : (
		                  verticalLeadRows.map((lead) => (
		                    <tr key={lead.name} className={lead.isTotal ? 'vertical-lead-total-row' : ''}>
		                      <td className="vertical-lead-name">{lead.name}</td>
		                  <td className="vertical-lead-email">{lead.email}</td>
		                      <td className="count">{lead.branches}</td>
		                      <td className="count">{lead.activeLOs}</td>
		                      <td className="count">{lead.loans}</td>
		                      <td className="amount">{formatCurrency(lead.outstanding)}</td>
		                      <td className="amount">{lead.dailyTarget}</td>
		                      <td className="metric">{formatDecimal(lead.avgDPD, 1)}</td>
		                      <td className="metric">{lead.maxDPD}</td>
		                      <td className="count">{lead.dpd0}</td>
		                      <td className="count">{lead.dpd1_6}</td>
		                      <td className="count">{lead.dpd7_14}</td>
		                      <td className="count">{lead.dpd14_21}</td>
		                      <td className="count">{lead.dpd21_plus}</td>
		                      <td className="count">{lead.quiet}</td>
		                      <td className="amount">{formatCurrency(lead.quietValue)}</td>
		                      <td className="metric">{lead.riskScore}</td>
		                      <td className="status-cell">{lead.status}</td>
		                    </tr>
		                  ))
		                )}
		              </tbody>
		            </table>
		          </div>
		        </div>
		      )}
	    </div>
	  );
};

export default CreditHealthByBranch;

