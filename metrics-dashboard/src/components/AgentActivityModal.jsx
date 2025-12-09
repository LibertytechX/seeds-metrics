import React, { useMemo } from 'react';
import { X } from 'lucide-react';
import './AgentActivityModal.css';

const CATEGORY_CONFIG = {
  critical_no_collection: {
    title: 'CRITICAL: Officers with NO COLLECTION',
    description: 'Officers with no repayments recorded in the last 7 days.',
  },
  stopped_collecting: {
    title: 'Officers STOPPED Collecting',
    description:
      'Were active in the first 4 days of the week but made zero collections in the last 3 days.',
  },
  severe_decline: {
    title: 'Officers in SEVERE DECLINE',
    description:
      'Collections in the last 3 days are less than 30% of the first 4 days of the week.',
  },
  not_yet_started_today: {
    title: 'Officers NOT YET STARTED Today',
    description: 'Collected on at least 5 of the last 7 days but have not collected today.',
  },
  strong_growth: {
    title: 'Officers with STRONG GROWTH',
    description:
      'Collections in the last 3 days are more than 150% of the first 4 days of the week.',
  },
  started_today: {
    title: 'Officers STARTED Today',
    description: 'Have collections recorded today (including new starters).',
  },
};

	const AgentActivityModal = ({
		isOpen,
		onClose,
		categoryKey,
		data,
		loading,
		error,
		onRefresh,
		regionFilter,
		onRegionChange,
		availableRegions = [],
		waveFilter,
		onWaveChange,
		availableWaves = [],
	}) => {
  const categoryMeta = CATEGORY_CONFIG[categoryKey] || {
    title: 'Agent Activity Detail',
    description: 'Per-officer 7-day activity for the selected category.',
  };

  const rows = useMemo(() => {
    if (!Array.isArray(data)) return [];
    return data.map((raw) => ({
      officerId: raw.officer_id || raw.officerId,
      officerName: raw.officer_name || raw.officerName,
      officerEmail: raw.officer_email || raw.officerEmail,
      branch: raw.branch || 'All',
      region: raw.region || 'All',
      repaymentRate:
        typeof raw.repayment_rate === 'number'
          ? raw.repayment_rate
          : typeof raw.repaymentRate === 'number'
          ? raw.repaymentRate
          : 0,
      amount5dAgo:
        typeof raw.amount_5_days_ago === 'number'
          ? raw.amount_5_days_ago
          : typeof raw.amount5DaysAgo === 'number'
          ? raw.amount5DaysAgo
          : 0,
      amount4dAgo:
        typeof raw.amount_4_days_ago === 'number'
          ? raw.amount_4_days_ago
          : typeof raw.amount4DaysAgo === 'number'
          ? raw.amount4DaysAgo
          : 0,
      amount3dAgo:
        typeof raw.amount_3_days_ago === 'number'
          ? raw.amount_3_days_ago
          : typeof raw.amount3DaysAgo === 'number'
          ? raw.amount3DaysAgo
          : 0,
      amount2dAgo:
        typeof raw.amount_2_days_ago === 'number'
          ? raw.amount_2_days_ago
          : typeof raw.amount2DaysAgo === 'number'
          ? raw.amount2DaysAgo
          : 0,
      amount1dAgo:
        typeof raw.amount_1_day_ago === 'number'
          ? raw.amount_1_day_ago
          : typeof raw.amount1DayAgo === 'number'
          ? raw.amount1DayAgo
          : 0,
      amountToday:
        typeof raw.amount_today === 'number'
          ? raw.amount_today
          : typeof raw.amountToday === 'number'
          ? raw.amountToday
          : 0,
      totalCollected:
        typeof raw.total_collected === 'number'
          ? raw.total_collected
          : typeof raw.totalCollected === 'number'
          ? raw.totalCollected
          : 0,
    }));
  }, [data]);

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

	  const hasRows = rows.length > 0;
	  const canExport = hasRows && !loading && !error;

	  const escapeCsvValue = (value) => {
	    if (value === null || value === undefined) return '';
	    const str = String(value);
	    if (/[",\n\r]/.test(str)) {
	      return `"${str.replace(/"/g, '""')}"`;
	    }
	    return str;
	  };

	  const getCategoryLabelForFilename = () => {
	    if (categoryKey && CATEGORY_CONFIG[categoryKey]) {
	      return categoryKey
	        .split('_')
	        .map((part) => part.charAt(0).toUpperCase() + part.slice(1))
	        .join('_');
	    }
	    const raw = categoryMeta.title || 'Agent Activity Detail';
	    const cleaned = raw.replace(/[^A-Za-z0-9]+/g, '_').replace(/^_+|_+$/g, '');
	    return cleaned || 'Agent_Activity_Detail';
	  };

	  const handleExportClick = () => {
	    if (!canExport) return;

	    const headers = [
	      'Officer Name',
	      'Officer Email',
	      'Branch',
	      'Region',
	      'Repayment Rate (%)',
	      'Amount 5 Days Ago',
	      'Amount 4 Days Ago',
	      'Amount 3 Days Ago',
	      'Amount 2 Days Ago',
	      'Amount 1 Day Ago',
	      'Amount Today',
	      'Total Collected',
	    ];

	    const dataRows = rows.map((row) => {
	      const repaymentRateValue =
	        typeof row.repaymentRate === 'number' && !Number.isNaN(row.repaymentRate)
	          ? row.repaymentRate.toFixed(1)
	          : '0.0';

	      return [
	        escapeCsvValue(row.officerName || 'Unknown officer'),
	        escapeCsvValue(row.officerEmail || ''),
	        escapeCsvValue(row.branch || ''),
	        escapeCsvValue(row.region || ''),
	        escapeCsvValue(repaymentRateValue),
	        escapeCsvValue(row.amount5dAgo ?? 0),
	        escapeCsvValue(row.amount4dAgo ?? 0),
	        escapeCsvValue(row.amount3dAgo ?? 0),
	        escapeCsvValue(row.amount2dAgo ?? 0),
	        escapeCsvValue(row.amount1dAgo ?? 0),
	        escapeCsvValue(row.amountToday ?? 0),
	        escapeCsvValue(row.totalCollected ?? 0),
	      ].join(',');
	    });

	    const csvLines = [headers.join(','), ...dataRows];
	    const csvContent = csvLines.join('\r\n');

	    const categoryLabel = getCategoryLabelForFilename();
	    const datePart = new Date().toISOString().slice(0, 10);
	    const filename = `Agent_Activity_${categoryLabel}_${datePart}.csv`;

	    const blob = new Blob([csvContent], { type: 'text/csv;charset=utf-8;' });
	    const url = URL.createObjectURL(blob);
	    const link = document.createElement('a');
	    link.href = url;
	    link.setAttribute('download', filename);
	    document.body.appendChild(link);
	    link.click();
	    document.body.removeChild(link);
	    URL.revokeObjectURL(url);
	  };

		return (
		<div className="agent-activity-overlay" onClick={onClose}>
		  <div className="agent-activity-modal" onClick={(e) => e.stopPropagation()}>
				<div className="agent-activity-header">
				  <div className="agent-activity-header-main">
					<div className="agent-activity-title">
					  <h2>{categoryMeta.title}</h2>
					  <p>{categoryMeta.description}</p>
					</div>
					{(Array.isArray(availableRegions) && availableRegions.length > 0) ||
					(Array.isArray(availableWaves) && availableWaves.length > 0) ? (
					  <div className="agent-activity-filters">
						{Array.isArray(availableRegions) && availableRegions.length > 0 && (
						  <>
							<label htmlFor="agent-activity-region-select">Region</label>
							<select
								id="agent-activity-region-select"
								value={regionFilter || ''}
								onChange={(e) =>
								  onRegionChange && onRegionChange(e.target.value)
								}
							>
							  <option value="">All Regions</option>
							  {availableRegions.map((region) => (
								<option key={region} value={region}>
								  {region}
								</option>
							  ))}
							</select>
						  </>
						)}
							{Array.isArray(availableWaves) && availableWaves.length > 0 && (
						  <>
							<label htmlFor="agent-activity-wave-select">Wave</label>
							<select
								id="agent-activity-wave-select"
								value={waveFilter || ''}
								onChange={(e) =>
								  onWaveChange && onWaveChange(e.target.value)
								}
							>
							  <option value="">All Waves</option>
							  {availableWaves.map((wave) => (
								<option key={wave} value={wave}>
								  {wave}
								</option>
							  ))}
							</select>
						  </>
						)}
							<button
							  type="button"
							  onClick={handleExportClick}
							  disabled={!canExport}
							>
							  Export to CSV
							</button>
					  </div>
					) : null}
				  </div>
			  <button
					type="button"
					className="agent-activity-close"
					onClick={onClose}
			  >
				<X size={18} />
			  </button>
			</div>

        <div className="agent-activity-body">
          {loading && (
            <div className="agent-activity-state">
              <div className="agent-activity-spinner" />
              <p>Loading Agent Activity detail...</p>
            </div>
          )}

          {!loading && error && (
            <div className="agent-activity-state error">
              <p>{error}</p>
              {onRefresh && (
                <button type="button" onClick={onRefresh}>
                  Retry
                </button>
              )}
            </div>
          )}

          {!loading && !error && !hasRows && (
            <div className="agent-activity-state">
              <p>No officers found for this category with the current filters.</p>
            </div>
          )}

          {!loading && !error && hasRows && (
            <div className="agent-activity-table-wrapper">
              <table className="agent-activity-table">
                <thead>
                  <tr>
                    <th>Officer</th>
                    <th>Branch</th>
                    <th>Region</th>
                    <th>Repayment Rate</th>
                    <th>5 Days Ago</th>
                    <th>4 Days Ago</th>
                    <th>3 Days Ago</th>
                    <th>2 Days Ago</th>
                    <th>1 Day Ago</th>
                    <th>Collected Today</th>
                    <th>Total Collected</th>
                  </tr>
                </thead>
                <tbody>
                  {rows.map((officer) => (
                    <tr key={officer.officerId || officer.officerName}>
                      <td>
                        <div className="agent-activity-officer">
                          <span className="officer-name">
                            {officer.officerName || 'Unknown officer'}
                          </span>
                          {officer.officerEmail && (
                            <span className="officer-email">
                              {officer.officerEmail}
                            </span>
                          )}
                        </div>
                      </td>
                      <td>{officer.branch}</td>
                      <td>{officer.region}</td>
                      <td>{formatPercent(officer.repaymentRate)}</td>
                      <td>{formatCurrency(officer.amount5dAgo)}</td>
                      <td>{formatCurrency(officer.amount4dAgo)}</td>
                      <td>{formatCurrency(officer.amount3dAgo)}</td>
                      <td>{formatCurrency(officer.amount2dAgo)}</td>
                      <td>{formatCurrency(officer.amount1dAgo)}</td>
                      <td>{formatCurrency(officer.amountToday)}</td>
                      <td>{formatCurrency(officer.totalCollected)}</td>
                    </tr>
                  ))}
                </tbody>
              </table>
            </div>
          )}
        </div>

        {onRefresh && !loading && !error && (
          <div className="agent-activity-footer">
            <button type="button" onClick={onRefresh}>
              Refresh data
            </button>
          </div>
        )}
      </div>
    </div>
  );
};

export default AgentActivityModal;

