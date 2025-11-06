import React, { useState, useEffect, useMemo } from 'react';
import { getBand } from '../utils/metrics';
import { formatMetricTooltip } from '../utils/metricInfo';
import { MetricHeader } from './Tooltip';
import { Filter, Download } from 'lucide-react';
import './DataTables.css';
import './CreditHealthByBranch.css';

const BandBadge = ({ band }) => {
  const colors = {
    green: '#10b981',
    yellow: '#f59e0b',
    red: '#ef4444',
    amber: '#f97316',
  };
  return (
    <span
      className="band-badge"
      style={{ backgroundColor: colors[band.color] }}
    >
      {band.label}
    </span>
  );
};

const OfficerPerformanceTable = ({ officers }) => {
  const [sortBy, setSortBy] = useState('riskScore');
  const [sortDir, setSortDir] = useState('asc');

  const sorted = [...officers].sort((a, b) => {
    const aVal = a[sortBy];
    const bVal = b[sortBy];

    // Handle string sorting (for name, region, etc.)
    if (typeof aVal === 'string' && typeof bVal === 'string') {
      return sortDir === 'asc'
        ? aVal.localeCompare(bVal)
        : bVal.localeCompare(aVal);
    }

    // Handle numeric sorting
    return sortDir === 'asc' ? aVal - bVal : bVal - aVal;
  });

  const handleSort = (column) => {
    if (sortBy === column) {
      setSortDir(sortDir === 'asc' ? 'desc' : 'asc');
    } else {
      setSortBy(column);
      setSortDir('asc');
    }
  };

  return (
    <div className="table-container">
      <table className="data-table">
        <thead>
          <tr>
            <th onClick={() => handleSort('name')}>Officer</th>
            <th onClick={() => handleSort('region')}>Region</th>
            <th onClick={() => handleSort('ayr')}>
              <MetricHeader label="AYR" metricKey="ayr" info={formatMetricTooltip('ayr')} />
            </th>
            <th onClick={() => handleSort('dqi')}>
              <MetricHeader label="DQI" metricKey="dqi" info={formatMetricTooltip('dqi')} />
            </th>
            <th onClick={() => handleSort('riskScore')}>
              <MetricHeader label="Risk Score" metricKey="riskScore" info={formatMetricTooltip('riskScore')} />
            </th>
            <th onClick={() => handleSort('overdue15dVolume')}>
              <MetricHeader label="Overdue &gt;15D" metricKey="overdue15d" info={formatMetricTooltip('overdue15d')} />
            </th>
            <th onClick={() => handleSort('yield')}>
              <MetricHeader label="Yield" metricKey="yield" info={formatMetricTooltip('yield')} />
            </th>
            <th>Band</th>
            <th>Actions</th>
          </tr>
        </thead>
        <tbody>
          {sorted.map((officer) => {
            const riskBand = getBand(officer.riskScore, 'riskScore');
            return (
              <tr key={officer.id}>
                <td>{officer.name}</td>
                <td>{officer.region}</td>
                <td>{(officer.ayr * 100).toFixed(2)}%</td>
                <td>{officer.dqi}</td>
                <td>{officer.riskScore}</td>
                <td>₦{(officer.overdue15dVolume / 1000000).toFixed(1)}M</td>
                <td>₦{(officer.yield / 1000000).toFixed(1)}M</td>
                <td>
                  <BandBadge band={riskBand} />
                </td>
                <td>
                  <button className="btn-action">View</button>
                </td>
              </tr>
            );
          })}
        </tbody>
      </table>
    </div>
  );
};

