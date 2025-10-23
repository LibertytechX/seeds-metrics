import React, { useState, useMemo } from 'react';
import { Download, Filter } from 'lucide-react';
import './CreditHealthByBranch.css';

const CreditHealthByBranch = ({ branches }) => {
  const [sortConfig, setSortConfig] = useState({ key: 'branch', direction: 'asc' });
  const [filters, setFilters] = useState({
    region: '',
  });
  const [showFilters, setShowFilters] = useState(false);

  // Get unique values for filter dropdowns
  const filterOptions = useMemo(() => {
    return {
      regions: [...new Set(branches.map(b => b.region))].sort(),
    };
  }, [branches]);

  // Apply filters
  const filteredBranches = useMemo(() => {
    return branches.filter(branch => {
      if (filters.region && branch.region !== filters.region) return false;
      return true;
    });
  }, [branches, filters]);

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
  };

  const clearFilters = () => {
    setFilters({
      region: '',
    });
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
              <select
                value={filters.region}
                onChange={(e) => handleFilterChange('region', e.target.value)}
              >
                <option value="">All Regions</option>
                {filterOptions.regions.map(region => (
                  <option key={region} value={region}>{region}</option>
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
            {sortedBranches.map((branch, index) => (
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
            ))}
          </tbody>
        </table>
      </div>
    </div>
  );
};

export default CreditHealthByBranch;

