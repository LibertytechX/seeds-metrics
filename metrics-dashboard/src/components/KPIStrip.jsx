import React from 'react';
import './KPIStrip.css';

const formatCurrency = (value) => {
  if (value >= 1000000) return `₦${(value / 1000000).toFixed(1)}M`;
  if (value >= 1000) return `₦${(value / 1000).toFixed(1)}K`;
  return `₦${value}`;
};

const formatPercentage = (value) => {
  return `${(value * 100).toFixed(2)}%`;
};

const KPICard = ({ title, value, unit = '', trend = null, icon = null }) => {
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
    </div>
  );
};

export const KPIStrip = ({ portfolioMetrics }) => {
  // Handle null topOfficer gracefully
  const topOfficerName = portfolioMetrics.topOfficer?.name || 'N/A';
  const topOfficerAYR = portfolioMetrics.topOfficer?.ayr || 0;

  return (
    <div className="kpi-strip">
      <KPICard
        title="Portfolio Overdue >15 Days"
        value={formatCurrency(portfolioMetrics.totalOverdue15d || 0)}
        trend={{ direction: 'down', value: '3% WoW' }}
        icon="📊"
      />
      <KPICard
        title="Average DQI"
        value={portfolioMetrics.avgDQI || 0}
        unit="✅"
        trend={{ direction: 'up', value: '2pts' }}
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
        unit={`Officers / ${formatCurrency((portfolioMetrics.totalOverdue15d || 0) / 10)}`}
      />
    </div>
  );
};

