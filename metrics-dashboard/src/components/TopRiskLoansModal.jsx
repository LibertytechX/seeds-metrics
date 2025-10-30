import React, { useState, useEffect, useMemo } from 'react';
import { X, Download, FileText, AlertTriangle } from 'lucide-react';
import jsPDF from 'jspdf';
import 'jspdf-autotable';
import './TopRiskLoansModal.css';

const TopRiskLoansModal = ({ isOpen, onClose, officerName, officerId }) => {
  const [loans, setLoans] = useState([]);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState(null);
  const [sortConfig, setSortConfig] = useState({ key: 'risk_score', direction: 'desc' });

  useEffect(() => {
    if (isOpen && officerId) {
      fetchTopRiskLoans();
    }
  }, [isOpen, officerId]);

  const fetchTopRiskLoans = async () => {
    setLoading(true);
    setError(null);
    try {
      const API_BASE_URL = import.meta.env.VITE_API_URL || 'http://localhost:8081/api/v1';
      const response = await fetch(`${API_BASE_URL}/officers/${officerId}/top-risk-loans?limit=20`);
      const data = await response.json();

      if (data.status === 'success') {
        setLoans(data.data.loans || []);
      } else {
        setError(data.message || 'Failed to fetch top risk loans');
      }
    } catch (err) {
      setError('Failed to connect to the server');
      console.error('Error fetching top risk loans:', err);
    } finally {
      setLoading(false);
    }
  };

  const handleSort = (key) => {
    setSortConfig(prev => ({
      key,
      direction: prev.key === key && prev.direction === 'asc' ? 'desc' : 'asc'
    }));
  };

  const sortedLoans = useMemo(() => {
    if (!loans || loans.length === 0) return [];

    const sorted = [...loans].sort((a, b) => {
      let aVal = a[sortConfig.key];
      let bVal = b[sortConfig.key];

      // Handle null/undefined values
      if (aVal === null || aVal === undefined) return 1;
      if (bVal === null || bVal === undefined) return -1;

      // Numeric comparison
      if (typeof aVal === 'number' && typeof bVal === 'number') {
        return sortConfig.direction === 'asc' ? aVal - bVal : bVal - aVal;
      }

      // String comparison
      const aStr = String(aVal).toLowerCase();
      const bStr = String(bVal).toLowerCase();
      if (aStr < bStr) return sortConfig.direction === 'asc' ? -1 : 1;
      if (aStr > bStr) return sortConfig.direction === 'asc' ? 1 : -1;
      return 0;
    });

    return sorted;
  }, [loans, sortConfig]);

  const formatCurrency = (value) => {
    return new Intl.NumberFormat('en-NG', {
      style: 'currency',
      currency: 'NGN',
      minimumFractionDigits: 0,
    }).format(value);
  };

  const formatDate = (dateString) => {
    if (!dateString) return 'N/A';
    const date = new Date(dateString);
    return date.toLocaleDateString('en-GB', { day: '2-digit', month: 'short', year: 'numeric' });
  };

  const getRiskCategoryColor = (category) => {
    switch (category) {
      case 'Critical': return 'risk-critical';
      case 'High': return 'risk-high';
      case 'Medium': return 'risk-medium';
      case 'Low': return 'risk-low';
      default: return 'risk-low';
    }
  };

  const handleExportCSV = () => {
    const headers = [
      'Loan ID', 'Customer Name', 'Phone', 'Loan Amount', 'Disbursement Date',
      'Current DPD', 'Max DPD Ever', 'Total Outstanding', 'Principal Outstanding',
      'Interest Outstanding', 'Fees Outstanding', 'Status', 'FIMR Tagged',
      'Risk Score', 'Risk Category', 'Channel', 'Days Since Disbursement'
    ];

    const rows = sortedLoans.map(loan => [
      loan.loan_id,
      loan.customer_name,
      loan.customer_phone || '',
      loan.loan_amount,
      loan.disbursement_date,
      loan.current_dpd,
      loan.max_dpd_ever,
      loan.total_outstanding,
      loan.principal_outstanding,
      loan.interest_outstanding,
      loan.fees_outstanding,
      loan.status,
      loan.fimr_tagged ? 'Yes' : 'No',
      loan.risk_score.toFixed(2),
      loan.risk_category,
      loan.channel,
      loan.days_since_disbursement
    ]);

    const csvContent = [
      `Top 20 Risk Loans - ${officerName}`,
      `Generated: ${new Date().toLocaleString()}`,
      '',
      headers.join(','),
      ...rows.map(row => row.map(cell => `"${cell}"`).join(','))
    ].join('\n');

    const blob = new Blob([csvContent], { type: 'text/csv;charset=utf-8;' });
    const link = document.createElement('a');
    const url = URL.createObjectURL(blob);
    link.setAttribute('href', url);
    link.setAttribute('download', `Top_Risk_Loans_${officerName.replace(/\s+/g, '_')}_${new Date().toISOString().split('T')[0]}.csv`);
    link.style.visibility = 'hidden';
    document.body.appendChild(link);
    link.click();
    document.body.removeChild(link);
  };

  const handleExportPDF = () => {
    const doc = new jsPDF('landscape');

    // Title
    doc.setFontSize(16);
    doc.text(`Top 20 Risk Loans - ${officerName}`, 14, 15);
    doc.setFontSize(10);
    doc.text(`Generated: ${new Date().toLocaleString()}`, 14, 22);

    // Table
    const headers = [
      ['Loan ID', 'Customer', 'Amount', 'Disb. Date', 'DPD', 'Max DPD', 'Outstanding', 'Risk Score', 'Risk Category', 'FIMR']
    ];

    const data = sortedLoans.map(loan => [
      loan.loan_id,
      loan.customer_name,
      formatCurrency(loan.loan_amount),
      formatDate(loan.disbursement_date),
      loan.current_dpd,
      loan.max_dpd_ever,
      formatCurrency(loan.total_outstanding),
      loan.risk_score.toFixed(2),
      loan.risk_category,
      loan.fimr_tagged ? 'Yes' : 'No'
    ]);

    doc.autoTable({
      head: headers,
      body: data,
      startY: 28,
      styles: { fontSize: 8 },
      headStyles: { fillColor: [239, 68, 68] },
    });

    doc.save(`Top_Risk_Loans_${officerName.replace(/\s+/g, '_')}_${new Date().toISOString().split('T')[0]}.pdf`);
  };

  if (!isOpen) return null;

  return (
    <div className="modal-overlay" onClick={onClose}>
      <div className="modal-content top-risk-modal" onClick={(e) => e.stopPropagation()}>
        <div className="modal-header">
          <div className="modal-title">
            <AlertTriangle size={24} className="warning-icon" />
            <div>
              <h2>Top 20 Risk Loans - Audit</h2>
              <p className="officer-name">{officerName}</p>
            </div>
          </div>
          <button className="close-button" onClick={onClose}>
            <X size={24} />
          </button>
        </div>

        <div className="modal-actions">
          <div className="loan-count">
            {loans.length} {loans.length === 1 ? 'Loan' : 'Loans'} Found
          </div>
          <div className="export-buttons">
            <button className="export-button" onClick={handleExportCSV} disabled={loading || loans.length === 0}>
              <Download size={16} />
              Export CSV
            </button>
            <button className="export-button" onClick={handleExportPDF} disabled={loading || loans.length === 0}>
              <FileText size={16} />
              Export PDF
            </button>
          </div>
        </div>

        <div className="modal-body">
          {loading && (
            <div className="loading-state">
              <div className="spinner"></div>
              <p>Loading top risk loans...</p>
            </div>
          )}

          {error && (
            <div className="error-state">
              <AlertTriangle size={48} />
              <p>{error}</p>
              <button onClick={fetchTopRiskLoans}>Retry</button>
            </div>
          )}

          {!loading && !error && loans.length === 0 && (
            <div className="empty-state">
              <p>No high-risk loans found for this officer.</p>
            </div>
          )}

          {!loading && !error && loans.length > 0 && (
            <div className="table-container">
              <table className="risk-loans-table">
                <thead>
                  <tr>
                    <th onClick={() => handleSort('risk_score')}>Risk Score</th>
                    <th onClick={() => handleSort('risk_category')}>Category</th>
                    <th onClick={() => handleSort('loan_id')}>Loan ID</th>
                    <th onClick={() => handleSort('customer_name')}>Customer</th>
                    <th onClick={() => handleSort('loan_amount')}>Loan Amount</th>
                    <th onClick={() => handleSort('current_dpd')}>DPD</th>
                    <th onClick={() => handleSort('max_dpd_ever')}>Max DPD</th>
                    <th onClick={() => handleSort('total_outstanding')}>Outstanding</th>
                    <th onClick={() => handleSort('fimr_tagged')}>FIMR</th>
                    <th onClick={() => handleSort('disbursement_date')}>Disb. Date</th>
                    <th onClick={() => handleSort('days_since_disbursement')}>Age (Days)</th>
                  </tr>
                </thead>
                <tbody>
                  {sortedLoans.map((loan) => (
                    <tr key={loan.loan_id}>
                      <td className="risk-score">{loan.risk_score.toFixed(2)}</td>
                      <td>
                        <span className={`risk-badge ${getRiskCategoryColor(loan.risk_category)}`}>
                          {loan.risk_category}
                        </span>
                      </td>
                      <td className="loan-id">{loan.loan_id}</td>
                      <td>{loan.customer_name}</td>
                      <td className="amount">{formatCurrency(loan.loan_amount)}</td>
                      <td className="dpd">{loan.current_dpd}</td>
                      <td className="dpd">{loan.max_dpd_ever}</td>
                      <td className="amount">{formatCurrency(loan.total_outstanding)}</td>
                      <td className="fimr-cell">
                        {loan.fimr_tagged ? (
                          <span className="fimr-badge fimr-yes">Yes</span>
                        ) : (
                          <span className="fimr-badge fimr-no">No</span>
                        )}
                      </td>
                      <td>{formatDate(loan.disbursement_date)}</td>
                      <td className="age">{loan.days_since_disbursement}</td>
                    </tr>
                  ))}
                </tbody>
              </table>
            </div>
          )}
        </div>
      </div>
    </div>
  );
};

export default TopRiskLoansModal;

