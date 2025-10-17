import React, { useState } from 'react';
import { Info } from 'lucide-react';
import './Tooltip.css';

/**
 * Tooltip Component
 * Shows detailed information on hover
 */
export const Tooltip = ({ children, content, position = 'top' }) => {
  const [isVisible, setIsVisible] = useState(false);

  if (!content) {
    return children;
  }

  return (
    <div className="tooltip-wrapper">
      <div
        className="tooltip-trigger"
        onMouseEnter={() => setIsVisible(true)}
        onMouseLeave={() => setIsVisible(false)}
      >
        {children}
      </div>
      {isVisible && (
        <div className={`tooltip-content tooltip-${position}`}>
          {content}
        </div>
      )}
    </div>
  );
};

/**
 * Metric Header with Tooltip
 */
export const MetricHeader = ({ label, metricKey, info }) => {
  return (
    <div className="metric-header">
      <span>{label}</span>
      {info && (
        <Tooltip content={info} position="right">
          <Info size={16} className="info-icon" />
        </Tooltip>
      )}
    </div>
  );
};

/**
 * Tab Header with Tooltip
 */
export const TabHeader = ({ label, tabKey, info }) => {
  return (
    <div className="tab-header">
      <span>{label}</span>
      {info && (
        <Tooltip content={info} position="bottom">
          <Info size={14} className="info-icon-small" />
        </Tooltip>
      )}
    </div>
  );
};

