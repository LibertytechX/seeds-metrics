import React, { useMemo } from 'react';
import { Download, RefreshCw, LogOut, User } from 'lucide-react';
import './Header.css';

export const Header = ({ filters, onFilterChange, onExport, lastRefresh, branches = [], onLogout }) => {
  const handleDateChange = (e) => {
    onFilterChange({ ...filters, dateRange: e.target.value });
  };

  const handleBranchChange = (e) => {
    onFilterChange({ ...filters, branch: e.target.value });
  };

  const handleWaveChange = (e) => {
    onFilterChange({ ...filters, wave: e.target.value });
  };

  const handleToggle = (toggleName) => {
    onFilterChange({
      ...filters,
      [toggleName]: !filters[toggleName],
    });
  };

  const formatTime = (date) => {
    return new Date(date).toLocaleTimeString();
  };

  // Extract unique branch names from branches data
  const branchOptions = useMemo(() => {
    if (!branches || branches.length === 0) {
      return [];
    }
    // Extract unique branch names and sort them
    const uniqueBranches = [...new Set(branches.map(b => b.branch))].filter(Boolean).sort();
    return uniqueBranches;
  }, [branches]);

  const username = localStorage.getItem('username') || 'Admin';

  const handleLogout = () => {
    if (window.confirm('Are you sure you want to logout?')) {
      if (onLogout) {
        onLogout();
      }
    }
  };

  return (
    <header className="header">
      <div className="header-top">
        <h1>Loan Officer Metrics Dashboard</h1>
        <div className="header-top-right">
          <div className="user-info">
            <User size={16} />
            <span>{username}</span>
          </div>
          <div className="refresh-info">
            Last refresh: {formatTime(lastRefresh)}
            <RefreshCw size={16} />
          </div>
          <button onClick={handleLogout} className="btn-logout" title="Logout">
            <LogOut size={16} />
            <span>Logout</span>
          </button>
        </div>
      </div>

      <div className="header-controls">
        <div className="control-group">
          <select value={filters.dateRange} onChange={handleDateChange}>
            <option value="week">This Week</option>
            <option value="month">This Month</option>
            <option value="quarter">This Quarter</option>
          </select>
        </div>

        <div className="control-group">
          <select value={filters.branch} onChange={handleBranchChange}>
            <option value="">All Branches</option>
            {branchOptions.map(branch => (
              <option key={branch} value={branch}>{branch}</option>
            ))}
          </select>
        </div>

        <div className="control-group">
          <select value={filters.wave} onChange={handleWaveChange}>
            <option value="">All Waves</option>
            <option value="Wave 1">Wave 1</option>
            <option value="Wave 2">Wave 2</option>
          </select>
        </div>

        <div className="toggles">
          <label className="toggle-label">
            <input
              type="checkbox"
              checked={filters.includeWatch}
              onChange={() => handleToggle('includeWatch')}
            />
            Include Watch
          </label>
          <label className="toggle-label">
            <input
              type="checkbox"
              checked={filters.dqiCpToggle}
              onChange={() => handleToggle('dqiCpToggle')}
            />
            DQI×CP
          </label>
          <label className="toggle-label">
            <input
              type="checkbox"
              checked={filters.showRedOnly}
              onChange={() => handleToggle('showRedOnly')}
            />
            Show Red Only 🚨
          </label>
        </div>

        <div className="export-buttons">
          <button onClick={() => onExport('csv')} className="btn-export">
            <Download size={16} /> CSV
          </button>
          <button onClick={() => onExport('pdf')} className="btn-export">
            <Download size={16} /> PDF
          </button>
        </div>
      </div>
    </header>
  );
};

