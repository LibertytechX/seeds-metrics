import React, { useState } from 'react';
import { getBand } from '../utils/metrics';
import { formatMetricTooltip } from '../utils/metricInfo';
import { MetricHeader } from './Tooltip';
import './DataTables.css';

const BandBadge = ({ band }) => {
  const colors = {
    green: '#10b981',
    yellow: '#f59e0b',
    red: '#ef4444',
    amber: '#f97316',
  };
  return (
    <span
      className="band-badge"
      style={{ backgroundColor: colors[band.color] }}
    >
      {band.label}
    </span>
  );
};

const OfficerPerformanceTable = ({ officers }) => {
  const [sortBy, setSortBy] = useState('riskScore');
  const [sortDir, setSortDir] = useState('asc');

  const sorted = [...officers].sort((a, b) => {
    const aVal = a[sortBy];
    const bVal = b[sortBy];
    return sortDir === 'asc' ? aVal - bVal : bVal - aVal;
  });

  const handleSort = (column) => {
    if (sortBy === column) {
      setSortDir(sortDir === 'asc' ? 'desc' : 'asc');
    } else {
      setSortBy(column);
      setSortDir('asc');
    }
  };

  return (
    <div className="table-container">
      <table className="data-table">
        <thead>
          <tr>
            <th onClick={() => handleSort('name')}>Officer</th>
            <th onClick={() => handleSort('region')}>Region</th>
            <th onClick={() => handleSort('ayr')}>
              <MetricHeader label="AYR" metricKey="ayr" info={formatMetricTooltip('ayr')} />
            </th>
            <th onClick={() => handleSort('dqi')}>
              <MetricHeader label="DQI" metricKey="dqi" info={formatMetricTooltip('dqi')} />
            </th>
            <th onClick={() => handleSort('riskScore')}>
              <MetricHeader label="Risk Score" metricKey="riskScore" info={formatMetricTooltip('riskScore')} />
            </th>
            <th onClick={() => handleSort('overdue15dVolume')}>
              <MetricHeader label="Overdue &gt;15D" metricKey="overdue15d" info={formatMetricTooltip('overdue15d')} />
            </th>
            <th onClick={() => handleSort('yield')}>
              <MetricHeader label="Yield" metricKey="yield" info={formatMetricTooltip('yield')} />
            </th>
            <th>Band</th>
            <th>Actions</th>
          </tr>
        </thead>
        <tbody>
          {sorted.map((officer) => {
            const riskBand = getBand(officer.riskScore, 'riskScore');
            return (
              <tr key={officer.id}>
                <td>{officer.name}</td>
                <td>{officer.region}</td>
                <td>{officer.ayr.toFixed(2)}</td>
                <td>{officer.dqi}</td>
                <td>{officer.riskScore}</td>
                <td>₦{(officer.overdue15dVolume / 1000000).toFixed(1)}M</td>
                <td>₦{(officer.yield / 1000000).toFixed(1)}M</td>
                <td>
                  <BandBadge band={riskBand} />
                </td>
                <td>
                  <button className="btn-action">View</button>
                </td>
              </tr>
            );
          })}
        </tbody>
      </table>
    </div>
  );
};

const CreditHealthTable = ({ officers }) => {
  return (
    <div className="table-container">
      <table className="data-table">
        <thead>
          <tr>
            <th>Officer</th>
            <th>Portfolio Total</th>
            <th>
              <MetricHeader label="Overdue &gt;15D" metricKey="overdue15d" info={formatMetricTooltip('overdue15d')} />
            </th>
            <th>
              <MetricHeader label="AYR" metricKey="ayr" info={formatMetricTooltip('ayr')} />
            </th>
            <th>
              <MetricHeader label="DQI" metricKey="dqi" info={formatMetricTooltip('dqi')} />
            </th>
            <th>
              <MetricHeader label="FIMR" metricKey="fimr" info={formatMetricTooltip('fimr')} />
            </th>
            <th>Actions</th>
          </tr>
        </thead>
        <tbody>
          {officers.map((officer) => (
            <tr key={officer.id}>
              <td>{officer.name}</td>
              <td>₦{(officer.totalPortfolio / 1000000).toFixed(1)}M</td>
              <td>₦{(officer.overdue15d / 1000000).toFixed(1)}M</td>
              <td>{officer.ayr.toFixed(2)}</td>
              <td>{officer.dqi}</td>
              <td>{(officer.fimr * 100).toFixed(1)}%</td>
              <td>
                <button className="btn-action">Drill Down</button>
              </td>
            </tr>
          ))}
        </tbody>
      </table>
    </div>
  );
};

const EarlyIndicatorsTable = ({ officers }) => {
  return (
    <div className="table-container">
      <table className="data-table">
        <thead>
          <tr>
            <th>Officer</th>
            <th>
              <MetricHeader label="FIMR" metricKey="fimr" info={formatMetricTooltip('fimr')} />
            </th>
            <th>
              <MetricHeader label="D0-6 Slippage" metricKey="slippage" info={formatMetricTooltip('slippage')} />
            </th>
            <th>
              <MetricHeader label="Roll" metricKey="roll" info={formatMetricTooltip('roll')} />
            </th>
            <th>
              <MetricHeader label="FRR" metricKey="frr" info={formatMetricTooltip('frr')} />
            </th>
            <th>
              <MetricHeader label="Channel Purity" metricKey="channelPurity" info={formatMetricTooltip('channelPurity')} />
            </th>
            <th>Actions</th>
          </tr>
        </thead>
        <tbody>
          {officers.map((officer) => (
            <tr key={officer.id}>
              <td>{officer.name}</td>
              <td>
                <BandBadge band={getBand(officer.fimr, 'fimr')} />
                {(officer.fimr * 100).toFixed(1)}%
              </td>
              <td>
                <BandBadge band={getBand(officer.slippage, 'slippage')} />
                {(officer.slippage * 100).toFixed(1)}%
              </td>
              <td>
                <BandBadge band={getBand(officer.roll, 'roll')} />
                {(officer.roll * 100).toFixed(1)}%
              </td>
              <td>{(officer.frr * 100).toFixed(1)}%</td>
              <td>{(officer.channelPurity * 100).toFixed(1)}%</td>
              <td>
                <button className="btn-action">View Trends</button>
              </td>
            </tr>
          ))}
        </tbody>
      </table>
    </div>
  );
};

export const DataTables = ({ officers, activeTab }) => {
  const tabs = {
    creditHealth: <CreditHealthTable officers={officers} />,
    performance: <OfficerPerformanceTable officers={officers} />,
    earlyIndicators: <EarlyIndicatorsTable officers={officers} />,
  };

  return tabs[activeTab] || tabs.performance;
};

