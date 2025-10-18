import React, { useState, useMemo } from 'react';
import { Download, Filter } from 'lucide-react';
import './FIMRDrilldown.css';

const FIMRDrilldown = ({ loans }) => {
  const [sortConfig, setSortConfig] = useState({ key: 'daysSinceDue', direction: 'desc' });
  const [filters, setFilters] = useState({
    officer: '',
    region: '',
    branch: '',
    channel: '',
    status: '',
    startDate: '',
    endDate: '',
  });
  const [showFilters, setShowFilters] = useState(false);

  // Get unique values for filter dropdowns
  const filterOptions = useMemo(() => {
    return {
      officers: [...new Set(loans.map(l => l.officerName))].sort(),
      regions: [...new Set(loans.map(l => l.region))].sort(),
      branches: [...new Set(loans.map(l => l.branch))].sort(),
      channels: [...new Set(loans.map(l => l.channel))].sort(),
      statuses: [...new Set(loans.map(l => l.status))].sort(),
    };
  }, [loans]);

  // Apply filters
  const filteredLoans = useMemo(() => {
    return loans.filter(loan => {
      if (filters.officer && loan.officerName !== filters.officer) return false;
      if (filters.region && loan.region !== filters.region) return false;
      if (filters.branch && loan.branch !== filters.branch) return false;
      if (filters.channel && loan.channel !== filters.channel) return false;
      if (filters.status && loan.status !== filters.status) return false;

      // Date range filter
      if (filters.startDate && loan.disbursementDate < filters.startDate) return false;
      if (filters.endDate && loan.disbursementDate > filters.endDate) return false;

      return true;
    });
  }, [loans, filters]);

  // Apply sorting
  const sortedLoans = useMemo(() => {
    const sorted = [...filteredLoans];
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
  }, [filteredLoans, sortConfig]);

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
      officer: '',
      region: '',
      branch: '',
      channel: '',
      status: '',
      startDate: '',
      endDate: '',
    });
  };

  const handleExport = () => {
    // Create CSV content
    const headers = [
      'Loan ID', 'Officer Name', 'Region', 'Branch', 'Customer Name', 'Customer Phone Number',
      'Disbursement Date', 'Loan Amount', 'First Payment Due Date',
      'Days Since Due', 'Amount Due (1st Installment)', 'Amount Paid',
      'Outstanding Balance', 'Current DPD', 'Channel', 'Status', 'FIMR Tagged'
    ];

    const rows = sortedLoans.map(loan => [
      loan.loanId,
      loan.officerName,
      loan.region,
      loan.branch,
      loan.customerName,
      loan.customerPhone,
      loan.disbursementDate,
      loan.loanAmount,
      loan.firstPaymentDueDate,
      loan.daysSinceDue,
      loan.amountDue1stInstallment,
      loan.amountPaid,
      loan.outstandingBalance,
      loan.currentDPD,
      loan.channel,
      loan.status,
      loan.fimrTagged ? 'True' : 'False',
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
    link.setAttribute('download', `FIMR_Drilldown_${new Date().toISOString().split('T')[0]}.csv`);
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

  const formatDate = (dateString) => {
    const date = new Date(dateString);
    return date.toLocaleDateString('en-GB', { day: '2-digit', month: 'short', year: 'numeric' });
  };

  const getStatusColor = (status) => {
    switch (status) {
      case 'First Payment Missed': return 'status-flag';
      case 'Partially Paid': return 'status-watch';
      case 'Defaulted': return 'status-red';
      default: return 'status-gray';
    }
  };

  const activeFilterCount = Object.values(filters).filter(v => v !== '').length;

  return (
    <div className="fimr-drilldown">
      <div className="fimr-header">
        <div className="fimr-title">
          <h3>FIMR Drilldown - Loans with Missed First Installment</h3>
          <span className="loan-count">{sortedLoans.length} loans</span>
        </div>
        <div className="fimr-actions">
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
              <label>Officer</label>
              <select value={filters.officer} onChange={(e) => handleFilterChange('officer', e.target.value)}>
                <option value="">All Officers</option>
                {filterOptions.officers.map(officer => (
                  <option key={officer} value={officer}>{officer}</option>
                ))}
              </select>
            </div>
            <div className="filter-group">
              <label>Region</label>
              <select value={filters.region} onChange={(e) => handleFilterChange('region', e.target.value)}>
                <option value="">All Regions</option>
                {filterOptions.regions.map(region => (
                  <option key={region} value={region}>{region}</option>
                ))}
              </select>
            </div>
            <div className="filter-group">
              <label>Branch</label>
              <select value={filters.branch} onChange={(e) => handleFilterChange('branch', e.target.value)}>
                <option value="">All Branches</option>
                {filterOptions.branches.map(branch => (
                  <option key={branch} value={branch}>{branch}</option>
                ))}
              </select>
            </div>
            <div className="filter-group">
              <label>Channel</label>
              <select value={filters.channel} onChange={(e) => handleFilterChange('channel', e.target.value)}>
                <option value="">All Channels</option>
                {filterOptions.channels.map(channel => (
                  <option key={channel} value={channel}>{channel}</option>
                ))}
              </select>
            </div>
            <div className="filter-group">
              <label>Status</label>
              <select value={filters.status} onChange={(e) => handleFilterChange('status', e.target.value)}>
                <option value="">All Statuses</option>
                {filterOptions.statuses.map(status => (
                  <option key={status} value={status}>{status}</option>
                ))}
              </select>
            </div>
            <div className="filter-group date-range-group">
              <label>Date Range (Disbursement)</label>
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

      <div className="fimr-table-container">
        <table className="fimr-table">
          <thead>
            <tr>
              <th onClick={() => handleSort('loanId')}>Loan ID</th>
              <th onClick={() => handleSort('officerName')}>Officer Name</th>
              <th onClick={() => handleSort('region')}>Region</th>
              <th onClick={() => handleSort('branch')}>Branch</th>
              <th onClick={() => handleSort('customerName')}>Customer Name</th>
              <th onClick={() => handleSort('customerPhone')}>Customer Phone</th>
              <th onClick={() => handleSort('disbursementDate')}>Disbursement Date</th>
              <th onClick={() => handleSort('loanAmount')}>Loan Amount</th>
              <th onClick={() => handleSort('firstPaymentDueDate')}>First Payment Due</th>
              <th onClick={() => handleSort('daysSinceDue')}>Days Since Due</th>
              <th onClick={() => handleSort('amountDue1stInstallment')}>Amount Due (1st)</th>
              <th onClick={() => handleSort('amountPaid')}>Amount Paid</th>
              <th onClick={() => handleSort('outstandingBalance')}>Outstanding Balance</th>
              <th onClick={() => handleSort('currentDPD')}>Current DPD</th>
              <th onClick={() => handleSort('channel')}>Channel</th>
              <th onClick={() => handleSort('status')}>Status</th>
              <th onClick={() => handleSort('fimrTagged')}>FIMR Tagged</th>
            </tr>
          </thead>
          <tbody>
            {sortedLoans.map(loan => (
              <tr key={loan.loanId}>
                <td className="loan-id">{loan.loanId}</td>
                <td>{loan.officerName}</td>
                <td>{loan.region}</td>
                <td>{loan.branch}</td>
                <td className="customer-name">{loan.customerName}</td>
                <td className="phone-number">{loan.customerPhone}</td>
                <td>{formatDate(loan.disbursementDate)}</td>
                <td className="amount">{formatCurrency(loan.loanAmount)}</td>
                <td>{formatDate(loan.firstPaymentDueDate)}</td>
                <td className="days-overdue">{loan.daysSinceDue}</td>
                <td className="amount">{formatCurrency(loan.amountDue1stInstallment)}</td>
                <td className="amount">{formatCurrency(loan.amountPaid)}</td>
                <td className="amount">{formatCurrency(loan.outstandingBalance)}</td>
                <td className="dpd">{loan.currentDPD}</td>
                <td>{loan.channel}</td>
                <td>
                  <span className={`status-badge ${getStatusColor(loan.status)}`}>
                    {loan.status}
                  </span>
                </td>
                <td>
                  <span className={`fimr-badge ${loan.fimrTagged ? 'fimr-true' : 'fimr-false'}`}>
                    {loan.fimrTagged ? 'True' : 'False'}
                  </span>
                </td>
              </tr>
            ))}
          </tbody>
        </table>
      </div>
    </div>
  );
};

export default FIMRDrilldown;

