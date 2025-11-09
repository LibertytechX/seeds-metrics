import React, { useMemo, useState, useEffect, useRef } from 'react';
import { Download, RefreshCw, LogOut, User, ChevronDown } from 'lucide-react';
import './Header.css';

export const Header = ({ filters, onFilterChange, onExport, lastRefresh, branches = [], onLogout }) => {
  const [regionOptions, setRegionOptions] = useState([]);
  const [isRegionDropdownOpen, setIsRegionDropdownOpen] = useState(false);
  const regionDropdownRef = useRef(null);

  // Fetch region options from API
  useEffect(() => {
    const fetchRegionOptions = async () => {
      try {
        const API_BASE_URL = import.meta.env.VITE_API_URL ||
          (import.meta.env.MODE === 'production' ? '/api/v1' : 'http://localhost:8081/api/v1');
        const response = await fetch(`${API_BASE_URL}/filters/regions`);
        const data = await response.json();
        if (data.status === 'success') {
          setRegionOptions(data.data.regions || []);
        }
      } catch (error) {
        console.error('Error fetching region options:', error);
      }
    };
    fetchRegionOptions();
  }, []);

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

  const handleDateChange = (e) => {
    onFilterChange({ ...filters, dateRange: e.target.value });
  };

  const handleBranchChange = (e) => {
    onFilterChange({ ...filters, branch: e.target.value });
  };

  const handleWaveChange = (e) => {
    onFilterChange({ ...filters, wave: e.target.value });
  };

  const handleRegionToggle = (region) => {
    const currentRegions = filters.regions || [];
    const newRegions = currentRegions.includes(region)
      ? currentRegions.filter(r => r !== region)
      : [...currentRegions, region];
    onFilterChange({ ...filters, regions: newRegions });
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

        <div className="control-group multi-select-wrapper" ref={regionDropdownRef}>
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
              <div className="multi-select-option" onClick={() => onFilterChange({ ...filters, regions: [] })}>
                <input
                  type="checkbox"
                  checked={!filters.regions || filters.regions.length === 0}
                  readOnly
                />
                <span>All Regions</span>
              </div>
              {regionOptions.map(region => (
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

