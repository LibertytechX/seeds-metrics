import React, { useState, useMemo, useEffect } from 'react';
import { Download, Filter } from 'lucide-react';
import './CreditHealthByBranch.css';
import apiService from '../services/api';

const CreditHealthByBranch = ({ branches: initialBranches, onFilterChange }) => {
  const [sortConfig, setSortConfig] = useState({ key: 'branch', direction: 'asc' });
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

  // Get unique values for filter dropdowns (fallback if API fails)
  const fallbackFilterOptions = useMemo(() => {
    return {
      regions: [...new Set(branches.map(b => b.region))].sort(),
    };
  }, [branches]);

  // Apply sorting
  const sortedBranches = useMemo(() => {
    const sorted = [...filteredBranches];
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
  }, [filteredBranches, sortConfig]);

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
      'AYR', 'DQI', 'FIMR', 'Active Loans', 'Total Officers'
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
                <th onClick={() => handleSort('activeLoans')}>Active Loans</th>
                <th onClick={() => handleSort('totalOfficers')}>Total Officers</th>
              </tr>
            </thead>
            <tbody>
              {sortedBranches.length === 0 ? (
                <tr>
                  <td colSpan="10" className="no-data">No branches found matching the selected filters</td>
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
                    <td className="count">{branch.activeLoans}</td>
                    <td className="count">{branch.totalOfficers}</td>
                  </tr>
                ))
              )}
            </tbody>
          </table>
        )}
      </div>
    </div>
  );
};

export default CreditHealthByBranch;