const CreditHealthTable = ({ officers, onFilterChange }) => {
  const [showFilters, setShowFilters] = useState(false);
  const [filters, setFilters] = useState({
    branch: '',
    region: '',
    channel: '',
    user_type: '',
  });
  const [filterOptions, setFilterOptions] = useState({
    branches: [],
    regions: [],
    channels: [],
    userTypes: [],
  });

  // Fetch filter options from API
  useEffect(() => {
    const fetchFilterOptions = async () => {
      try {
        const API_BASE_URL = import.meta.env.VITE_API_URL || '/api/v1';

        const [branchesRes, regionsRes, channelsRes, userTypesRes] = await Promise.all([
          fetch(`${API_BASE_URL}/filters/branches`),
          fetch(`${API_BASE_URL}/filters/regions`),
          fetch(`${API_BASE_URL}/filters/channels`),
          fetch(`${API_BASE_URL}/filters/user-types`),
        ]);

        const [branchesData, regionsData, channelsData, userTypesData] = await Promise.all([
          branchesRes.json(),
          regionsRes.json(),
          channelsRes.json(),
          userTypesRes.json(),
        ]);

        setFilterOptions({
          branches: branchesData.data?.branches || [],
          regions: regionsData.data?.regions || [],
          channels: channelsData.data?.channels || [],
          userTypes: userTypesData.data?.['user-types'] || [],
          waves: [],
        });
      } catch (error) {
        console.error('Error fetching filter options:', error);
      }
    };

    fetchFilterOptions();
  }, []);

  // Notify parent component when filters change
  useEffect(() => {
    if (onFilterChange) {
      onFilterChange(filters);
    }
  }, [filters, onFilterChange]);

  const handleFilterChange = (filterKey, value) => {
    setFilters(prev => ({ ...prev, [filterKey]: value }));
  };

  const clearFilters = () => {
    setFilters({
      branch: '',
      region: '',
      channel: '',
      user_type: '',
    });
  };

  const activeFilterCount = Object.values(filters).filter(v => v !== '').length;

  return (
    <div className="credit-health-branch">
      <div className="branch-header">
        <div className="branch-title">
          <h2>Credit Health Overview</h2>
          <span className="branch-count">{officers.length} Officers</span>
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
                {filterOptions.branches.map(branch => (
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
                {filterOptions.regions.map(region => (
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
                {filterOptions.channels.map(channel => (
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
                {filterOptions.userTypes.map(userType => (
                  <option key={userType} value={userType}>{userType}</option>
                ))}
              </select>
            </div>
            <div className="filter-group">
              <button className="clear-filters" onClick={clearFilters}>Clear All</button>
            </div>
          </div>
        </div>
      )}

      <div className="table-container">
        <table className="data-table">
          <thead>
            <tr>
              <th>Officer</th>
              <th>Portfolio Total</th>
              <th>
                <MetricHeader label="Overdue &gt;15D" metricKey="overdue15d" info={formatMetricTooltip('overdue15d')} />
              </th>
              <th>
                <MetricHeader label="AYR" metricKey="ayr" info={formatMetricTooltip('ayr')} />
              </th>
              <th>
                <MetricHeader label="DQI" metricKey="dqi" info={formatMetricTooltip('dqi')} />
              </th>
              <th>
                <MetricHeader label="FIMR" metricKey="fimr" info={formatMetricTooltip('fimr')} />
              </th>
              <th>Actions</th>
            </tr>
          </thead>
          <tbody>
            {officers.map((officer) => (
              <tr key={officer.id}>
                <td>{officer.name}</td>
                <td>₦{(officer.totalPortfolio / 1000000).toFixed(1)}M</td>
                <td>₦{(officer.overdue15d / 1000000).toFixed(1)}M</td>
                <td>{officer.ayr.toFixed(2)}</td>
                <td>{officer.dqi}</td>
                <td>{(officer.fimr * 100).toFixed(1)}%</td>
                <td>
                  <button className="btn-action">Drill Down</button>
                </td>
              </tr>
            ))}
          </tbody>
        </table>
      </div>
    </div>
  );
};

const EarlyIndicatorsTable = ({ officers }) => {
  return (
    <div className="table-container">
      <table className="data-table">
        <thead>
          <tr>
            <th>Officer</th>
            <th>
              <MetricHeader label="FIMR" metricKey="fimr" info={formatMetricTooltip('fimr')} />
            </th>
            <th>
              <MetricHeader label="D0-6 Slippage" metricKey="slippage" info={formatMetricTooltip('slippage')} />
            </th>
            <th>
              <MetricHeader label="Roll" metricKey="roll" info={formatMetricTooltip('roll')} />
            </th>
            <th>
              <MetricHeader label="FRR" metricKey="frr" info={formatMetricTooltip('frr')} />
            </th>
            <th>
              <MetricHeader label="Channel Purity" metricKey="channelPurity" info={formatMetricTooltip('channelPurity')} />
            </th>
            <th>Actions</th>
          </tr>
        </thead>
        <tbody>
          {officers.map((officer) => (
            <tr key={officer.id}>
              <td>{officer.name}</td>
              <td>
                <BandBadge band={getBand(officer.fimr, 'fimr')} />
                {(officer.fimr * 100).toFixed(1)}%
              </td>
              <td>
                <BandBadge band={getBand(officer.slippage, 'slippage')} />
                {(officer.slippage * 100).toFixed(1)}%
              </td>
              <td>
                <BandBadge band={getBand(officer.roll, 'roll')} />
                {(officer.roll * 100).toFixed(1)}%
              </td>
              <td>{(officer.frr * 100).toFixed(1)}%</td>
              <td>{(officer.channelPurity * 100).toFixed(1)}%</td>
              <td>
                <button className="btn-action">View Trends</button>
              </td>
            </tr>
          ))}
        </tbody>
      </table>
    </div>
  );
};

export const DataTables = ({ officers, activeTab, onFilterChange }) => {
  const tabs = {
    creditHealth: <CreditHealthTable officers={officers} onFilterChange={onFilterChange} />,
    performance: <OfficerPerformanceTable officers={officers} />,
    earlyIndicators: <EarlyIndicatorsTable officers={officers} />,
  };

  return tabs[activeTab] || tabs.performance;
};

