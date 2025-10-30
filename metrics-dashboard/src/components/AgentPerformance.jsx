import React, { useState, useMemo } from 'react';
import { Download, Filter, ChevronDown } from 'lucide-react';
import { mockTeamMembers } from '../utils/mockData';
import TopRiskLoansModal from './TopRiskLoansModal';
import './AgentPerformance.css';

const AgentPerformance = ({ agents }) => {
  const [sortConfig, setSortConfig] = useState({ key: 'riskScore', direction: 'asc' });
  const [filters, setFilters] = useState({
    region: '',
    branch: '',
    riskBand: '',
    startDate: '',
    endDate: '',
  });
  const [showFilters, setShowFilters] = useState(false);
  const [activeActionMenu, setActiveActionMenu] = useState(null);
  const [agentData, setAgentData] = useState(agents);
  const [topRiskModalOpen, setTopRiskModalOpen] = useState(false);
  const [selectedOfficer, setSelectedOfficer] = useState(null);

  // Get unique values for filter dropdowns
  const filterOptions = useMemo(() => {
    return {
      regions: [...new Set(agents.map(a => a.region))].sort(),
      branches: [...new Set(agents.map(a => a.branch))].sort(),
      riskBands: [...new Set(agents.map(a => a.riskBand))].sort(),
    };
  }, [agents]);

  // Apply filters
  const filteredAgents = useMemo(() => {
    return agentData.filter(agent => {
      if (filters.region && agent.region !== filters.region) return false;
      if (filters.branch && agent.branch !== filters.branch) return false;
      if (filters.riskBand && agent.riskBand !== filters.riskBand) return false;

      // Date range filter (filter by lastAuditDate)
      if (filters.startDate && agent.lastAuditDate && agent.lastAuditDate < filters.startDate) return false;
      if (filters.endDate && agent.lastAuditDate && agent.lastAuditDate > filters.endDate) return false;

      return true;
    });
  }, [agentData, filters]);

  // Apply sorting
  const sortedAgents = useMemo(() => {
    const sorted = [...filteredAgents];
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
  }, [filteredAgents, sortConfig]);

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
      branch: '',
      riskBand: '',
      startDate: '',
      endDate: '',
    });
  };

  const handleAssigneeChange = (officerName, newAssignee) => {
    console.log(`Changing assignee for ${officerName} to ${newAssignee}`);
    // Update local state
    setAgentData(prev => prev.map(agent =>
      agent.officerName === officerName
        ? { ...agent, assignee: newAssignee }
        : agent
    ));
    // In real implementation, this would update backend
    alert(`Assignee for ${officerName} changed to: ${newAssignee}`);
  };

  const handleAuditStatusChange = (officerName, newStatus) => {
    console.log(`Changing audit status for ${officerName} to ${newStatus}`);
    const currentDate = new Date().toISOString().split('T')[0];
    // Update local state - update both status and last audit date
    setAgentData(prev => prev.map(agent =>
      agent.officerName === officerName
        ? {
            ...agent,
            auditStatus: newStatus,
            lastAuditDate: currentDate // Update date when status changes
          }
        : agent
    ));
    // In real implementation, this would update backend
    alert(`Audit status for ${officerName} changed to: ${newStatus}\nLast Audit Date updated to: ${currentDate}`);
  };

  const handleAction = (agent, actionType) => {
    switch (actionType) {
      case 'audit20':
        // Open the Top Risk Loans modal
        setSelectedOfficer({
          name: agent.officerName,
          id: agent.officerId
        });
        setTopRiskModalOpen(true);
        break;
      case 'freeze':
        if (confirm(`Freeze disbursement for ${agent.officerName}?`)) {
          alert(`Disbursement frozen for ${agent.officerName}`);
        }
        break;
      case 'viewPortfolio':
        alert(`View Entire Portfolio for ${agent.officerName}\n\nThis will open a detailed view.`);
        break;
      case 'exportPortfolio':
        // Export officer's portfolio as CSV
        exportOfficerPortfolio(agent);
        break;
    }
    setActiveActionMenu(null);
  };

  const exportOfficerPortfolio = (agent) => {
    const csvContent = `Officer Portfolio Export\nOfficer: ${agent.officerName}\nRegion: ${agent.region}\nBranch: ${agent.branch}\n\nThis would contain all loans for this officer.`;
    const blob = new Blob([csvContent], { type: 'text/csv' });
    const url = window.URL.createObjectURL(blob);
    const a = document.createElement('a');
    a.href = url;
    a.download = `${agent.officerName.replace(/\s+/g, '_')}_Portfolio_${new Date().toISOString().split('T')[0]}.csv`;
    a.click();
    window.URL.revokeObjectURL(url);
  };

  const handleExport = () => {
    // Create CSV content
    const headers = [
      'Officer Name', 'Region', 'Branch', 'Risk Score', 'Risk Band', 'Assignee',
      'Audit Status', 'Last Audit Date', 'AYR', 'DQI', 'FIMR', 'All-Time FIMR', 'D0-6 Slippage',
      'Roll', 'FRR', 'Portfolio Total', 'Overdue >15D', 'Active Loans', 'Channel',
      'Yield', 'PORR', 'Channel Purity', 'Rank',
      'Avg Timeliness Score', 'Avg Repayment Health', 'Avg Days Since Last Repayment', 'Avg Loan Age', 'Repayment Delay Rate'
    ];

    const rows = sortedAgents.map(agent => [
      agent.officerName,
      agent.region,
      agent.branch,
      agent.riskScore,
      agent.riskBand,
      agent.assignee,
      agent.auditStatus,
      agent.lastAuditDate || 'Never',
      agent.ayr,
      agent.dqi,
      agent.fimr,
      agent.allTimeFimr,
      agent.d06Slippage,
      agent.roll,
      agent.frr,
      agent.portfolioTotal,
      agent.overdue15d,
      agent.activeLoans,
      agent.channel,
      agent.yield,
      agent.porr,
      agent.channelPurity,
      agent.rank,
      agent.avgTimelinessScore != null ? agent.avgTimelinessScore.toFixed(2) : 'N/A',
      agent.avgRepaymentHealth != null ? agent.avgRepaymentHealth.toFixed(2) : 'N/A',
      agent.avgDaysSinceLastRepayment != null ? agent.avgDaysSinceLastRepayment.toFixed(1) : 'N/A',
      agent.avgLoanAge != null ? agent.avgLoanAge.toFixed(1) : 'N/A',
      agent.repaymentDelayRate != null ? agent.repaymentDelayRate.toFixed(2) + '%' : 'N/A',
    ]);

    const csvContent = [
      headers.join(','),
      ...rows.map(row => row.map(cell => `"${cell}"`).join(','))
    ].join('\n');

    // Download CSV
    const blob = new Blob([csvContent], { type: 'text/csv;charset=utf-8;' });
    const link = document.createElement('a');
    const url = URL.createObjectURL(blob);
    link.setAttribute('href', url);
    link.setAttribute('download', `Agent_Performance_${new Date().toISOString().split('T')[0]}.csv`);
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

  // Get performance band for Repayment Delay Rate (9-tier system)
  const getRepaymentDelayBand = (rate) => {
    if (rate == null) return { band: 'N/A', color: 'gray' };

    // 9-tier system: Healthy 1-3, Watch 1-3, Risky 1-3
    // Assuming 100% is divided into 9 equal sections (~11.11% each)
    // Higher rate = better performance
    if (rate >= 88.89) return { band: 'Healthy 1', color: 'green-dark' };
    if (rate >= 77.78) return { band: 'Healthy 2', color: 'green' };
    if (rate >= 66.67) return { band: 'Healthy 3', color: 'green-light' };
    if (rate >= 55.56) return { band: 'Watch 1', color: 'yellow-dark' };
    if (rate >= 44.44) return { band: 'Watch 2', color: 'yellow' };
    if (rate >= 33.33) return { band: 'Watch 3', color: 'yellow-light' };
    if (rate >= 22.22) return { band: 'Risky 1', color: 'red-light' };
    if (rate >= 11.11) return { band: 'Risky 2', color: 'red' };
    return { band: 'Risky 3', color: 'red-dark' };
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

  const getRiskBandColor = (band) => {
    switch (band) {
      case 'Green': return 'band-green';
      case 'Watch': return 'band-watch';
      case 'Amber': return 'band-amber';
      case 'Red': return 'band-red';
      default: return 'band-gray';
    }
  };

  const activeFilterCount = Object.values(filters).filter(v => v !== '').length;

  return (
    <div className="agent-performance">
      <div className="agent-header">
        <div className="agent-title">
          <h3>Agent Performance - Officer-Level Metrics</h3>
          <span className="agent-count">{sortedAgents.length} officers</span>
        </div>
        <div className="agent-actions">
          <button
            className={`filter-toggle ${showFilters ? 'active' : ''}`}
            onClick={() => setShowFilters(!showFilters)}
          >
            <Filter size={16} />
            Filters
            {activeFilterCount > 0 && <span className="filter-badge">{activeFilterCount}</span>}
          </button>
          <button className="export-btn" onClick={handleExport}>
            <Download size={16} />
            Export CSV
          </button>
        </div>
      </div>

      {showFilters && (
        <div className="filter-panel">
          <div className="filter-row">
            <div className="filter-group">
              <select value={filters.region} onChange={(e) => handleFilterChange('region', e.target.value)}>
                <option value="">All Regions</option>
                {filterOptions.regions.map(region => (
                  <option key={region} value={region}>{region}</option>
                ))}
              </select>
            </div>
            <div className="filter-group">
              <select value={filters.branch} onChange={(e) => handleFilterChange('branch', e.target.value)}>
                <option value="">All Branches</option>
                {filterOptions.branches.map(branch => (
                  <option key={branch} value={branch}>{branch}</option>
                ))}
              </select>
            </div>
            <div className="filter-group">
              <select value={filters.riskBand} onChange={(e) => handleFilterChange('riskBand', e.target.value)}>
                <option value="">All Risk Bands</option>
                {filterOptions.riskBands.map(band => (
                  <option key={band} value={band}>{band}</option>
                ))}
              </select>
            </div>
            <div className="filter-group date-range-group">
              <div className="date-inputs">
                <input
                  type="date"
                  value={filters.startDate}
                  onChange={(e) => handleFilterChange('startDate', e.target.value)}
                  placeholder="Start Date"
                />
                <span className="date-separator">to</span>
                <input
                  type="date"
                  value={filters.endDate}
                  onChange={(e) => handleFilterChange('endDate', e.target.value)}
                  placeholder="End Date"
                />
              </div>
            </div>
            <div className="filter-group">
              <button className="clear-filters" onClick={clearFilters}>Clear All</button>
            </div>
          </div>
        </div>
      )}

      <div className="agent-table-container">
        <table className="agent-table">
          <thead>
            <tr>
              <th onClick={() => handleSort('officerName')}>Officer Name</th>
              <th onClick={() => handleSort('region')}>Region</th>
              <th onClick={() => handleSort('branch')}>Branch</th>
              <th onClick={() => handleSort('riskScore')}>Risk Score</th>
              <th onClick={() => handleSort('riskBand')}>Risk Band</th>
              <th onClick={() => handleSort('assignee')}>Assignee</th>
              <th onClick={() => handleSort('auditStatus')}>Audit Status</th>
              <th onClick={() => handleSort('lastAuditDate')}>Last Audit Date</th>
              <th onClick={() => handleSort('ayr')}>AYR</th>
              <th onClick={() => handleSort('dqi')}>DQI</th>
              <th onClick={() => handleSort('fimr')}>FIMR</th>
              <th onClick={() => handleSort('allTimeFimr')}>All-Time FIMR</th>
              <th onClick={() => handleSort('d06Slippage')}>D0-6 Slippage</th>
              <th onClick={() => handleSort('roll')}>Roll</th>
              <th onClick={() => handleSort('frr')}>FRR</th>
              <th onClick={() => handleSort('portfolioTotal')}>Portfolio Total</th>
              <th onClick={() => handleSort('overdue15d')}>Overdue &gt;15D</th>
              <th onClick={() => handleSort('activeLoans')}>Active Loans</th>
              <th onClick={() => handleSort('channel')}>Channel</th>
              <th onClick={() => handleSort('yield')}>Yield</th>
              <th onClick={() => handleSort('porr')}>PORR</th>
              <th onClick={() => handleSort('channelPurity')}>Channel Purity</th>
              <th onClick={() => handleSort('rank')}>Rank</th>
              <th onClick={() => handleSort('avgTimelinessScore')}>Avg Timeliness Score</th>
              <th onClick={() => handleSort('avgRepaymentHealth')}>Avg Repayment Health</th>
              <th onClick={() => handleSort('avgDaysSinceLastRepayment')}>Avg Days Since Last Repayment</th>
              <th onClick={() => handleSort('avgLoanAge')}>Avg Loan Age</th>
              <th onClick={() => handleSort('repaymentDelayRate')}>Repayment Delay Rate</th>
              <th>Action</th>
            </tr>
          </thead>
          <tbody>
            {sortedAgents.map((agent, index) => (
              <tr key={agent.officerName}>
                <td className="officer-name">{agent.officerName}</td>
                <td>{agent.region}</td>
                <td>{agent.branch}</td>
                <td className="risk-score">{formatDecimal(agent.riskScore, 1)}</td>
                <td>
                  <span className={`band-badge ${getRiskBandColor(agent.riskBand)}`}>
                    {agent.riskBand}
                  </span>
                </td>
                <td>
                  <select
                    value={agent.assignee}
                    onChange={(e) => handleAssigneeChange(agent.officerName, e.target.value)}
                    className="assignee-select"
                  >
                    {mockTeamMembers.map(member => (
                      <option key={member.id} value={member.name}>
                        {member.name}
                      </option>
                    ))}
                  </select>
                </td>
                <td>
                  <select
                    value={agent.auditStatus}
                    onChange={(e) => handleAuditStatusChange(agent.officerName, e.target.value)}
                    className="audit-status-select"
                  >
                    <option value="In Progress">In Progress</option>
                    <option value="Assigned">Assigned</option>
                    <option value="Resolved">Resolved</option>
                  </select>
                </td>
                <td className="audit-date">
                  {agent.lastAuditDate ? formatDate(agent.lastAuditDate) : <span className="never-audited">Never</span>}
                </td>
                <td className="metric">{formatPercent(agent.ayr)}</td>
                <td className="metric">{formatDecimal(agent.dqi, 0)}</td>
                <td className="metric">{formatPercent(agent.fimr)}</td>
                <td className="metric all-time-fimr">{formatPercent(agent.allTimeFimr)}</td>
                <td className="metric">{formatPercent(agent.d06Slippage)}</td>
                <td className="metric">{formatPercent(agent.roll)}</td>
                <td className="metric">{formatPercent(agent.frr)}</td>
                <td className="amount">{formatCurrency(agent.portfolioTotal)}</td>
                <td className="amount">{formatCurrency(agent.overdue15d)}</td>
                <td className="count">{agent.activeLoans}</td>
                <td>{agent.channel}</td>
                <td className="metric">{formatPercent(agent.yield)}</td>
                <td className="metric">{formatPercent(agent.porr)}</td>
                <td className="metric">{formatPercent(agent.channelPurity)}</td>
                <td className="rank">#{agent.rank}</td>
                <td className="metric">{agent.avgTimelinessScore != null ? formatDecimal(agent.avgTimelinessScore, 2) : 'N/A'}</td>
                <td className="metric">{agent.avgRepaymentHealth != null ? formatDecimal(agent.avgRepaymentHealth, 2) : 'N/A'}</td>
                <td className="metric">{agent.avgDaysSinceLastRepayment != null ? formatDecimal(agent.avgDaysSinceLastRepayment, 1) : 'N/A'}</td>
                <td className="metric">{agent.avgLoanAge != null ? formatDecimal(agent.avgLoanAge, 1) : 'N/A'}</td>
                <td className="metric">
                  {agent.repaymentDelayRate != null ? (
                    <span className={`delay-rate-badge ${getRepaymentDelayBand(agent.repaymentDelayRate).color}`}>
                      {formatDecimal(agent.repaymentDelayRate, 2)}%
                      <span className="band-label">{getRepaymentDelayBand(agent.repaymentDelayRate).band}</span>
                    </span>
                  ) : 'N/A'}
                </td>
                <td>
                  <div className="action-dropdown">
                    <button
                      className="action-button"
                      onClick={() => setActiveActionMenu(activeActionMenu === agent.officerName ? null : agent.officerName)}
                    >
                      Actions <ChevronDown size={14} />
                    </button>
                    {activeActionMenu === agent.officerName && (
                      <div className="action-menu">
                        <button onClick={() => handleAction(agent, 'audit20')}>
                          Audit 20 Top Risk Loans
                        </button>
                        <button onClick={() => handleAction(agent, 'freeze')}>
                          Freeze Disbursement
                        </button>
                        <button onClick={() => handleAction(agent, 'viewPortfolio')}>
                          View Entire Portfolio
                        </button>
                        <button onClick={() => handleAction(agent, 'exportPortfolio')}>
                          Export Entire Portfolio
                        </button>
                      </div>
                    )}
                  </div>
                </td>
              </tr>
            ))}
          </tbody>
        </table>
      </div>

      {/* Top Risk Loans Modal */}
      <TopRiskLoansModal
        isOpen={topRiskModalOpen}
        onClose={() => setTopRiskModalOpen(false)}
        officerName={selectedOfficer?.name || ''}
        officerId={selectedOfficer?.id || ''}
      />
    </div>
  );
};

export default AgentPerformance;

