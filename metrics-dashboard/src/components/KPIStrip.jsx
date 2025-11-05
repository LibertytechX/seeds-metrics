import React from 'react';
import './KPIStrip.css';

const formatCurrency = (value) => {
  if (value >= 1000000) return `₦${(value / 1000000).toFixed(1)}M`;
  if (value >= 1000) return `₦${(value / 1000).toFixed(1)}K`;
  return `₦${value.toFixed(2)}`;
};

const formatPercentage = (value) => {
  return `${(value * 100).toFixed(2)}%`;
};

const KPICard = ({ title, value, unit = '', trend = null, icon = null, buttons = null }) => {
  return (
    <div className="kpi-card">
      <div className="kpi-header">
        <h3>{title}</h3>
        {icon && <span className="kpi-icon">{icon}</span>}
      </div>
      <div className="kpi-value">{value}</div>
      {unit && <div className="kpi-unit">{unit}</div>}
      {trend && (
        <div className={`kpi-trend ${trend.direction}`}>
          {trend.direction === 'up' ? '↑' : '↓'} {trend.value}
        </div>
      )}
      {buttons && (
        <div className="kpi-buttons">
          {buttons}
        </div>
      )}
    </div>
  );
};

export const KPIStrip = ({ portfolioMetrics, onViewOverdueLoans, onViewActiveLoans, onViewInactiveLoans, onViewEarlyROT, onViewLateROT, onViewAtRiskOfficers, onViewLowDelayOfficers }) => {
  // Handle null topOfficer gracefully
  const topOfficerName = portfolioMetrics.topOfficer?.name || 'N/A';
  const topOfficerAYR = portfolioMetrics.topOfficer?.ayr || 0;

  return (
    <div className="kpi-strip">
      <KPICard
        title="Portfolio Overdue >15 Days"
        value={
          <div style={{ fontSize: '0.75em', lineHeight: '1.3' }}>
            <div><strong>Total Outstanding:</strong> {formatCurrency(portfolioMetrics.totalOverdue15d || 0)}</div>
            <div><strong>Actual Outstanding (Due to Date):</strong> {formatCurrency(portfolioMetrics.actualOverdue15d || 0)}</div>
          </div>
        }
        icon="📊"
        buttons={
          <button className="kpi-btn" onClick={onViewOverdueLoans}>View Overdue Loans</button>
        }
      />

      {/* REMOVED: Average DQI Card */}

      {/* NEW CARD 1: Active vs Inactive Loans */}
      <KPICard
        title="Active vs Inactive Loans"
        value={
          <div style={{ fontSize: '0.75em', lineHeight: '1.3' }}>
            <div><strong>Active:</strong> {portfolioMetrics.activeLoansCount || 0} loans ({formatCurrency(portfolioMetrics.activeLoansVolume || 0)})</div>
            <div><strong>Inactive:</strong> {portfolioMetrics.inactiveLoansCount || 0} loans ({formatCurrency(portfolioMetrics.inactiveLoansVolume || 0)})</div>
          </div>
        }
        icon="📈"
        buttons={
          <>
            <button className="kpi-btn" onClick={onViewActiveLoans}>View Active</button>
            <button className="kpi-btn" onClick={onViewInactiveLoans}>View Inactive</button>
          </>
        }
      />

      {/* NEW CARD 2: ROT Analysis */}
      <KPICard
        title="ROT (Risk of Termination)"
        value={
          <div style={{ fontSize: '0.75em', lineHeight: '1.3' }}>
            <div><strong>Early ROT:</strong> {portfolioMetrics.earlyROTCount || 0} loans ({formatCurrency(portfolioMetrics.earlyROTVolume || 0)})</div>
            <div><strong>Late ROT:</strong> {portfolioMetrics.lateROTCount || 0} loans ({formatCurrency(portfolioMetrics.lateROTVolume || 0)})</div>
          </div>
        }
        icon="⚠️"
        buttons={
          <>
            <button className="kpi-btn" onClick={onViewEarlyROT}>View Early ROT</button>
            <button className="kpi-btn" onClick={onViewLateROT}>View Late ROT</button>
          </>
        }
      />

      {/* NEW CARD 3: Portfolio Delinquency Risk */}
      <KPICard
        title="Portfolio Delinquency Risk"
        value={`${(portfolioMetrics.atRiskOfficersPercentage || 0).toFixed(2)}%`}
        unit={`${portfolioMetrics.atRiskOfficersCount || 0} of ${portfolioMetrics.totalOfficers || 0} officers at risk`}
        icon="🚨"
        buttons={
          <button className="kpi-btn" onClick={onViewAtRiskOfficers}>View At-Risk Officers</button>
        }
      />

      {/* NEW CARD 4: Portfolio Repayment Behavior */}
      <KPICard
        title="Portfolio Repayment Behavior"
        value={
          <div style={{ fontSize: '0.75em', lineHeight: '1.3' }}>
            <div><strong>Avg DPD:</strong> {(portfolioMetrics.avgDaysPastDue || 0).toFixed(1)} days</div>
            <div><strong>Avg Timeliness:</strong> {(portfolioMetrics.avgTimelinessScore || 0).toFixed(2)}</div>
            <div><strong>Avg Delay Rate:</strong> {(portfolioMetrics.avgRepaymentDelayRate || 0).toFixed(2)}%</div>
          </div>
        }
        icon="📊"
        buttons={
          <button className="kpi-btn" onClick={onViewLowDelayOfficers}>View Officers</button>
        }
      />

      <KPICard
        title="Average AYR"
        value={formatPercentage(portfolioMetrics.avgAYR || 0)}
        unit="⚖️ Efficient"
      />
      <KPICard
        title="Risk Score (Avg)"
        value={portfolioMetrics.avgRiskScore || 0}
        unit="🟩"
      />
      <KPICard
        title="Top Performing Officer"
        value={topOfficerName}
        unit={topOfficerName !== 'N/A' ? `AYR ${formatPercentage(topOfficerAYR)}` : 'No data'}
      />
      <KPICard
        title="Watchlist Count"
        value={portfolioMetrics.watchlistCount || 0}
        unit={`Officers / ${formatCurrency(portfolioMetrics.watchlistPortfolio || 0)} Portfolio`}
      />
    </div>
  );
};

