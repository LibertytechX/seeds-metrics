import React, { useEffect, useMemo, useState } from 'react';
import './CollectionControlCentre.css';

const PERIOD_OPTIONS = [
  { value: 'today', label: 'Today' },
  { value: 'this_week', label: 'This Week' },
  { value: 'this_month', label: 'This Month' },
  { value: 'last_month', label: 'Last Month' },
];

const CollectionControlCentre = () => {
  const [filters, setFilters] = useState({
    period: 'today',
    region: '',
    branch: '',
    product: '',
  });

  const [filterOptions, setFilterOptions] = useState({
    regions: [],
    branches: [],
    products: [],
  });

  const [summaryMetrics, setSummaryMetrics] = useState(null);
  const [loadingFilters, setLoadingFilters] = useState(false);
  const [loadingMetrics, setLoadingMetrics] = useState(false);
  const [error, setError] = useState(null);
  const [lastUpdated, setLastUpdated] = useState(null);

  // Fetch dropdown options (regions, branches, products/loan types)
  useEffect(() => {
    const fetchFilterOptions = async () => {
      try {
        setLoadingFilters(true);

        const API_BASE_URL = import.meta.env.VITE_API_URL ||
          (import.meta.env.MODE === 'production' ? '/api/v1' : 'http://localhost:8081/api/v1');

        const [regionsRes, branchesRes, productsRes] = await Promise.all([
          fetch(`${API_BASE_URL}/filters/regions`),
          fetch(`${API_BASE_URL}/filters/branches`),
          fetch(`${API_BASE_URL}/filters/loan-types`),
        ]);

        const [regionsData, branchesData, productsData] = await Promise.all([
          regionsRes.json(),
          branchesRes.json(),
          productsRes.json(),
        ]);

        const regions = regionsData?.data?.regions || [];
        const branches = (branchesData?.data?.branches || []).map((b) => b.branch || b);
        const products = productsData?.data?.['loan-types'] || [];

        setFilterOptions({ regions, branches, products });
      } catch (err) {
        console.error('Error fetching Collection Control Centre filter options:', err);
      } finally {
        setLoadingFilters(false);
      }
    };

    fetchFilterOptions();
  }, []);

  // Refresh collections metrics whenever filters change
  useEffect(() => {
    const fetchSummaryMetrics = async () => {
      try {
        setLoadingMetrics(true);
        setError(null);

        const API_BASE_URL = import.meta.env.VITE_API_URL ||
          (import.meta.env.MODE === 'production' ? '/api/v1' : 'http://localhost:8081/api/v1');

        const params = new URLSearchParams();
        // We only need summary metrics, so request a single row from /loans
        params.set('page', '1');
        params.set('limit', '1');

        if (filters.branch) {
          params.set('branch', filters.branch);
        }
        if (filters.region) {
          params.set('region', filters.region);
        }
        if (filters.product) {
          params.set('loan_type', filters.product);
        }

        // NOTE: For now, the backend summary metrics are calculated for "today".
        // The period filter is wired and will trigger refreshes, but broader
        // period handling will be implemented in a dedicated backend endpoint.

        const response = await fetch(`${API_BASE_URL}/loans?${params.toString()}`);
        const data = await response.json();

        if (data.status !== 'success') {
          throw new Error(data.message || 'Failed to load collections summary metrics');
        }

        setSummaryMetrics(data.data?.summary_metrics || null);
        setLastUpdated(new Date());
      } catch (err) {
        console.error('Error fetching collections summary metrics:', err);
        setError(err.message || 'Error fetching collections data');
        setSummaryMetrics(null);
      } finally {
        setLoadingMetrics(false);
      }
    };

    fetchSummaryMetrics();
  }, [filters.branch, filters.region, filters.product, filters.period]);

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
    return `${safe.toFixed(1)}%`;
  };

  const lastUpdatedLabel = useMemo(() => {
    if (!lastUpdated) return '—';
    return lastUpdated.toLocaleTimeString();
  }, [lastUpdated]);

  const isLoading = loadingFilters || loadingMetrics;

  // Derived values from existing summary metrics
  const totalDueToday = summaryMetrics?.total_due_for_today ?? null;
  const totalRepaidToday = summaryMetrics?.total_repayments_today ?? null;
  const collectionRateToday = summaryMetrics?.percentage_of_due_collected ?? null;
  const atRiskInfo = summaryMetrics?.at_risk_loans || null;
  const totalPortfolioAmount = summaryMetrics?.total_portfolio_amount ?? null;
  const totalInDPD = summaryMetrics?.total_amount_in_dpd ?? null;

  const handleCardClick = (target) => {
    // Placeholder click handlers – these will later open specific drilldown tables
    // using loans + repayments filters only.
    // For now we simply log so we can verify interactions end-to-end.
    // eslint-disable-next-line no-console
    console.log(`Collections Control Centre card clicked: ${target}`);
  };

  return (
    <div className="collections-page">
      <div className="collections-header">
        <div>
          <h2>Collections Control Centre</h2>
          <p className="collections-subtitle">
            Central view of collections performance. All metrics are derived strictly
            from loans and repayments data.
          </p>
        </div>
        <div className="collections-meta">
          <span className="last-updated">Last updated: {lastUpdatedLabel}</span>
          {isLoading && <span className="loading-indicator">Refreshing...</span>}
        </div>
      </div>

      <div className="collections-filters">
        <div className="filter-group">
          <label>Period</label>
          <select
            value={filters.period}
            onChange={(e) => handleFilterChange('period', e.target.value)}
          >
            {PERIOD_OPTIONS.map((opt) => (
              <option key={opt.value} value={opt.value}>{opt.label}</option>
            ))}
          </select>
        </div>

        <div className="filter-group">
          <label>Region</label>
          <select
            value={filters.region}
            onChange={(e) => handleFilterChange('region', e.target.value)}
          >
            <option value="">All Regions</option>
            {filterOptions.regions.map((region) => (
              <option key={region} value={region}>{region}</option>
            ))}
          </select>
        </div>

        <div className="filter-group">
          <label>Branch</label>
          <select
            value={filters.branch}
            onChange={(e) => handleFilterChange('branch', e.target.value)}
          >
            <option value="">All Branches</option>
            {filterOptions.branches.map((branch) => (
              <option key={branch} value={branch}>{branch}</option>
            ))}
          </select>
        </div>

        <div className="filter-group">
          <label>Product (Loan Type)</label>
          <select
            value={filters.product}
            onChange={(e) => handleFilterChange('product', e.target.value)}
          >
            <option value="">All Products</option>
            {filterOptions.products.map((product) => (
              <option key={product} value={product}>{product}</option>
            ))}
          </select>
        </div>
      </div>

      {error && (
        <div className="collections-error">
          Failed to load collections metrics: {error}
        </div>
      )}

      <div className="collections-grid">
        {/* 1. Collections Due (Period) */}
        <button
          type="button"
          className="collection-card kpi-blue"
          onClick={() => handleCardClick('collections-due')}
        >
          <div className="card-label">Collections Due ({PERIOD_OPTIONS.find(p => p.value === filters.period)?.label || 'Period'})</div>
          <div className="card-value">{totalDueToday != null ? formatCurrency(totalDueToday) : '—'}</div>
          <div className="card-subtitle">Total scheduled repayments for the selected period</div>
        </button>

        {/* 2. Collections Received (Period) */}
        <button
          type="button"
          className="collection-card kpi-green"
          onClick={() => handleCardClick('collections-received')}
        >
          <div className="card-label">Collections Received ({PERIOD_OPTIONS.find(p => p.value === filters.period)?.label || 'Period'})</div>
          <div className="card-value">{totalRepaidToday != null ? formatCurrency(totalRepaidToday) : '—'}</div>
          <div className="card-subtitle">Cash collected from repayments in the period</div>
        </button>

        {/* 3. Collection Rate (Period) */}
        <button
          type="button"
          className="collection-card kpi-green"
          onClick={() => handleCardClick('collection-rate')}
        >
          <div className="card-label">Collection Rate ({PERIOD_OPTIONS.find(p => p.value === filters.period)?.label || 'Period'})</div>
          <div className="card-value">{collectionRateToday != null ? formatPercent(collectionRateToday) : '—'}</div>
          <div className="card-subtitle">% of due amount collected in the period</div>
        </button>

        {/* 4. Missed Repayments (Period) – placeholder */}
        <button
          type="button"
          className="collection-card kpi-amber"
          onClick={() => handleCardClick('missed-repayments')}
        >
          <div className="card-label">Missed Repayments ({PERIOD_OPTIONS.find(p => p.value === filters.period)?.label || 'Period'})</div>
          <div className="card-value">Coming soon</div>
          <div className="card-subtitle">Will use loans + repayments only to count missed dues</div>
        </button>

        {/* 5. Past Maturity Outstanding – placeholder */}
        <button
          type="button"
          className="collection-card kpi-red"
          onClick={() => handleCardClick('past-maturity-outstanding')}
        >
          <div className="card-label">Past Maturity Outstanding</div>
          <div className="card-value">Coming soon</div>
          <div className="card-subtitle">Outstanding balance on loans past maturity date</div>
        </button>

        {/* 6. Portfolio Outstanding (using total portfolio for now) */}
        <button
          type="button"
          className="collection-card kpi-blue"
          onClick={() => handleCardClick('portfolio-outstanding')}
        >
          <div className="card-label">Portfolio Outstanding</div>
          <div className="card-value">{totalPortfolioAmount != null ? formatCurrency(totalPortfolioAmount) : '—'}</div>
          <div className="card-subtitle">Total portfolio amount for selected filters</div>
        </button>

        {/* 7. Portfolio Health – placeholder derived from at-risk share later */}
        <button
          type="button"
          className="collection-card kpi-green"
          onClick={() => handleCardClick('portfolio-health')}
        >
          <div className="card-label">Portfolio Health</div>
          <div className="card-value">Coming soon</div>
          <div className="card-subtitle">Summary view of healthy vs at-risk portfolio</div>
        </button>

        {/* 8. At-Risk Portfolio */}
        <button
          type="button"
          className="collection-card kpi-amber"
          onClick={() => handleCardClick('at-risk-portfolio')}
        >
          <div className="card-label">At-Risk Portfolio</div>
          <div className="card-value">{atRiskInfo?.amount != null ? formatCurrency(atRiskInfo.amount) : '—'}</div>
          <div className="card-subtitle">
            {atRiskInfo?.percentage != null ? `${formatPercent(atRiskInfo.percentage)} of total portfolio` : 'Loans with current DPD > 14 days'}
          </div>
        </button>

        {/* 9. NPL Ratio – placeholder using total in DPD for now */}
        <button
          type="button"
          className="collection-card kpi-red"
          onClick={() => handleCardClick('npl-ratio')}
        >
          <div className="card-label">NPL Ratio</div>
          <div className="card-value">
            {totalInDPD != null && totalPortfolioAmount
              ? `${formatPercent((totalInDPD / totalPortfolioAmount) * 100)} (approx)`
              : 'Coming soon'}
          </div>
          <div className="card-subtitle">Provisional, to be aligned with your exact NPL definition</div>
        </button>
      </div>
    </div>
  );
};

export default CollectionControlCentre;
