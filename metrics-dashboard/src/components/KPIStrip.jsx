import React from 'react';
import './KPIStrip.css';

const formatCurrency = (value) => {
  if (value >= 1000000) return `₦${(value / 1000000).toFixed(1)}M`;
  if (value >= 1000) return `₦${(value / 1000).toFixed(1)}K`;
  return `₦${value}`;
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
  return (
    <div className="kpi-strip">
      <KPICard
        title="Portfolio Overdue >15 Days"
        value={formatCurrency(portfolioMetrics.totalOverdue15d)}
        trend={{ direction: 'down', value: '3% WoW' }}
        icon="📊"
      />
      <KPICard
        title="Average DQI"
        value={portfolioMetrics.avgDQI}
        unit="✅"
        trend={{ direction: 'up', value: '2pts' }}
      />
      <KPICard
        title="Average AYR"
        value={portfolioMetrics.avgAYR}
        unit="⚖️ Efficient"
      />
      <KPICard
        title="Risk Score (Avg)"
        value={portfolioMetrics.avgRiskScore}
        unit="🟩"
      />
      <KPICard
        title="Top Performing Officer"
        value={portfolioMetrics.topOfficer.name}
        unit={`AYR ${portfolioMetrics.topOfficer.ayr.toFixed(2)}`}
      />
      <KPICard
        title="Watchlist Count"
        value={portfolioMetrics.watchlistCount}
        unit={`Officers / ${formatCurrency(portfolioMetrics.totalOverdue15d / 10)}`}
      />
    </div>
  );
};

