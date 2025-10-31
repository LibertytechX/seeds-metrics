import React, { useState, useEffect, useMemo } from 'react';
import { Download, Filter, FileText, Eye, RefreshCw } from 'lucide-react';
import jsPDF from 'jspdf';
import 'jspdf-autotable';
import LoanRepaymentsModal from './LoanRepaymentsModal';
import './AllLoans.css';

const AllLoans = ({ initialLoans = [], initialFilter = null }) => {
  const [loans, setLoans] = useState(initialLoans);
  const [loading, setLoading] = useState(false);
  const [sortConfig, setSortConfig] = useState({ key: 'disbursement_date', direction: 'desc' });
  const [filters, setFilters] = useState({
    officer_id: initialFilter?.officer_id || '',
    branch: '',
    region: '',
    channel: '',
    status: '',
    loan_type: initialFilter?.loan_type || '', // 'active' or 'inactive'
    rot_type: initialFilter?.rot_type || '', // 'early' or 'late'
  });
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

  // Fetch loans from API
  const fetchLoans = async () => {
    setLoading(true);
    try {
      console.log('🔍 AllLoans: fetchLoans called with filters:', filters);

      // Exclude loan_type and rot_type from API params (client-side filtering)
      const apiFilters = Object.fromEntries(
        Object.entries(filters).filter(([k, v]) => v !== '' && k !== 'loan_type' && k !== 'rot_type')
      );

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
        let fetchedLoans = data.data.loans || [];
        console.log(`📦 AllLoans: Fetched ${fetchedLoans.length} loans from API`);

        // Apply client-side filtering for loan_type and rot_type
        if (filters.loan_type === 'active') {
          console.log('🔵 Filtering for ACTIVE loans');
          fetchedLoans = fetchedLoans.filter(loan =>
            loan.total_outstanding > 2000 && loan.days_since_last_repayment < 6
          );
          console.log(`✅ Active loans filtered: ${fetchedLoans.length} loans`);
        } else if (filters.loan_type === 'inactive') {
          console.log('🔵 Filtering for INACTIVE loans');
          fetchedLoans = fetchedLoans.filter(loan =>
            loan.total_outstanding <= 2000 || loan.days_since_last_repayment > 5
          );
          console.log(`✅ Inactive loans filtered: ${fetchedLoans.length} loans`);
        } else if (filters.loan_type === 'overdue_15d') {
          console.log('🔵 Filtering for OVERDUE >15 Days loans');
          fetchedLoans = fetchedLoans.filter(loan =>
            loan.current_dpd > 15
          );
          console.log(`✅ Overdue >15 Days loans filtered: ${fetchedLoans.length} loans`);
        }

        if (filters.rot_type === 'early') {
          console.log('🔵 Filtering for EARLY ROT loans');
          fetchedLoans = fetchedLoans.filter(loan => {
            const loanAge = Math.floor((new Date() - new Date(loan.disbursement_date)) / (1000 * 60 * 60 * 24));
            return loanAge < 7 && loan.current_dpd > 4;
          });
          console.log(`✅ Early ROT loans filtered: ${fetchedLoans.length} loans`);
        } else if (filters.rot_type === 'late') {
          console.log('🔵 Filtering for LATE ROT loans');
          fetchedLoans = fetchedLoans.filter(loan => {
            const loanAge = Math.floor((new Date() - new Date(loan.disbursement_date)) / (1000 * 60 * 60 * 24));
            return loanAge >= 7 && loan.current_dpd > 4;
          });
          console.log(`✅ Late ROT loans filtered: ${fetchedLoans.length} loans`);
        }

        setLoans(fetchedLoans);
        setPagination({
          page: data.data.page,
          limit: data.data.limit,
          total: fetchedLoans.length, // Update total to reflect filtered count
          pages: Math.ceil(fetchedLoans.length / data.data.limit),
        });
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
        region: '',
        channel: '',
        status: '',
        loan_type: initialFilter.loan_type || '',
        rot_type: initialFilter.rot_type || '',
      });
      setFilterLabel(
        initialFilter.officer_name ? `Officer: ${initialFilter.officer_name}` :
        initialFilter.label ? initialFilter.label : ''
      );
    }
  }, [initialFilter]);

  useEffect(() => {
    fetchLoans();
  }, [pagination.page, pagination.limit, sortConfig, filters]);

  // Get unique values for filter dropdowns
  const filterOptions = useMemo(() => {
    return {
      officers: [...new Set(loans.map(l => l.officer_name))].filter(Boolean).sort(),
      branches: [...new Set(loans.map(l => l.branch))].filter(Boolean).sort(),
      regions: [...new Set(loans.map(l => l.region))].filter(Boolean).sort(),
      channels: [...new Set(loans.map(l => l.channel))].filter(Boolean).sort(),
      statuses: [...new Set(loans.map(l => l.status))].filter(Boolean).sort(),
    };
  }, [loans]);

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

  const clearFilters = () => {
    setFilters({
      officer_id: '',
      branch: '',
      region: '',
      channel: '',
      status: '',
      loan_type: '',
      rot_type: '',
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
      'Channel', 'Loan Amount', 'Repayment Amount', 'Disbursement Date', 'Loan Tenure', 'Maturity Date',
      'Timeliness Score', 'Repayment Health', 'Days Since Last Repayment', 'Current DPD',
      'Principal Outstanding', 'Interest Outstanding', 'Fees Outstanding', 'Total Outstanding',
      'Actual Outstanding', 'Total Repayments', 'Status', 'FIMR Tagged'
    ];

    const rows = loans.map(loan => [
      loan.loan_id,
      loan.customer_name,
      loan.customer_phone || '',
      loan.officer_name,
      loan.region,
      loan.branch,
      loan.channel,
      loan.loan_amount,
      loan.repayment_amount || 'N/A',
      loan.disbursement_date,
      formatTenure(loan.loan_term_days),
      loan.maturity_date,
      loan.timeliness_score != null ? loan.timeliness_score.toFixed(2) : 'N/A',
      loan.repayment_health != null ? loan.repayment_health.toFixed(2) : 'N/A',
      loan.days_since_last_repayment != null ? loan.days_since_last_repayment : 'N/A',
      loan.current_dpd,
      loan.principal_outstanding,
      loan.interest_outstanding,
      loan.fees_outstanding,
      loan.total_outstanding,
      loan.actual_outstanding || 0,
      loan.total_repayments || 0,
      loan.status,
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
      loan.officer_name,
      loan.branch,
      `₦${(loan.loan_amount / 1000000).toFixed(2)}M`,
      loan.repayment_amount ? `₦${(loan.repayment_amount / 1000000).toFixed(2)}M` : 'N/A',
      loan.disbursement_date,
      formatTenure(loan.loan_term_days),
      loan.timeliness_score != null ? loan.timeliness_score.toFixed(1) : 'N/A',
      loan.repayment_health != null ? loan.repayment_health.toFixed(1) : 'N/A',
      loan.days_since_last_repayment != null ? loan.days_since_last_repayment : 'N/A',
      loan.current_dpd,
      `₦${(loan.total_outstanding / 1000000).toFixed(2)}M`,
      `₦${((loan.actual_outstanding || 0) / 1000000).toFixed(2)}M`,
      loan.status,
      loan.fimr_tagged ? 'Yes' : 'No',
    ]);

    doc.autoTable({
      startY: 32,
      head: [['Loan ID', 'Customer', 'Officer', 'Branch', 'Amount', 'Repay. Amt', 'Disbursed', 'Tenure', 'T.Score', 'R.Health', 'Days Since', 'DPD', 'Total Out.', 'Actual Out.', 'Status', 'FIMR']],
      body: tableData,
      styles: { fontSize: 6 },
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

  const handleRecalculateFields = async () => {
    setRecalculating(true);
    setRecalculateMessage('');

    try {
      const response = await fetch('http://localhost:8081/api/v1/loans/recalculate-fields', {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
        },
      });

      const result = await response.json();

      if (response.ok && result.status === 'success') {
        setRecalculateMessage(`✓ Successfully recalculated fields for ${result.data.loans_updated} loans`);
        // Refresh the loans table after successful recalculation
        await fetchLoans();

        // Clear success message after 5 seconds
        setTimeout(() => {
          setRecalculateMessage('');
        }, 5000);
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
          <button className="export-button" onClick={handleExportCSV}>
            <Download size={16} />
            Export CSV
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
                value={filters.status}
                onChange={(e) => handleFilterChange('status', e.target.value)}
              >
                <option value="">All Statuses</option>
                {filterOptions.statuses.map(status => (
                  <option key={status} value={status}>{status}</option>
                ))}
              </select>
            </div>
            <div className="filter-group">
              <button className="clear-filters" onClick={clearFilters}>Clear All</button>
            </div>
          </div>
        </div>
      )}

      <div className="pagination-controls">
        <div className="rows-per-page">
          <label>Rows per page:</label>
          <select value={pagination.limit} onChange={(e) => handleLimitChange(e.target.value)}>
            <option value="10">10</option>
            <option value="25">25</option>
            <option value="50">50</option>
            <option value="100">100</option>
          </select>
        </div>
        <div className="page-info">
          Page {pagination.page} of {pagination.pages} ({pagination.total} total loans)
        </div>
        <div className="page-buttons">
          <button
            onClick={() => handlePageChange(1)}
            disabled={pagination.page === 1}
          >
            First
          </button>
          <button
            onClick={() => handlePageChange(pagination.page - 1)}
            disabled={pagination.page === 1}
          >
            Previous
          </button>
          <button
            onClick={() => handlePageChange(pagination.page + 1)}
            disabled={pagination.page === pagination.pages}
          >
            Next
          </button>
          <button
            onClick={() => handlePageChange(pagination.pages)}
            disabled={pagination.page === pagination.pages}
          >
            Last
          </button>
        </div>
      </div>

      <div className="all-loans-table-container">
        {loading ? (
          <div className="loading">Loading...</div>
        ) : (
          <table className="all-loans-table">
            <thead>
              <tr>
                <th onClick={() => handleSort('loan_id')}>Loan ID</th>
                <th onClick={() => handleSort('customer_name')}>Customer Name</th>
                <th onClick={() => handleSort('officer_name')}>Officer Name</th>
                <th onClick={() => handleSort('region')}>Region</th>
                <th onClick={() => handleSort('branch')}>Branch</th>
                <th onClick={() => handleSort('channel')}>Channel</th>
                <th onClick={() => handleSort('loan_amount')}>Loan Amount</th>
                <th onClick={() => handleSort('repayment_amount')}>Repayment Amount</th>
                <th onClick={() => handleSort('disbursement_date')}>Disbursement Date</th>
                <th onClick={() => handleSort('loan_term_days')}>Loan Tenure</th>
                <th onClick={() => handleSort('timeliness_score')}>Timeliness Score</th>
                <th onClick={() => handleSort('repayment_health')}>Repayment Health</th>
                <th onClick={() => handleSort('days_since_last_repayment')}>Days Since Last Repayment</th>
                <th onClick={() => handleSort('current_dpd')}>Current DPD</th>
                <th onClick={() => handleSort('total_outstanding')}>Total Outstanding</th>
                <th onClick={() => handleSort('actual_outstanding')}>Actual Outstanding</th>
              <th onClick={() => handleSort('total_repayments')}>Total Repayments</th>
                <th onClick={() => handleSort('status')}>Status</th>
                <th>FIMR Tagged</th>
                <th>Actions</th>
              </tr>
            </thead>
            <tbody>
              {loans.map((loan) => (
                <tr key={loan.loan_id}>
                  <td className="loan-id">{loan.loan_id}</td>
                  <td>{loan.customer_name}</td>
                  <td>{loan.officer_name}</td>
                  <td>{loan.region}</td>
                  <td>{loan.branch}</td>
                  <td>{loan.channel}</td>
                  <td className="amount">{formatCurrency(loan.loan_amount)}</td>
                  <td className="amount">{loan.repayment_amount ? formatCurrency(loan.repayment_amount) : 'N/A'}</td>
                  <td>{formatDate(loan.disbursement_date)}</td>
                  <td className="tenure">{formatTenure(loan.loan_term_days)}</td>
                  <td className="score">{loan.timeliness_score != null ? loan.timeliness_score.toFixed(2) : 'N/A'}</td>
                  <td className="score">{loan.repayment_health != null ? loan.repayment_health.toFixed(2) : 'N/A'}</td>
                  <td className="days-since">{loan.days_since_last_repayment != null ? loan.days_since_last_repayment : 'N/A'}</td>
                  <td className="dpd">{loan.current_dpd}</td>
                  <td className="amount">{formatCurrency(loan.total_outstanding)}</td>
                  <td className="amount">{formatCurrency(loan.actual_outstanding || 0)}</td>
                  <td className="amount">{formatCurrency(loan.total_repayments || 0)}</td>
                  <td>
                    <span className={`status-badge status-${loan.status.toLowerCase()}`}>
                      {loan.status}
                    </span>
                  </td>
                  <td className="fimr-tagged">
                    {loan.fimr_tagged ? (
                      <span className="badge-yes">Yes</span>
                    ) : (
                      <span className="badge-no">No</span>
                    )}
                  </td>
                  <td className="action-cell">
                    <button
                      className="view-repayments-btn"
                      onClick={() => handleViewRepayments(loan)}
                      title="View Repayment History"
                    >
                      <Eye size={16} />
                      Repayments
                    </button>
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        )}
      </div>

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

