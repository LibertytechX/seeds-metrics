import React, { useMemo, useState } from 'react';
import { X } from 'lucide-react';
import './RepaymentWatchModal.css';

const RepaymentWatchModal = ({ isOpen, onClose, data, loading, error, onRefresh }) => {
  const [activeTab, setActiveTab] = useState('zero');

  const buckets = useMemo(() => {
    const initial = { zero: [], lt20: [], lt60: [], gte60: [] };
    if (!Array.isArray(data)) return initial;

    data.forEach((raw) => {
      const totalLoans =
        typeof raw.total_wave2_open_loans === 'number'
          ? raw.total_wave2_open_loans
          : raw.totalWave2OpenLoans || 0;
      const loansWithRepayment =
        typeof raw.loans_with_repayment_today === 'number'
          ? raw.loans_with_repayment_today
          : raw.loansWithRepaymentToday || 0;
      const amountCollected =
        typeof raw.amount_collected_today === 'number'
          ? raw.amount_collected_today
          : raw.amountCollectedToday || 0;

      let rate =
        typeof raw.repayment_rate === 'number'
          ? raw.repayment_rate
          : typeof raw.repaymentRate === 'number'
          ? raw.repaymentRate
          : 0;

      if (!rate && totalLoans > 0 && loansWithRepayment >= 0) {
        rate = (loansWithRepayment / totalLoans) * 100;
      }

      const officer = {
        officer_id: raw.officer_id || raw.officerId,
        officer_name: raw.officer_name || raw.officerName,
        officer_email: raw.officer_email || raw.officerEmail,
        branch: raw.branch,
        region: raw.region,
        total_wave2_open_loans: totalLoans,
        loans_with_repayment_today: loansWithRepayment,
        amount_collected_today: amountCollected,
        repayment_rate: rate || 0,
      };

      if (officer.total_wave2_open_loans <= 0) {
        initial.zero.push(officer);
        return;
      }

      if (officer.repayment_rate === 0) {
        initial.zero.push(officer);
      } else if (officer.repayment_rate > 0 && officer.repayment_rate < 20) {
        initial.lt20.push(officer);
      } else if (officer.repayment_rate >= 20 && officer.repayment_rate < 60) {
        initial.lt60.push(officer);
      } else {
        initial.gte60.push(officer);
      }
    });

    const sortAsc = (arr) =>
      arr.sort(
        (a, b) =>
          a.repayment_rate - b.repayment_rate ||
          b.total_wave2_open_loans - a.total_wave2_open_loans,
      );
    const sortDesc = (arr) =>
      arr.sort(
        (a, b) =>
          b.repayment_rate - a.repayment_rate ||
          b.total_wave2_open_loans - a.total_wave2_open_loans,
      );

    initial.zero.sort((a, b) => b.total_wave2_open_loans - a.total_wave2_open_loans);
    sortAsc(initial.lt20);
    sortAsc(initial.lt60);
    sortDesc(initial.gte60);

    return initial;
  }, [data]);

  const tabs = [
    {
      key: 'zero',
      label: 'Zero',
      description: '0% of Wave 2 open loans have any repayment today',
      badgeClass: 'badge-red',
    },
    {
      key: 'lt20',
      label: '<20%',
      description: 'Some activity, but less than 20% of loans have repaid today',
      badgeClass: 'badge-amber',
    },
    {
      key: 'lt60',
      label: '<60%',
      description: 'Decent performance, 20-60% of loans have repaid today',
      badgeClass: 'badge-yellow',
    },
    {
      key: 'gte60',
      label: '60%+',
      description: 'Strong performance, at least 60% of loans have repaid today',
      badgeClass: 'badge-green',
    },
  ];

  const currentItems = buckets[activeTab] || [];

  const formatCurrency = (value) => {
    const safe = typeof value === 'number' ? value : 0;
    return new Intl.NumberFormat('en-NG', {
      style: 'currency',
      currency: 'NGN',
      minimumFractionDigits: 0,
      maximumFractionDigits: 0,
    }).format(safe);
  };

  const formatPercent = (value) => {
    if (typeof value !== 'number' || Number.isNaN(value)) return '0%';
    return `${value.toFixed(1)}%`;
  };

  if (!isOpen) return null;

  return (
    <div className="repayment-watch-overlay" onClick={onClose}>
      <div className="repayment-watch-modal" onClick={(e) => e.stopPropagation()}>
        <div className="repayment-watch-header">
          <div className="repayment-watch-title">
            <h2>Repayment Watch - Wave 2 (Today)</h2>
            <p>
              Monitor Wave 2 officers by today&apos;s repayment performance. Only
              non-reversed repayments made today are counted.
            </p>
          </div>
          <button
            type="button"
            className="repayment-watch-close"
            onClick={onClose}
          >
            <X size={18} />
          </button>
        </div>

        <div className="repayment-watch-tabs">
          {tabs.map((tab) => (
            <button
              key={tab.key}
              type="button"
              className={`repayment-watch-tab ${activeTab === tab.key ? 'active' : ''}`}
              onClick={() => setActiveTab(tab.key)}
            >
              <span className={`repayment-watch-tab-badge ${tab.badgeClass}`}>
                {tab.label}
              </span>
              <span className="repayment-watch-tab-count">
                {(buckets[tab.key] || []).length}
              </span>
            </button>
          ))}
        </div>

        <div className="repayment-watch-tab-description">
          {tabs.find((t) => t.key === activeTab)?.description}
        </div>

        <div className="repayment-watch-body">
          {loading && (
            <div className="repayment-watch-state">
              <div className="repayment-watch-spinner" />
              <p>Loading Repayment Watch data...</p>
            </div>
          )}

          {!loading && error && (
            <div className="repayment-watch-state error">
              <p>{error}</p>
              {onRefresh && (
                <button type="button" onClick={onRefresh}>
                  Retry
                </button>
              )}
            </div>
          )}

          {!loading && !error && currentItems.length === 0 && (
            <div className="repayment-watch-state">
              <p>No officers found for this category with the current filters.</p>
            </div>
          )}

          {!loading && !error && currentItems.length > 0 && (
            <div className="repayment-watch-table-wrapper">
              <table className="repayment-watch-table">
                <thead>
                  <tr>
                    <th>Officer</th>
                    <th>Branch</th>
                    <th>Region</th>
                    <th>Wave 2 Open Loans</th>
                    <th>Loans With Repayment Today</th>
                    <th>Amount Collected Today</th>
                    <th>Repayment Rate</th>
                  </tr>
                </thead>
                <tbody>
                  {currentItems.map((officer) => (
                    <tr key={officer.officer_id || officer.officer_name}>
                      <td>
                        <div className="repayment-watch-officer">
                          <span className="officer-name">
                            {officer.officer_name || 'Unknown officer'}
                          </span>
                          {officer.officer_email && (
                            <span className="officer-email">
                              {officer.officer_email}
                            </span>
                          )}
                        </div>
                      </td>
                      <td>{officer.branch || 'All'}</td>
                      <td>{officer.region || 'All'}</td>
                      <td>{officer.total_wave2_open_loans?.toLocaleString() ?? '0'}</td>
                      <td>
                        {officer.loans_with_repayment_today?.toLocaleString() ?? '0'}
                      </td>
                      <td>{formatCurrency(officer.amount_collected_today)}</td>
                      <td>{formatPercent(officer.repayment_rate)}</td>
                    </tr>
                  ))}
                </tbody>
              </table>
            </div>
          )}
        </div>

        {onRefresh && !loading && !error && (
          <div className="repayment-watch-footer">
            <button type="button" onClick={onRefresh}>
              Refresh data
            </button>
          </div>
        )}
      </div>
    </div>
  );
};

export default RepaymentWatchModal;
