import React, { useState, useEffect } from 'react';
import { X, Download, FileText } from 'lucide-react';
import jsPDF from 'jspdf';
import 'jspdf-autotable';
import './LoanRepaymentsModal.css';

const LoanRepaymentsModal = ({
  isOpen,
  onClose,
  loanId,
  customerName,
  repaymentAmount,
  loanAmount,
  totalOutstanding,
  actualOutstanding,
  maturityDate,
  firstPaymentDueDate,
}) => {
  const [repayments, setRepayments] = useState([]);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState(null);
  const [sortConfig, setSortConfig] = useState({ key: 'payment_date', direction: 'desc' });

  useEffect(() => {
    if (isOpen && loanId) {
      fetchRepayments();
    }
  }, [isOpen, loanId]);

  const fetchRepayments = async () => {
    setLoading(true);
    setError(null);
    try {
      const API_BASE_URL = import.meta.env.VITE_API_URL || 'http://localhost:8081/api/v1';
      const response = await fetch(`${API_BASE_URL}/loans/${loanId}/repayments`);
      const data = await response.json();

      if (data.status === 'success') {
        setRepayments(data.data.repayments || []);
      } else {
        setError(data.message || 'Failed to fetch repayments');
      }
    } catch (err) {
      console.error('Error fetching repayments:', err);
      setError('Failed to connect to the server');
    } finally {
      setLoading(false);
    }
  };

  const handleSort = (key) => {
    let direction = 'asc';
    if (sortConfig.key === key && sortConfig.direction === 'asc') {
      direction = 'desc';
    }
    setSortConfig({ key, direction });
  };

  const sortedRepayments = React.useMemo(() => {
    if (!repayments || repayments.length === 0) return [];

    const sorted = [...repayments].sort((a, b) => {
      let aVal = a[sortConfig.key];
      let bVal = b[sortConfig.key];

      // Handle date sorting
      if (sortConfig.key === 'payment_date') {
        aVal = new Date(aVal);
        bVal = new Date(bVal);
      }

      // Handle numeric sorting
      if (typeof aVal === 'number' && typeof bVal === 'number') {
        return sortConfig.direction === 'asc' ? aVal - bVal : bVal - aVal;
      }

      // Handle string sorting
      if (aVal < bVal) return sortConfig.direction === 'asc' ? -1 : 1;
      if (aVal > bVal) return sortConfig.direction === 'asc' ? 1 : -1;
      return 0;
    });

    return sorted;
  }, [repayments, sortConfig]);

  const totalRepaymentsAmount = React.useMemo(() => {
    if (!repayments || repayments.length === 0) return 0;

    return repayments
      .filter(r => !r.is_reversed)
      .reduce((sum, r) => {
        const amt = Number(r.payment_amount ?? 0);
        if (Number.isNaN(amt)) return sum;
        return sum + amt;
      }, 0);
  }, [repayments]);

  const formatCurrency = (value) => {
    return new Intl.NumberFormat('en-NG', {
      style: 'currency',
      currency: 'NGN',
      minimumFractionDigits: 2,
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

  const handleExportCSV = () => {
    const headers = [
      'Repayment ID', 'Payment Date', 'Payment Amount', 'Principal Paid',
      'Interest Paid', 'Fees Paid', 'Penalty Paid', 'Payment Method',
      'Payment Reference', 'DPD at Payment', 'Backdated', 'Reversed'
    ];

    const rows = sortedRepayments.map(r => [
      r.repayment_id,
      formatDate(r.payment_date),
      r.payment_amount,
      r.principal_paid,
      r.interest_paid,
      r.fees_paid,
      r.penalty_paid || 0,
      r.payment_method,
      r.payment_reference || '',
      r.dpd_at_payment,
      r.is_backdated ? 'Yes' : 'No',
      r.is_reversed ? 'Yes' : 'No',
    ]);

    const csvContent = [
      headers.join(','),
      ...rows.map(row => row.join(','))
    ].join('\n');

    const blob = new Blob([csvContent], { type: 'text/csv' });
    const url = window.URL.createObjectURL(blob);
    const a = document.createElement('a');
    a.href = url;
    a.download = `Repayments_${loanId}_${new Date().toISOString().split('T')[0]}.csv`;
    a.click();
    window.URL.revokeObjectURL(url);
  };

  const handleExportPDF = () => {
    const doc = new jsPDF('landscape');

    doc.setFontSize(16);
    doc.text(`Repayment History - ${loanId}`, 14, 15);
    doc.setFontSize(10);
    doc.text(`Customer: ${customerName}`, 14, 22);
    doc.text(`Generated: ${new Date().toLocaleString()}`, 14, 27);
    doc.text(`Total Repayments: ${repayments.length}`, 14, 32);

    const tableData = sortedRepayments.map(r => [
      r.repayment_id,
      formatDate(r.payment_date),
      formatCurrency(r.payment_amount),
      formatCurrency(r.principal_paid),
      formatCurrency(r.interest_paid),
      formatCurrency(r.fees_paid),
      r.payment_method,
      r.dpd_at_payment,
      r.is_backdated ? 'Yes' : 'No',
      r.is_reversed ? 'Yes' : 'No',
    ]);

    doc.autoTable({
      startY: 37,
      head: [['ID', 'Date', 'Amount', 'Principal', 'Interest', 'Fees', 'Method', 'DPD', 'Backdated', 'Reversed']],
      body: tableData,
      styles: { fontSize: 8 },
      headStyles: { fillColor: [41, 128, 185] },
    });

    doc.save(`Repayments_${loanId}_${new Date().toISOString().split('T')[0]}.pdf`);
  };

  if (!isOpen) return null;

  return (
    <div className="loan-repayments-modal-overlay" onClick={onClose}>
      <div className="loan-repayments-modal-content" onClick={(e) => e.stopPropagation()}>
        <div className="loan-repayments-modal-header">
          <div>
            <h2>Repayment History</h2>
            <p className="loan-repayments-subtitle">
              Loan: <strong>{loanId}</strong> | Customer: <strong>{customerName}</strong>
            </p>
            <div className="loan-repayments-summary">
              <div className="loan-repayments-summary-row">
                {(repaymentAmount != null || loanAmount != null) && (
                  <span className="loan-repayments-summary-item">
                    <span className="loan-repayments-summary-label">Repayment Amount:</span>
                    <span className="loan-repayments-summary-value">
                      {formatCurrency(
                        repaymentAmount != null ? repaymentAmount : loanAmount
                      )}
                    </span>
                  </span>
                )}
                {totalOutstanding != null && (
                  <span className="loan-repayments-summary-item">
                    <span className="loan-repayments-summary-label">Outstanding:</span>
                    <span className="loan-repayments-summary-value">{formatCurrency(totalOutstanding)}</span>
                  </span>
                )}
                {actualOutstanding != null && (
                  <span className="loan-repayments-summary-item">
	                <span className="loan-repayments-summary-label">Missed Repayments:</span>
                    <span className="loan-repayments-summary-value">{formatCurrency(actualOutstanding)}</span>
                  </span>
                )}
              </div>
              <div className="loan-repayments-summary-row">
                <span className="loan-repayments-summary-item">
                  <span className="loan-repayments-summary-label">Sum of all repayments:</span>
                  <span className="loan-repayments-summary-value">{formatCurrency(totalRepaymentsAmount)}</span>
                </span>
                {maturityDate && (
                  <span className="loan-repayments-summary-item">
                    <span className="loan-repayments-summary-label">Maturity Date:</span>
                    <span className="loan-repayments-summary-value">{formatDate(maturityDate)}</span>
                  </span>
                )}
                {firstPaymentDueDate && (
                  <span className="loan-repayments-summary-item">
                    <span className="loan-repayments-summary-label">First Payment Due Date:</span>
                    <span className="loan-repayments-summary-value">{formatDate(firstPaymentDueDate)}</span>
                  </span>
                )}
              </div>
            </div>
          </div>
          <button className="loan-repayments-close-btn" onClick={onClose}>
            <X size={24} />
          </button>
        </div>

        <div className="loan-repayments-modal-body">
          {loading ? (
            <div className="loan-repayments-loading">
              <div className="spinner"></div>
              <p>Loading repayment history...</p>
            </div>
          ) : error ? (
            <div className="loan-repayments-error">
              <p>❌ {error}</p>
              <button onClick={fetchRepayments}>Retry</button>
            </div>
          ) : repayments.length === 0 ? (
            <div className="loan-repayments-empty">
              <p>No repayments found for this loan.</p>
            </div>
          ) : (
            <>
              <div className="loan-repayments-actions">
                <div className="loan-repayments-count">
                  {repayments.length} Repayment{repayments.length !== 1 ? 's' : ''}
                </div>
                <div className="loan-repayments-export-buttons">
                  <button className="export-btn" onClick={handleExportCSV}>
                    <Download size={16} />
                    Export CSV
                  </button>
                  <button className="export-btn" onClick={handleExportPDF}>
                    <FileText size={16} />
                    Export PDF
                  </button>
                </div>
              </div>

              <div className="loan-repayments-table-container">
                <table className="loan-repayments-table">
                  <thead>
                    <tr>
                      <th onClick={() => handleSort('repayment_id')}>
                        Repayment ID {sortConfig.key === 'repayment_id' && (sortConfig.direction === 'asc' ? '↑' : '↓')}
                      </th>
                      <th onClick={() => handleSort('payment_date')}>
                        Payment Date {sortConfig.key === 'payment_date' && (sortConfig.direction === 'asc' ? '↑' : '↓')}
                      </th>
                      <th onClick={() => handleSort('payment_amount')}>
                        Amount {sortConfig.key === 'payment_amount' && (sortConfig.direction === 'asc' ? '↑' : '↓')}
                      </th>
                      <th onClick={() => handleSort('principal_paid')}>
                        Principal {sortConfig.key === 'principal_paid' && (sortConfig.direction === 'asc' ? '↑' : '↓')}
                      </th>
                      <th onClick={() => handleSort('interest_paid')}>
                        Interest {sortConfig.key === 'interest_paid' && (sortConfig.direction === 'asc' ? '↑' : '↓')}
                      </th>
                      <th onClick={() => handleSort('fees_paid')}>
                        Fees {sortConfig.key === 'fees_paid' && (sortConfig.direction === 'asc' ? '↑' : '↓')}
                      </th>
                      <th onClick={() => handleSort('penalty_paid')}>
                        Penalty {sortConfig.key === 'penalty_paid' && (sortConfig.direction === 'asc' ? '↑' : '↓')}
                      </th>
                      <th onClick={() => handleSort('payment_method')}>
                        Method {sortConfig.key === 'payment_method' && (sortConfig.direction === 'asc' ? '↑' : '↓')}
                      </th>
                      <th onClick={() => handleSort('dpd_at_payment')}>
                        DPD {sortConfig.key === 'dpd_at_payment' && (sortConfig.direction === 'asc' ? '↑' : '↓')}
                      </th>
                      <th>Status</th>
                    </tr>
                  </thead>
                  <tbody>
                    {sortedRepayments.map((repayment) => (
                      <tr key={repayment.repayment_id}>
                        <td className="repayment-id-cell">{repayment.repayment_id}</td>
                        <td>{formatDate(repayment.payment_date)}</td>
                        <td className="amount">{formatCurrency(repayment.payment_amount)}</td>
                        <td className="amount">{formatCurrency(repayment.principal_paid)}</td>
                        <td className="amount">{formatCurrency(repayment.interest_paid)}</td>
                        <td className="amount">{formatCurrency(repayment.fees_paid)}</td>
                        <td className="amount">{formatCurrency(repayment.penalty_paid || 0)}</td>
                        <td>{repayment.payment_method}</td>
                        <td className="dpd-cell">{repayment.dpd_at_payment}</td>
                        <td>
                          {repayment.is_reversed ? (
                            <span className="status-badge status-reversed">Reversed</span>
                          ) : repayment.is_backdated ? (
                            <span className="status-badge status-backdated">Backdated</span>
                          ) : (
                            <span className="status-badge status-normal">Normal</span>
                          )}
                        </td>
                      </tr>
                    ))}
                  </tbody>
                </table>
              </div>
            </>
          )}
        </div>
      </div>
    </div>
  );
};

export default LoanRepaymentsModal;

