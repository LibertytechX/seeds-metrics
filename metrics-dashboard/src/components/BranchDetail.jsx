import React, { useEffect, useMemo, useState } from 'react';
import {
  Bar,
  CartesianGrid,
  ComposedChart,
  Legend,
  ResponsiveContainer,
  Tooltip as RechartsTooltip,
  XAxis,
  YAxis,
  LabelList,
} from 'recharts';
import './BranchDetail.css';

// Reuse the same sentinel value used in AllLoans and backend (MissingValueSentinel)
const MISSING_VALUE = '__MISSING__';

const PERIOD_OPTIONS = [
  { value: 'today', label: 'Today' },
  { value: 'today_only', label: 'Today Only' },
  { value: 'this_week', label: 'This Week' },
  { value: 'this_month', label: 'This Month' },
  { value: 'last_month', label: 'Last Month' },
];

const BranchDetail = ({ branchSlug, onBack }) => {
  // Convert slug to branch name (e.g., "lagos-island" -> "Lagos-Island")
  const branchName = useMemo(() => {
    if (!branchSlug) return '';
    return branchSlug
      .split('-')
      .map((word) => word.charAt(0).toUpperCase() + word.slice(1))
      .join('-');
  }, [branchSlug]);

	  const [filters, setFilters] = useState({
	    period: 'this_month',
	    region: '',
	    product: '',
	    wave: '',
	  });

	  const [filterOptions, setFilterOptions] = useState({
	    regions: [],
	    products: [],
	    waves: [],
	  });

	  const [summaryMetrics, setSummaryMetrics] = useState(null);
	  const [totalRepaidAll, setTotalRepaidAll] = useState(null);
	  const [branchInfo, setBranchInfo] = useState(null);
	  // This holds the exact backend branch name (e.g. "EPE"), used for filters
	  const [branchFilterValue, setBranchFilterValue] = useState('');
	  const [dailyCollections, setDailyCollections] = useState([]);
	  const [agentLeaderboard, setAgentLeaderboard] = useState([]);
	  const [loadingFilters, setLoadingFilters] = useState(false);
	  const [loadingMetrics, setLoadingMetrics] = useState(false);
	  const [loadingDaily, setLoadingDaily] = useState(false);
	  const [loadingAgents, setLoadingAgents] = useState(false);
	  const [error, setError] = useState(null);

  // Fetch dropdown options (regions, products/loan types)
  useEffect(() => {
	    const fetchFilterOptions = async () => {
	      try {
	        setLoadingFilters(true);
	        const API_BASE_URL = import.meta.env.VITE_API_URL ||
	          (import.meta.env.MODE === 'production' ? '/api/v1' : 'http://localhost:8081/api/v1');

	        const [regionsRes, productsRes, wavesRes] = await Promise.all([
	          fetch(`${API_BASE_URL}/filters/regions`),
	          fetch(`${API_BASE_URL}/filters/loan-types`),
	          fetch(`${API_BASE_URL}/filters/waves`),
	        ]);

	        const [regionsData, productsData, wavesData] = await Promise.all([
	          regionsRes.json(),
	          productsRes.json(),
	          wavesRes.json(),
	        ]);

	        const regions = regionsData?.data?.regions || [];
	        const products = productsData?.data?.['loan-types'] || [];
	        const waves = wavesData?.data?.waves || [];
	        setFilterOptions({ regions, products, waves });
	      } catch (err) {
        console.error('Error fetching Branch Detail filter options:', err);
      } finally {
        setLoadingFilters(false);
      }
    };

    fetchFilterOptions();
  }, []);

	  // Fetch branch info from leaderboard to get region and canonical branch name
  useEffect(() => {
	    const fetchBranchInfo = async () => {
	      if (!branchName) return;
	      try {
		        const API_BASE_URL = import.meta.env.VITE_API_URL ||
		          (import.meta.env.MODE === 'production' ? '/api/v1' : 'http://localhost:8081/api/v1');

		        const params = new URLSearchParams();
		        // Do NOT filter by branch here; fetch all, then match by slug so we can
		        // discover the exact backend branch name (which may be upper-case like "EPE").
		        if (filters.region) params.set('region', filters.region);
		        if (filters.product) params.set('loan_type', filters.product);
		        if (filters.wave) params.set('wave', filters.wave);
	        const djangoStatusFilter = filters.period === 'today_only'
	          ? 'OPEN'
	          : ['OPEN', 'PAST_MATURITY', MISSING_VALUE].join(',');
	        params.set('django_status', djangoStatusFilter);

		        const res = await fetch(`${API_BASE_URL}/collections/branches?${params.toString()}`);
	        const data = await res.json();
	        if (data.status === 'success' && data.data?.branches) {
	          const match = data.data.branches.find(
	            (b) => b.branch && b.branch.toLowerCase().replace(/\s+/g, '-') === branchSlug
	          );
	          if (match) {
	            setBranchInfo(match);
		            setBranchFilterValue(match.branch || '');
	          }
	        }
	      } catch (err) {
        console.error('Error fetching branch info:', err);
      }
    };

	    fetchBranchInfo();
	  }, [branchSlug, branchName, filters.region, filters.product, filters.wave, filters.period]);

	  // Fetch summary metrics for this branch (uses canonical branchFilterValue)
  useEffect(() => {
    const fetchSummaryMetrics = async () => {
	      if (!branchFilterValue) return;
      try {
        setLoadingMetrics(true);
        setError(null);

        const API_BASE_URL = import.meta.env.VITE_API_URL ||
          (import.meta.env.MODE === 'production' ? '/api/v1' : 'http://localhost:8081/api/v1');

		        const baseParams = new URLSearchParams();
		        baseParams.set('page', '1');
		        baseParams.set('limit', '1');
		        baseParams.set('branch', branchFilterValue);

	        if (filters.region) baseParams.set('region', filters.region);
	        if (filters.product) baseParams.set('loan_type', filters.product);
	        if (filters.wave) baseParams.set('wave', filters.wave);
	        if (filters.period) baseParams.set('period', filters.period);

        // For "today_only" period, only show OPEN loans
        const restrictedParams = new URLSearchParams(baseParams.toString());
        const djangoStatusFilter = filters.period === 'today_only'
          ? 'OPEN'
          : ['OPEN', 'PAST_MATURITY', MISSING_VALUE].join(',');
        restrictedParams.set('django_status', djangoStatusFilter);

        const unrestrictedParams = new URLSearchParams(baseParams.toString());

        const [restrictedRes, unrestrictedRes] = await Promise.all([
          fetch(`${API_BASE_URL}/loans?${restrictedParams.toString()}`),
          fetch(`${API_BASE_URL}/loans?${unrestrictedParams.toString()}`),
        ]);

        const [restrictedData, unrestrictedData] = await Promise.all([
          restrictedRes.json(),
          unrestrictedRes.json(),
        ]);

        if (restrictedData.status === 'success') {
          setSummaryMetrics(restrictedData.data?.summary_metrics || null);
        }
        if (unrestrictedData.status === 'success') {
          setTotalRepaidAll(unrestrictedData.data?.summary_metrics?.total_repayments_today ?? null);
        }
      } catch (err) {
        console.error('Error fetching branch summary metrics:', err);
        setError(err.message || 'Error fetching branch data');
      } finally {
        setLoadingMetrics(false);
      }
    };

	    fetchSummaryMetrics();
	  }, [branchFilterValue, filters.region, filters.product, filters.period, filters.wave]);

	  // Fetch daily collections for the chart (uses canonical branchFilterValue)
  useEffect(() => {
    const fetchDailyCollections = async () => {
	      if (!branchFilterValue) return;
      try {
        setLoadingDaily(true);

        const API_BASE_URL = import.meta.env.VITE_API_URL ||
          (import.meta.env.MODE === 'production' ? '/api/v1' : 'http://localhost:8081/api/v1');

		        const params = new URLSearchParams();
		        params.set('period', 'last_7_days');
		        params.set('branch', branchFilterValue);
	        if (filters.region) params.set('region', filters.region);
	        if (filters.product) params.set('loan_type', filters.product);
	        if (filters.wave) params.set('wave', filters.wave);

        const res = await fetch(`${API_BASE_URL}/collections/daily?${params.toString()}`);
        const data = await res.json();
        if (data.status === 'success') {
          setDailyCollections(data.data?.points || []);
        }
      } catch (err) {
        console.error('Error fetching daily collections:', err);
      } finally {
        setLoadingDaily(false);
      }
    };

	    fetchDailyCollections();
		  }, [branchFilterValue, filters.region, filters.product, filters.wave]);

	  // Fetch Agent/Officer collections leaderboard for this branch
	  useEffect(() => {
	    const fetchAgentLeaderboard = async () => {
	      if (!branchFilterValue) return;
	      try {
	        setLoadingAgents(true);
	        const API_BASE_URL = import.meta.env.VITE_API_URL ||
	          (import.meta.env.MODE === 'production' ? '/api/v1' : 'http://localhost:8081/api/v1');

	        const params = new URLSearchParams();
	        params.set('branch', branchFilterValue);
	        if (filters.region) params.set('region', filters.region);
	        if (filters.product) params.set('loan_type', filters.product);
	        if (filters.wave) params.set('wave', filters.wave);

	        const djangoStatusFilter = filters.period === 'today_only'
	          ? 'OPEN'
	          : ['OPEN', 'PAST_MATURITY', MISSING_VALUE].join(',');
	        params.set('django_status', djangoStatusFilter);

	        const res = await fetch(`${API_BASE_URL}/collections/officers?${params.toString()}`);
	        const data = await res.json();
	        if (data.status === 'success') {
	          setAgentLeaderboard(data.data?.officers || []);
	        } else {
	          console.error('Failed to fetch agent leaderboard:', data.message);
	          setAgentLeaderboard([]);
	        }
	      } catch (err) {
	        console.error('Error fetching agent leaderboard:', err);
	        setAgentLeaderboard([]);
	      } finally {
	        setLoadingAgents(false);
	      }
	    };

	    fetchAgentLeaderboard();
	  }, [branchFilterValue, filters.region, filters.product, filters.wave, filters.period]);

  const handleFilterChange = (key, value) => {
    setFilters((prev) => ({ ...prev, [key]: value }));
  };

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
    const safe = typeof value === 'number' ? value : 0;
    return `${safe.toFixed(0)}%`;
  };

  const formatShortCurrency = (value) => {
    if (typeof value !== 'number' || Number.isNaN(value)) return '₦0';
    const abs = Math.abs(value);
    if (abs >= 1_000_000_000) return `₦${(value / 1_000_000_000).toFixed(1)}B`;
    if (abs >= 1_000_000) return `₦${(value / 1_000_000).toFixed(1)}M`;
    if (abs >= 1_000) return `₦${(value / 1_000).toFixed(1)}k`;
    return formatCurrency(value);
  };

  // Derived metrics
  const totalDueToday = summaryMetrics?.total_due_for_today ?? 0;
  const totalRepaidToday = totalRepaidAll ?? summaryMetrics?.total_repayments_today ?? 0;
  const collectionRateToday = summaryMetrics?.percentage_of_due_collected ?? 0;
  const missedRepayments = summaryMetrics?.missed_repayments_today ?? 0;
  const pastMaturityOutstanding = summaryMetrics?.past_maturity_outstanding ?? 0;
  const portfolioAmount = summaryMetrics?.total_portfolio_amount ?? 0;
  const atRiskInfo = summaryMetrics?.at_risk_loans || {};
  const portfolioHealth = summaryMetrics?.portfolio_health || {};
  const totalLoans = summaryMetrics?.total_loans ?? 0;

  // Risk snapshot percentages
  const performingCount = portfolioHealth?.performing_loans_count ?? 0;
  const atRiskCount = atRiskInfo?.count ?? 0;
  const criticalCount = summaryMetrics?.critical_loans?.count ?? 0;
  const performingPercent = totalLoans > 0 ? (performingCount / totalLoans) * 100 : 0;
  const atRiskPercent = totalLoans > 0 ? (atRiskCount / totalLoans) * 100 : 0;
  const nplPercent = branchInfo?.npl_ratio ? branchInfo.npl_ratio * 100 : 0;

  // MTD rate from branchInfo
  const mtdRate = branchInfo?.mtd_rate ? branchInfo.mtd_rate * 100 : 0;

  // Daily collections chart series
  const dailyChartSeries = useMemo(() => {
    const byDate = new Map();
    if (Array.isArray(dailyCollections)) {
      dailyCollections.forEach((point) => {
        if (!point || !point.date) return;
        let key = point.date;
        const asDate = new Date(point.date);
        if (!Number.isNaN(asDate.getTime())) {
          key = asDate.toISOString().slice(0, 10);
        } else if (typeof key === 'string') {
          key = key.slice(0, 10);
        }

        const collected = typeof point.collected_amount === 'number' ? point.collected_amount : 0;
        byDate.set(key, { date: key, collected_amount: collected });
      });
    }

    const series = [];
    for (let offset = 6; offset >= 0; offset -= 1) {
      const d = new Date();
      d.setDate(d.getDate() - offset);
      const key = d.toISOString().slice(0, 10);
      const existing = byDate.get(key);
      const dayLabel = d.toLocaleDateString('en-NG', { weekday: 'short' });

      const dueAmount = offset === 0 ? totalDueToday : 0;
      const collected = existing ? existing.collected_amount : 0;
      const rate = dueAmount > 0 ? (collected / dueAmount) * 100 : null;

      series.push({
        date: key,
        day_label: dayLabel,
        collected_amount: collected,
        due_amount: dueAmount,
        collection_rate_percent: rate,
        isToday: offset === 0,
      });
    }

    return series;
  }, [dailyCollections, totalDueToday]);

	  const isLoading = loadingFilters || loadingMetrics;
	  const regionName = branchInfo?.region || filters.region || 'Region';

  return (
    <div className="branch-detail-page">
      {/* Breadcrumb / Header */}
      <div className="branch-detail-header">
        <div className="branch-detail-breadcrumb">
          <button type="button" className="breadcrumb-link" onClick={onBack}>
            Overview
          </button>
          <span className="breadcrumb-separator">/</span>
	          <span className="breadcrumb-current">{branchInfo?.branch || branchName || branchSlug}</span>
        </div>
        <p className="branch-detail-region">Region: {regionName}</p>
      </div>

      {/* Filters */}
      <div className="branch-detail-filters">
        <div className="filter-group">
          <label htmlFor="period">Period</label>
          <select
            id="period"
            value={filters.period}
            onChange={(e) => handleFilterChange('period', e.target.value)}
            disabled={isLoading}
          >
            {PERIOD_OPTIONS.map((opt) => (
              <option key={opt.value} value={opt.value}>{opt.label}</option>
            ))}
          </select>
        </div>
        <div className="filter-group">
          <label htmlFor="region">Region</label>
          <select
            id="region"
            value={filters.region}
            onChange={(e) => handleFilterChange('region', e.target.value)}
            disabled={isLoading}
          >
            <option value="">All Regions</option>
            {filterOptions.regions.map((r) => (
              <option key={r} value={r}>{r}</option>
            ))}
          </select>
        </div>
        <div className="filter-group">
          <label htmlFor="product">Product</label>
          <select
            id="product"
            value={filters.product}
            onChange={(e) => handleFilterChange('product', e.target.value)}
            disabled={isLoading}
          >
            <option value="">All Products</option>
            {filterOptions.products.map((p) => (
              <option key={p} value={p}>{p}</option>
            ))}
          </select>
        </div>
	        <div className="filter-group">
	          <label htmlFor="wave">Wave</label>
	          <select
	            id="wave"
	            value={filters.wave}
	            onChange={(e) => handleFilterChange('wave', e.target.value)}
	            disabled={isLoading}
	          >
	            <option value="">All Waves</option>
	            {filterOptions.waves.map((w) => (
	              <option key={w} value={w}>{w}</option>
	            ))}
	          </select>
	        </div>
      </div>

      {error && <div className="branch-detail-error">{error}</div>}

      {/* KPI Cards */}
      <div className="branch-detail-cards">
        <div className="branch-kpi-card">
          <div className="kpi-label">Portfolio</div>
          <div className="kpi-value">{formatCurrency(portfolioAmount)}</div>
        </div>
        <div className="branch-kpi-card">
          <div className="kpi-label">Today: Due / Collected</div>
          <div className="kpi-value">
            {formatShortCurrency(totalDueToday)} · {formatShortCurrency(totalRepaidToday)}
          </div>
        </div>
        <div className="branch-kpi-card">
          <div className="kpi-label">Today / MTD %</div>
          <div className="kpi-value">
            {formatPercent(collectionRateToday)} · {formatPercent(mtdRate)}
          </div>
        </div>
        <div className="branch-kpi-card">
          <div className="kpi-label">Missed (Today)</div>
          <div className="kpi-value">{formatCurrency(missedRepayments)}</div>
        </div>
        <div className="branch-kpi-card">
          <div className="kpi-label">Past Maturity</div>
          <div className="kpi-value">{formatCurrency(pastMaturityOutstanding)}</div>
        </div>
      </div>

      {/* Daily Collections Chart */}
      <div className="branch-detail-chart-section">
        <h3>Branch Daily Collections vs Due</h3>
        <p className="chart-subtitle">{branchName} · This Week</p>
        {loadingDaily ? (
          <div className="chart-loading">Loading chart...</div>
        ) : (
          <div className="branch-chart-container">
            <ResponsiveContainer width="100%" height={280}>
              <ComposedChart data={dailyChartSeries} margin={{ top: 30, right: 20, left: 20, bottom: 10 }}>
                <CartesianGrid strokeDasharray="3 3" vertical={false} />
                <XAxis dataKey="day_label" tick={{ fontSize: 12 }} />
                <YAxis tickFormatter={formatShortCurrency} tick={{ fontSize: 11 }} width={70} />
                <RechartsTooltip
                  formatter={(value, name) => [formatCurrency(value), name === 'collected_amount' ? 'Collected' : 'Due']}
                  labelFormatter={(label) => label}
                />
                <Bar dataKey="due_amount" name="Due" fill="#f59e0b" barSize={28} radius={[4, 4, 0, 0]} />
                <Bar dataKey="collected_amount" name="Collected" fill="#22c55e" barSize={28} radius={[4, 4, 0, 0]}>
                  <LabelList
                    dataKey="collection_rate_percent"
                    position="top"
                    formatter={(val) => (val !== null && val !== undefined ? `${Math.round(val)}%` : '')}
                    style={{ fontSize: 11, fontWeight: 600, fill: '#374151' }}
                  />
                </Bar>
              </ComposedChart>
            </ResponsiveContainer>
          </div>
        )}
      </div>

      {/* Risk Snapshot */}
      <div className="branch-detail-risk-section">
        <h3>Branch Risk Snapshot</h3>
        <div className="risk-snapshot-grid">
          <div className="risk-item">
            <span className="risk-label">Performing:</span>
            <span className="risk-value ok">{formatPercent(performingPercent)}</span>
          </div>
          <div className="risk-item">
            <span className="risk-label">At-Risk:</span>
            <span className="risk-value warning">{formatPercent(atRiskPercent)}</span>
          </div>
          <div className="risk-item">
            <span className="risk-label">NPL:</span>
            <span className="risk-value danger">{formatPercent(nplPercent)}</span>
          </div>
          <div className="risk-item">
            <span className="risk-label">Total Loans:</span>
            <span className="risk-value">{totalLoans}</span>
          </div>
        </div>

	      {/* Agent Leaderboard (per-officer collections for this branch) */}
	      <div className="branch-detail-agent-section">
	        <h3>Agent Leaderboard</h3>
	        <p className="agent-leaderboard-subtitle">
		        Loan officers in {branchInfo?.branch || branchName || branchSlug} - today's collections vs due
	        </p>
	        <div className="agent-leaderboard-table-wrapper">
	          <table className="agent-leaderboard-table">
	            <thead>
	              <tr>
	                <th>Officer</th>
	                <th>Email</th>
	                <th>Portfolio</th>
	                <th>Due</th>
	                <th>Coll.</th>
	                <th>Today%</th>
	                <th>MTD%</th>
	                <th>Missed</th>
	                <th>NPL%</th>
	                <th>Status</th>
	              </tr>
	            </thead>
	            <tbody>
	              {loadingAgents ? (
	                <tr>
	                  <td colSpan={10} className="agent-leaderboard-loading">
	                    Loading agent leaderboard...
	                  </td>
	                </tr>
	              ) : agentLeaderboard.length === 0 ? (
	                <tr>
	                  <td colSpan={10} className="agent-leaderboard-empty">
	                    No agents found for the selected filters.
	                  </td>
	                </tr>
	              ) : (
	                agentLeaderboard.map((a) => {
	                  const todayRatePercentage = (typeof a.today_rate === 'number' ? a.today_rate : 0) * 100;
	                  const mtdRatePercentage = (typeof a.mtd_rate === 'number' ? a.mtd_rate : 0) * 100;
	                  const nplPercentage = (typeof a.npl_ratio === 'number' ? a.npl_ratio : 0) * 100;
	                  const key = a.officer_id || a.officer_email || a.officer_name || Math.random().toString(36);
	                  return (
	                    <tr key={key}>
	                      <td>{a.officer_name || '-'}</td>
			                  <td>{a.officer_email || '—'}</td>
	                      <td>{formatCurrency(a.portfolio_total)}</td>
	                      <td>{formatCurrency(a.due_today)}</td>
	                      <td>{formatCurrency(a.collected_today)}</td>
	                      <td>{formatPercent(todayRatePercentage)}</td>
	                      <td>{formatPercent(mtdRatePercentage)}</td>
	                      <td>{formatCurrency(a.missed_today)}</td>
	                      <td>{formatPercent(nplPercentage)}</td>
	                      <td>
	                        <span className={`status-pill status-${(a.status || 'OK').toLowerCase()}`}>
	                          {a.status || 'OK'}
	                        </span>
	                      </td>
	                    </tr>
	                  );
	                })
	              )}
	            </tbody>
	          </table>
	        </div>
	      </div>
      </div>
    </div>
  );
};

export default BranchDetail;

