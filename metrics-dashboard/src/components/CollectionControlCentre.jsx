import React, { useEffect, useMemo, useState } from 'react';
import './CollectionControlCentre.css';

// Reuse the same sentinel value used in AllLoans and backend (MissingValueSentinel)
const MISSING_VALUE = '__MISSING__';

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
	  // For Collections Received we want "all repayments" for the period,
	  // not restricted to collections-specific django_status values.
	  // We'll fetch this alongside the restricted metrics.
	  const [totalRepaidTodayAll, setTotalRepaidTodayAll] = useState(null);

	  // Branch collections leaderboard (per-branch breakdown under the cards)
	  const [branchLeaderboard, setBranchLeaderboard] = useState([]);
	  const [loadingBranches, setLoadingBranches] = useState(false);
	  const [branchesError, setBranchesError] = useState(null);
	  const [branchSort, setBranchSort] = useState({
	    key: 'npl_ratio',
	    direction: 'desc',
	  });

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

	        const baseParams = new URLSearchParams();
	        // We only need summary metrics, so request a single row from /loans
	        baseParams.set('page', '1');
	        baseParams.set('limit', '1');

	        if (filters.branch) {
	          baseParams.set('branch', filters.branch);
	        }
	        if (filters.region) {
	          baseParams.set('region', filters.region);
	        }
	        if (filters.product) {
	          baseParams.set('loan_type', filters.product);
	        }
	        if (filters.period) {
	          baseParams.set('period', filters.period);
	        }

	        // Restricted metrics (used for Collections Due, At-Risk, etc.)
	        // Per collections requirements, restrict to loans that are relevant
	        // for collections work:
	        //   - django_status = OPEN
	        //   - django_status = PAST_MATURITY
	        //   - Missing django_status (NULL / empty), via sentinel
	        // This uses the same MissingValueSentinel logic as AllLoans/backend.
	        const restrictedParams = new URLSearchParams(baseParams.toString());
	        const djangoStatusFilter = ['OPEN', 'PAST_MATURITY', MISSING_VALUE].join(',');
	        restrictedParams.set('django_status', djangoStatusFilter);

	        // Unrestricted metrics (for Collections Received): all repayments,
	        // still respecting region/branch/product filters but NOT restricted
	        // by django_status.
	        const unrestrictedParams = new URLSearchParams(baseParams.toString());

	        const [restrictedRes, unrestrictedRes] = await Promise.all([
	          fetch(`${API_BASE_URL}/loans?${restrictedParams.toString()}`),
	          fetch(`${API_BASE_URL}/loans?${unrestrictedParams.toString()}`),
	        ]);

	        const [restrictedData, unrestrictedData] = await Promise.all([
	          restrictedRes.json(),
	          unrestrictedRes.json(),
	        ]);

	        if (restrictedData.status !== 'success') {
	          throw new Error(restrictedData.message || 'Failed to load collections summary metrics');
	        }
	        if (unrestrictedData.status !== 'success') {
	          throw new Error(unrestrictedData.message || 'Failed to load collections received metrics');
	        }

	        setSummaryMetrics(restrictedData.data?.summary_metrics || null);
	        setTotalRepaidTodayAll(
	          unrestrictedData.data?.summary_metrics?.total_repayments_today ?? null,
	        );
	        setLastUpdated(new Date());
	      } catch (err) {
	        console.error('Error fetching collections summary metrics:', err);
	        setError(err.message || 'Error fetching collections data');
	        setSummaryMetrics(null);
	        setTotalRepaidTodayAll(null);
	      } finally {
	        setLoadingMetrics(false);
	      }
	    };

	    fetchSummaryMetrics();
	  }, [filters.branch, filters.region, filters.product, filters.period]);

	  // Fetch branch collections leaderboard (per-branch breakdown) when filters change.
	  useEffect(() => {
	    const fetchBranchLeaderboard = async () => {
	      try {
	        setLoadingBranches(true);
	        setBranchesError(null);

	        const API_BASE_URL = import.meta.env.VITE_API_URL ||
	          (import.meta.env.MODE === 'production' ? '/api/v1' : 'http://localhost:8081/api/v1');

	        const params = new URLSearchParams();
	        if (filters.region) {
	          params.set('region', filters.region);
	        }
	        if (filters.branch) {
	          params.set('branch', filters.branch);
	        }
	        if (filters.product) {
	          params.set('loan_type', filters.product);
	        }

	        const queryString = params.toString();
	        const url = queryString
	          ? `${API_BASE_URL}/collections/branches?${queryString}`
	          : `${API_BASE_URL}/collections/branches`;

	        const res = await fetch(url);
	        const data = await res.json();
	        if (data.status !== 'success') {
	          throw new Error(data.message || 'Failed to load branch leaderboard');
	        }

	        setBranchLeaderboard(data.data?.branches || []);
	      } catch (err) {
	        console.error('Error fetching branch collections leaderboard:', err);
	        setBranchesError(err.message || 'Error fetching branch leaderboard');
	        setBranchLeaderboard([]);
	      } finally {
	        setLoadingBranches(false);
	      }
	    };

	    fetchBranchLeaderboard();
	  }, [filters.branch, filters.region, filters.product]);

  const handleFilterChange = (key, value) => {
    setFilters((prev) => ({ ...prev, [key]: value }));
  };

	  const handleLeaderboardSort = (key) => {
	    setBranchSort((prev) => {
	      if (prev.key === key) {
	        return {
	          key,
	          direction: prev.direction === 'asc' ? 'desc' : 'asc',
	        };
	      }
	      return {
	        key,
	        direction: 'desc',
	      };
	    });
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

	  const sortedBranchLeaderboard = useMemo(() => {
	    if (!branchLeaderboard || branchLeaderboard.length === 0) return [];
	    const rows = [...branchLeaderboard];
	    rows.sort((a, b) => {
	      const aVal = typeof a[branchSort.key] === 'number' ? a[branchSort.key] : 0;
	      const bVal = typeof b[branchSort.key] === 'number' ? b[branchSort.key] : 0;
	      if (aVal === bVal) return 0;
	      if (branchSort.direction === 'asc') {
	        return aVal - bVal;
	      }
	      return bVal - aVal;
	    });
	    return rows;
	  }, [branchLeaderboard, branchSort]);

			  // Derived values from existing summary metrics
			  const totalDueToday = summaryMetrics?.total_due_for_today ?? null;
			  // For Collections Received, use the unrestricted "all repayments" value
			  // when available; fall back to the restricted one if needed.
			  const totalRepaidToday =
			    totalRepaidTodayAll ?? summaryMetrics?.total_repayments_today ?? null;
			  const collectionRateToday = summaryMetrics?.percentage_of_due_collected ?? null;
			  const missedRepaymentsTodayAmount = summaryMetrics?.missed_repayments_today ?? null;
			  const missedRepaymentsTodayCount = summaryMetrics?.missed_repayments_today_count ?? null;
		  const atRiskInfo = summaryMetrics?.at_risk_loans || null;
		  const totalPortfolioAmount = summaryMetrics?.total_portfolio_amount ?? null;
		  const totalInDPD = summaryMetrics?.total_amount_in_dpd ?? null;
		  const pastMaturityOutstanding = summaryMetrics?.past_maturity_outstanding ?? null;
		  const portfolioHealthAmount =
		    summaryMetrics?.portfolio_health?.performing_actual_outstanding ?? null;
		  const portfolioHealthCount =
		    summaryMetrics?.portfolio_health?.performing_loans_count ?? null;

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

	        {/* 4. Missed Repayments (Today) */}
        <button
          type="button"
          className="collection-card kpi-amber"
          onClick={() => handleCardClick('missed-repayments')}
        >
	          <div className="card-label">Missed Repayments (Today)</div>
	          <div className="card-value">
	            {missedRepaymentsTodayAmount != null
	              ? formatCurrency(missedRepaymentsTodayAmount)
	              : '—'}
	          </div>
	          <div className="card-subtitle">
	            {missedRepaymentsTodayCount != null
	              ? `${missedRepaymentsTodayCount.toLocaleString()} loans with a scheduled repayment today but no payment recorded`
	              : 'Loans with a scheduled repayment today but no repayment recorded today'}
	          </div>
        </button>

	        {/* 5. Past Maturity Outstanding */}
	        <button
	          type="button"
	          className="collection-card kpi-red"
	          onClick={() => handleCardClick('past-maturity-outstanding')}
	        >
	          <div className="card-label">Past Maturity Outstanding</div>
		          <div className="card-value">
		            {pastMaturityOutstanding != null
		              ? formatCurrency(pastMaturityOutstanding)
		              : 'N/A'}
		          </div>
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

	        {/* 7. Portfolio Health – performing loans outstanding */}
	        <button
	          type="button"
	          className="collection-card kpi-green"
	          onClick={() => handleCardClick('portfolio-health')}
	        >
	          <div className="card-label">Portfolio Health</div>
	          <div className="card-value">
	            {portfolioHealthAmount != null
	              ? formatCurrency(portfolioHealthAmount)
	              : '—'}
	          </div>
	          <div className="card-subtitle">
	            {portfolioHealthCount != null
	              ? `${portfolioHealthCount.toLocaleString()} performing loans`
	              : 'Count of performing loans in selected portfolio'}
	          </div>
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

	      <div className="branch-leaderboard-section">
	        <div className="branch-leaderboard-header">
	          <div>
	            <h3>Branch Leaderboard</h3>
	            <p className="branch-leaderboard-subtitle">
	              Breakdown of expected due today and collections today by branch.
	            </p>
	          </div>
	          <div className="branch-leaderboard-actions">
	            <button
	              type="button"
	              className={`leaderboard-sort-button ${branchSort.key === 'npl_ratio' ? 'active' : ''}`}
	              onClick={() => handleLeaderboardSort('npl_ratio')}
	            >
	              Sort NPL
	            </button>
	            <button
	              type="button"
	              className={`leaderboard-sort-button ${branchSort.key === 'today_rate' ? 'active' : ''}`}
	              onClick={() => handleLeaderboardSort('today_rate')}
	            >
	              Sort Rate
	            </button>
	          </div>
	        </div>

	        {branchesError && (
	          <div className="collections-error small">
	            Failed to load branch leaderboard: {branchesError}
	          </div>
	        )}

	        <div className="branch-leaderboard-split">
	          <div className="branch-leaderboard-main">
	            <div className="branch-leaderboard-table-wrapper">
	              <table className="branch-leaderboard-table">
	                <thead>
	                  <tr>
	                    <th>Branch</th>
	                    <th>Portfolio</th>
	                    <th>Due</th>
	                    <th>Coll.</th>
	                    <th>Today%</th>
	                    <th>MTD%</th>
	                    <th>Progress</th>
	                    <th>Missed</th>
	                    <th>Perf%</th>
	                    <th>NPL%</th>
	                    <th>Status</th>
	                  </tr>
	                </thead>
	                <tbody>
	                  {loadingBranches ? (
	                    <tr>
	                      <td colSpan={11} className="branch-leaderboard-loading">
	                        Loading branch leaderboard...
	                      </td>
	                    </tr>
	                  ) : sortedBranchLeaderboard.length === 0 ? (
	                    <tr>
	                      <td colSpan={11} className="branch-leaderboard-empty">
	                        No branches found for the selected filters.
	                      </td>
	                    </tr>
	                  ) : (
	                    sortedBranchLeaderboard.map((b) => {
	                      const todayRatePercentage = (typeof b.today_rate === 'number' ? b.today_rate : 0) * 100;
	                      const mtdRatePercentage = (typeof b.mtd_rate === 'number' ? b.mtd_rate : 0) * 100;
	                      const progressRatePercentage = (typeof b.progress_rate === 'number' ? b.progress_rate : 0) * 100;
	                      const nplPercentage = (typeof b.npl_ratio === 'number' ? b.npl_ratio : 0) * 100;
	                      const perfPercentage = 100 - nplPercentage;
	                      const key = `${b.region || 'all'}-${b.branch || 'unknown'}`;

	                      return (
	                        <tr key={key}>
	                          <td>{b.branch || '-'}</td>
	                          <td>{formatCurrency(b.portfolio_total)}</td>
	                          <td>{formatCurrency(b.due_today)}</td>
	                          <td>{formatCurrency(b.collected_today)}</td>
	                          <td>{formatPercent(todayRatePercentage)}</td>
	                          <td>{formatPercent(mtdRatePercentage)}</td>
	                          <td>
	                            <div className="progress-cell">
	                              <div className="progress-bar">
	                                <div
	                                  className="progress-bar-fill"
	                                  style={{ width: `${Math.max(0, Math.min(100, progressRatePercentage))}%` }}
	                                />
	                              </div>
	                              <span className="progress-bar-label">
	                                {formatPercent(progressRatePercentage)}
	                              </span>
	                            </div>
	                          </td>
	                          <td>{formatCurrency(b.missed_today)}</td>
	                          <td>{formatPercent(perfPercentage)}</td>
	                          <td>{formatPercent(nplPercentage)}</td>
	                          <td>
	                            <span className={`status-pill status-${(b.status || 'OK').toLowerCase()}`}>
	                              {b.status || 'OK'}
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

	          <div className="branch-leaderboard-side">
	            <div className="branch-leaderboard-side-content">
	              <div className="branch-leaderboard-side-placeholder">
	                <h4>Agent Activity</h4>
	                <p><strong>Coming soon</strong></p>
	              </div>
	            </div>
	          </div>
	        </div>
	      </div>
    </div>
  );
};

export default CollectionControlCentre;
