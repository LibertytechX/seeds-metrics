import React, { useState, useMemo } from 'react';
import { Download, Filter } from 'lucide-react';
import Pagination from './Pagination';
import './EarlyIndicatorsDrilldown.css';

const EarlyIndicatorsDrilldown = ({ loans }) => {
  const [sortConfig, setSortConfig] = useState({ key: 'currentDPD', direction: 'desc' });
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
  const [pagination, setPagination] = useState({
    page: 1,
    limit: 50,
  });

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

  // Apply pagination
  const paginatedLoans = useMemo(() => {
    const startIndex = (pagination.page - 1) * pagination.limit;
    const endIndex = startIndex + pagination.limit;
    return sortedLoans.slice(startIndex, endIndex);
  }, [sortedLoans, pagination.page, pagination.limit]);

  const totalPages = Math.ceil(sortedLoans.length / pagination.limit);

  const handleSort = (key) => {
    setSortConfig(prev => ({
      key,
      direction: prev.key === key && prev.direction === 'asc' ? 'desc' : 'asc',
    }));
  };

  const handleFilterChange = (filterKey, value) => {
    setFilters(prev => ({ ...prev, [filterKey]: value }));
    setPagination(prev => ({ ...prev, page: 1 })); // Reset to first page when filter changes
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
    setPagination(prev => ({ ...prev, page: 1 })); // Reset to first page
  };

  const handlePageChange = (newPage) => {
    setPagination(prev => ({ ...prev, page: newPage }));
  };

  const handlePageSizeChange = (newLimit) => {
    setPagination({ page: 1, limit: newLimit });
  };

  const handleExport = () => {
    // Create CSV content
    const headers = [
      'Loan ID', 'Officer Name', 'Region', 'Branch', 'Customer Name', 'Customer Phone Number',
      'Disbursement Date', 'Loan Amount', 'Current DPD', 'Previous DPD Status',
      'Days in Current Status', 'Amount Due', 'Amount Paid', 'Outstanding Balance',
      'Channel', 'Status', 'FIMR Tagged', 'Roll Direction', 'Last Payment Date'
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
      loan.currentDPD,
      loan.previousDPDStatus,
      loan.daysInCurrentStatus,
      loan.amountDue,
      loan.amountPaid,
      loan.outstandingBalance,
      loan.channel,
      loan.status,
      loan.fimrTagged ? 'True' : 'False',
      loan.rollDirection,
      loan.lastPaymentDate,
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
    link.setAttribute('download', `Early_Indicators_Drilldown_${new Date().toISOString().split('T')[0]}.csv`);
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
    if (status.includes('D1-3')) return 'status-watch';
    if (status.includes('D4-6')) return 'status-flag';
    if (status.includes('D7-15')) return 'status-red';
    if (status.includes('D16-30')) return 'status-critical';
    return 'status-gray';
  };

  const getRollDirectionColor = (direction) => {
    if (direction === 'Worsening') return 'roll-worsening';
    if (direction === 'Stable') return 'roll-stable';
    if (direction === 'Improving') return 'roll-improving';
    return 'roll-neutral';
  };

  const activeFilterCount = Object.values(filters).filter(v => v !== '').length;

  return (
    <div className="early-indicators-drilldown">
      <div className="early-header">
        <div className="early-title">
          <h3>Early Indicators Drilldown - Loans in Early Delinquency</h3>
          <span className="loan-count">{sortedLoans.length} loans</span>
        </div>
        <div className="early-actions">
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

      <Pagination
        currentPage={pagination.page}
        totalPages={totalPages}
        totalRecords={sortedLoans.length}
        pageSize={pagination.limit}
        onPageChange={handlePageChange}
        onPageSizeChange={handlePageSizeChange}
        pageSizeOptions={[10, 25, 50, 100, 200]}
        position="top"
      />

      <div className="early-table-container">
        <table className="early-table">
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
              <th onClick={() => handleSort('currentDPD')}>Current DPD</th>
              <th onClick={() => handleSort('previousDPDStatus')}>Previous DPD Status</th>
              <th onClick={() => handleSort('daysInCurrentStatus')}>Days in Status</th>
              <th onClick={() => handleSort('amountDue')}>Amount Due</th>
              <th onClick={() => handleSort('amountPaid')}>Amount Paid</th>
              <th onClick={() => handleSort('outstandingBalance')}>Outstanding Balance</th>
              <th onClick={() => handleSort('channel')}>Channel</th>
              <th onClick={() => handleSort('status')}>Status</th>
              <th onClick={() => handleSort('fimrTagged')}>FIMR Tagged</th>
              <th onClick={() => handleSort('rollDirection')}>Roll Direction</th>
              <th onClick={() => handleSort('lastPaymentDate')}>Last Payment</th>
            </tr>
          </thead>
          <tbody>
            {paginatedLoans.map(loan => (
              <tr key={loan.loanId}>
                <td className="loan-id">{loan.loanId}</td>
                <td>{loan.officerName}</td>
                <td>{loan.region}</td>
                <td>{loan.branch}</td>
                <td className="customer-name">{loan.customerName}</td>
                <td className="phone-number">{loan.customerPhone}</td>
                <td>{formatDate(loan.disbursementDate)}</td>
                <td className="amount">{formatCurrency(loan.loanAmount)}</td>
                <td className="dpd-current">{loan.currentDPD}</td>
                <td>{loan.previousDPDStatus}</td>
                <td className="days-status">{loan.daysInCurrentStatus}</td>
                <td className="amount">{formatCurrency(loan.amountDue)}</td>
                <td className="amount">{formatCurrency(loan.amountPaid)}</td>
                <td className="amount">{formatCurrency(loan.outstandingBalance)}</td>
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
                <td>
                  <span className={`roll-badge ${getRollDirectionColor(loan.rollDirection)}`}>
                    {loan.rollDirection}
                  </span>
                </td>
                <td>{formatDate(loan.lastPaymentDate)}</td>
              </tr>
            ))}
          </tbody>
        </table>
      </div>

      <Pagination
        currentPage={pagination.page}
        totalPages={totalPages}
        totalRecords={sortedLoans.length}
        pageSize={pagination.limit}
        onPageChange={handlePageChange}
        onPageSizeChange={handlePageSizeChange}
        pageSizeOptions={[10, 25, 50, 100, 200]}
        position="bottom"
      />
    </div>
  );
};

export default EarlyIndicatorsDrilldown;

