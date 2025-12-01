import { useState, useEffect } from 'react';
import { Header } from './components/Header';
import { KPIStrip } from './components/KPIStrip';
import { DataTables } from './components/DataTables';
import FIMRDrilldown from './components/FIMRDrilldown';
import EarlyIndicatorsDrilldown from './components/EarlyIndicatorsDrilldown';
import AgentPerformance from './components/AgentPerformance';
import CreditHealthByBranch from './components/CreditHealthByBranch';
import AllLoans from './components/AllLoans';
import CollectionControlCentre from './components/CollectionControlCentre';
import BranchDetail from './components/BranchDetail';
import OfficerDetail from './components/OfficerDetail';
import Login from './components/Login';
import { TabHeader } from './components/Tooltip';
import { formatTabTooltip } from './utils/metricInfo';
import {
  mockOfficers,
  mockPortfolioMetrics,
  mockFIMRLoans,
  mockEarlyIndicatorLoans,
  mockAgentPerformance,
  mockBranchData
} from './utils/mockData';
import apiService from './services/api';
import './App.css';

function App() {
  // Authentication state
  const [isAuthenticated, setIsAuthenticated] = useState(false);
  const [isCheckingAuth, setIsCheckingAuth] = useState(true);

  // Check authentication on mount
  useEffect(() => {
    const checkAuth = () => {
      const authStatus = localStorage.getItem('isAuthenticated');
      setIsAuthenticated(authStatus === 'true');
      setIsCheckingAuth(false);
    };
    checkAuth();
  }, []);

  const [filters, setFilters] = useState({
    dateRange: 'week',
    branch: '',
    regions: [], // Multi-select region filter
    wave: '',
    includeWatch: false,
    dqiCpToggle: true,
    showRedOnly: false,
  });

  const [activeTab, setActiveTab] = useState('performance');
  const [branchSlug, setBranchSlug] = useState(null);
  const [lastRefresh, setLastRefresh] = useState(new Date());
  const [allLoansFilter, setAllLoansFilter] = useState(null);
  const [agentPerformanceFilter, setAgentPerformanceFilter] = useState(null);
	  const [officerDetailContext, setOfficerDetailContext] = useState({
	    officerId: null,
	    officerName: null,
	    branchSlug: null,
	  });
  const [creditHealthFilters, setCreditHealthFilters] = useState({
    branch: '',
    region: '',
    channel: '',
    user_type: '',
    officer_email: '',
  });

  // State for real data from API
  const [portfolioMetrics, setPortfolioMetrics] = useState(mockPortfolioMetrics);
  const [officers, setOfficers] = useState(mockOfficers);
  const [fimrLoans, setFimrLoans] = useState(mockFIMRLoans);
  const [earlyIndicatorLoans, setEarlyIndicatorLoans] = useState(mockEarlyIndicatorLoans);
  const [branches, setBranches] = useState(mockBranchData);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState(null);
  const [useRealData, setUseRealData] = useState(true); // Toggle between real and mock data

  // Fetch data from API
  useEffect(() => {
    const fetchData = async () => {
      if (!useRealData) {
        setLoading(false);
        return;
      }

      try {
        setLoading(true);
        setError(null);

        console.log('🔄 Fetching data from backend API...');

        // Build query params for wave and region filters
        const queryParams = {};
        if (filters.wave) {
          queryParams.wave = filters.wave;
        }
        if (filters.regions && filters.regions.length > 0) {
          queryParams.region = filters.regions.join(',');
        }

        // Build query params for officers with high limit to fetch all
        const officersQueryParams = { ...queryParams, limit: 10000 };

        // Fetch all data in parallel
        const [
          portfolioData,
          officersData,
          fimrData,
          earlyIndicatorData,
          branchesData,
        ] = await Promise.all([
          apiService.fetchPortfolioMetrics(queryParams),
          apiService.fetchOfficers(officersQueryParams),
          apiService.fetchFIMRLoans(queryParams),
          apiService.fetchEarlyIndicatorLoans(queryParams),
          apiService.fetchBranches(queryParams),
        ]);

        console.log('✅ Data fetched successfully:');
        console.log('  - Portfolio Metrics:', portfolioData);
        console.log('  - Officers:', officersData.length);
        console.log('  - FIMR Loans:', fimrData.length);
        console.log('  - Early Indicator Loans:', earlyIndicatorData.length);
        console.log('  - Branches:', branchesData.length);

        const transformedOfficers = officersData.map(o => apiService.transformOfficerData(o));
        console.log('✅ Transformed Officers:', transformedOfficers);

        setPortfolioMetrics(portfolioData);
        setOfficers(transformedOfficers);
        setFimrLoans(fimrData.map(l => apiService.transformFIMRLoan(l)));
        setEarlyIndicatorLoans(earlyIndicatorData.map(l => apiService.transformEarlyIndicatorLoan(l)));
        setBranches(branchesData.map(b => apiService.transformBranchData(b)));

        console.log('✅ Transformed Early Indicator Loans:', earlyIndicatorData.map(l => apiService.transformEarlyIndicatorLoan(l)));

        setLastRefresh(new Date());
      } catch (err) {
        console.error('❌ Error fetching data:', err);
        setError(err.message);
        // Fall back to mock data on error
        setUseRealData(false);
      } finally {
        setLoading(false);
      }
    };

    fetchData();
  }, [useRealData, filters.wave, filters.regions]);

  // Fetch officers when Credit Health filters change
  useEffect(() => {
    const fetchFilteredOfficers = async () => {
      if (!useRealData) return;

      try {
        // Build query params from Credit Health filters
        const queryParams = { limit: 10000 };
        if (creditHealthFilters.branch) queryParams.branch = creditHealthFilters.branch;
        if (creditHealthFilters.region) queryParams.region = creditHealthFilters.region;
        if (creditHealthFilters.channel) queryParams.channel = creditHealthFilters.channel;
        if (creditHealthFilters.user_type) queryParams.user_type = creditHealthFilters.user_type;
        if (creditHealthFilters.officer_email) queryParams.officer_email = creditHealthFilters.officer_email;

        console.log('🔍 Fetching officers with filters:', queryParams);
        const officersData = await apiService.fetchOfficers(queryParams);
        console.log(`✅ Fetched ${officersData.length} officers`);
        const transformedOfficers = officersData.map(o => apiService.transformOfficerData(o));
        setOfficers(transformedOfficers);
      } catch (err) {
        console.error('❌ Error fetching filtered officers:', err);
      }
    };

    fetchFilteredOfficers();
  }, [creditHealthFilters, useRealData]);

  // Filter officers based on current filters
  const filteredOfficers = officers.filter((officer) => {
    if (filters.branch && officer.branch !== filters.branch) return false;
    if (filters.regions && filters.regions.length > 0 && !filters.regions.includes(officer.region)) return false;
    if (filters.showRedOnly && officer.riskScore >= 40) return false;
    return true;
  });

  const handleFilterChange = (newFilters) => {
    setFilters(newFilters);
  };

  const handleCreditHealthFilterChange = (newFilters) => {
    setCreditHealthFilters(newFilters);
  };

  const handleExport = (format) => {
    console.log(`Exporting as ${format}...`);
    // TODO: Implement export functionality
    alert(`Export as ${format} - Coming soon!`);
  };

  const handleRefresh = () => {
    setUseRealData(true);
    setLastRefresh(new Date());
  };

	  const handleViewOfficerPortfolio = (officerId, officerName) => {
	    // Used by Officer Performance tab to drill into All Loans
	    setAllLoansFilter({ officer_id: officerId, officer_name: officerName });
	    setActiveTab('allLoans');
	  };

  const handleViewOfficerLowDelayLoans = (officerId, officerName, delayRate) => {
    console.log('🔵 handleViewOfficerLowDelayLoans called', { officerId, officerName, delayRate });
    // Set the filter for All Loans view with risky delay filter
    setAllLoansFilter({
      officer_id: officerId,
      officer_name: officerName,
      delay_type: 'risky', // Loans with high days_since_last_repayment
      label: `Risky Delay Loans for ${officerName} (Delay Rate: ${delayRate != null ? delayRate.toFixed(2) : 'N/A'}%)`
    });
    // Switch to All Loans tab
    setActiveTab('allLoans');
  };

  // Portfolio Metrics filtering callbacks
  const handleViewActiveLoans = () => {
    console.log('🔵 handleViewActiveLoans called');
    setAllLoansFilter({ loan_type: 'active', label: 'Active Loans' });
    setActiveTab('allLoans');
  };

  const handleViewInactiveLoans = () => {
    console.log('🔵 handleViewInactiveLoans called');
    setAllLoansFilter({ loan_type: 'inactive', label: 'Inactive Loans' });
    setActiveTab('allLoans');
  };

  const handleViewEarlyROT = () => {
    console.log('🔵 handleViewEarlyROT called');
    setAllLoansFilter({ rot_type: 'early', label: 'Early ROT Loans' });
    setActiveTab('allLoans');
  };

  const handleViewLateROT = () => {
    console.log('🔵 handleViewLateROT called');
    setAllLoansFilter({ rot_type: 'late', label: 'Late ROT Loans' });
    setActiveTab('allLoans');
  };

  const handleViewAtRiskOfficers = () => {
    console.log('🔵 handleViewAtRiskOfficers called');
    setActiveTab('agentPerformance');
    // TODO: Add filter for at-risk officers in AgentPerformance component
  };

  const handleViewLowDelayOfficers = () => {
    console.log('🔵 handleViewLowDelayOfficers called');
    setAgentPerformanceFilter({ delayRateMax: 60, label: 'Officers with Delay Rate < 60%' });
    setActiveTab('agentPerformance');
  };

  const handleViewOverdueLoans = () => {
    console.log('🔵 handleViewOverdueLoans called');
    setAllLoansFilter({ loan_type: 'overdue_15d', label: 'Overdue >15 Days Loans' });
    setActiveTab('allLoans');
  };

  // Lightweight URL awareness for routes
	  useEffect(() => {
	    const handleRouteChange = () => {
	      try {
	        const path = window.location?.pathname || '';
	        if (path.startsWith('/collection-control-center/branch/')) {
	          // Officer Detail deep link: /collection-control-center/branch/{branchSlug}/officer/{officerId}
	          const parts = path.split('/').filter(Boolean);
	          const branchIndex = parts.indexOf('branch');
	          const officerIndex = parts.indexOf('officer');
	          const slug =
	            branchIndex >= 0 && branchIndex + 1 < parts.length
	              ? parts[branchIndex + 1]
	              : null;
	          const officerId =
	            officerIndex >= 0 && officerIndex + 1 < parts.length
	              ? parts[officerIndex + 1]
	              : null;
	          if (slug && officerId) {
	            setBranchSlug(slug);
	            setOfficerDetailContext({
	              branchSlug: slug,
	              officerId,
	              officerName: null,
	            });
	            setActiveTab('officerDetail');
	            return;
	          }
	        }
	        if (path.startsWith('/branches/')) {
	          const slug = path.replace('/branches/', '').split('/')[0];
	          setBranchSlug(slug);
	          setActiveTab('branchDetail');
	        } else if (path.includes('/collections/control-centre')) {
	          setBranchSlug(null);
	          setActiveTab('collectionsControlCentre');
	        }
	      } catch (e) {
	        // Ignore in non-browser environments
	      }
	    };

    handleRouteChange();

    // Listen for popstate events (browser back/forward)
    window.addEventListener('popstate', handleRouteChange);
    return () => window.removeEventListener('popstate', handleRouteChange);
  }, []);

  // Navigate to branch detail page
  const navigateToBranch = (slug) => {
    const newPath = `/branches/${slug}`;
    try {
      if (window.history && window.location?.pathname !== newPath) {
        window.history.pushState({}, '', newPath);
      }
    } catch (e) {
      // Ignore history errors
    }
    setBranchSlug(slug);
    setActiveTab('branchDetail');
  };

	  // Navigate from Branch Detail to Officer Detail page
	  const navigateToOfficerDetail = (slug, officerId, officerName) => {
	    if (!slug || !officerId) return;
	    const newPath = `/collection-control-center/branch/${slug}/officer/${officerId}`;
	    try {
	      if (window.history && window.location?.pathname !== newPath) {
	        window.history.pushState({}, '', newPath);
	      }
	    } catch (e) {
	      // Ignore history errors
	    }
	    setBranchSlug(slug);
	    setOfficerDetailContext({ branchSlug: slug, officerId, officerName });
	    setActiveTab('officerDetail');
	  };

  // Navigate back from branch detail to Collections Control Centre
  const navigateBackToCollections = () => {
    const newPath = '/collections/control-centre';
    try {
      if (window.history && window.location?.pathname !== newPath) {
        window.history.pushState({}, '', newPath);
      }
    } catch (e) {
      // Ignore history errors
    }
    setBranchSlug(null);
    setActiveTab('collectionsControlCentre');
  };

  // Handle login
  const handleLogin = () => {
    setIsAuthenticated(true);
  };

  // Handle logout
  const handleLogout = () => {
    localStorage.removeItem('isAuthenticated');
    localStorage.removeItem('username');
    localStorage.removeItem('loginTime');
    setIsAuthenticated(false);
  };

  // Show loading while checking authentication
  if (isCheckingAuth) {
    return (
      <div className="app">
        <div style={{ display: 'flex', justifyContent: 'center', alignItems: 'center', height: '100vh' }}>
          <div>Loading...</div>
        </div>
      </div>
    );
  }

  // Show login if not authenticated
  if (!isAuthenticated) {
    return <Login onLogin={handleLogin} />;
  }

  if (loading) {
    return (
      <div className="app">
        <div style={{ display: 'flex', justifyContent: 'center', alignItems: 'center', height: '100vh' }}>
          <div>Loading data from backend...</div>
        </div>
      </div>
    );
  }

  return (
    <div className="app">
      {error && (
        <div style={{
          background: '#fff3cd',
          border: '1px solid #ffc107',
          padding: '10px',
          margin: '10px',
          borderRadius: '4px',
          color: '#856404'
        }}>
          ⚠️ Error loading data: {error}. Using mock data instead.
        </div>
      )}

      <Header
        filters={filters}
        onFilterChange={handleFilterChange}
        onExport={handleExport}
        lastRefresh={lastRefresh}
        branches={branches}
        onLogout={handleLogout}
      />

      <KPIStrip
        portfolioMetrics={portfolioMetrics}
        onViewOverdueLoans={handleViewOverdueLoans}
        onViewActiveLoans={handleViewActiveLoans}
        onViewInactiveLoans={handleViewInactiveLoans}
        onViewEarlyROT={handleViewEarlyROT}
        onViewLateROT={handleViewLateROT}
        onViewAtRiskOfficers={handleViewAtRiskOfficers}
        onViewLowDelayOfficers={handleViewLowDelayOfficers}
      />

      <div className="main-content">
        <div className="tabs">
          <button
            className={`tab ${activeTab === 'creditHealth' ? 'active' : ''}`}
            onClick={() => setActiveTab('creditHealth')}
            title={formatTabTooltip('creditHealth')}
          >
            <TabHeader
              label="Credit Health Overview"
              tabKey="creditHealth"
              info={formatTabTooltip('creditHealth')}
            />
          </button>
          <button
            className={`tab ${activeTab === 'performance' ? 'active' : ''}`}
            onClick={() => setActiveTab('performance')}
            title={formatTabTooltip('performance')}
          >
            <TabHeader
              label="Officer Performance"
              tabKey="performance"
              info={formatTabTooltip('performance')}
            />
          </button>
          <button
            className={`tab ${activeTab === 'earlyIndicators' ? 'active' : ''}`}
            onClick={() => setActiveTab('earlyIndicators')}
            title={formatTabTooltip('earlyIndicators')}
          >
            <TabHeader
              label="Early Indicators"
              tabKey="earlyIndicators"
              info={formatTabTooltip('earlyIndicators')}
            />
          </button>
          <button
            className={`tab ${activeTab === 'fimrDrilldown' ? 'active' : ''}`}
            onClick={() => setActiveTab('fimrDrilldown')}
            title={formatTabTooltip('fimrDrilldown')}
          >
            <TabHeader
              label="FIMR Drilldown"
              tabKey="fimrDrilldown"
              info={formatTabTooltip('fimrDrilldown')}
            />
          </button>
          <button
            className={`tab ${activeTab === 'earlyIndicatorsDrilldown' ? 'active' : ''}`}
            onClick={() => setActiveTab('earlyIndicatorsDrilldown')}
            title={formatTabTooltip('earlyIndicatorsDrilldown')}
          >
            <TabHeader
              label="Early Indicators Drilldown"
              tabKey="earlyIndicatorsDrilldown"
              info={formatTabTooltip('earlyIndicatorsDrilldown')}
            />
          </button>
          <button
            className={`tab ${activeTab === 'agentPerformance' ? 'active' : ''}`}
            onClick={() => setActiveTab('agentPerformance')}
            title={formatTabTooltip('agentPerformance')}
          >
            <TabHeader
              label="Agent Performance"
              tabKey="agentPerformance"
              info={formatTabTooltip('agentPerformance')}
            />
          </button>
          <button
            className={`tab ${activeTab === 'creditHealthByBranch' ? 'active' : ''}`}
            onClick={() => setActiveTab('creditHealthByBranch')}
            title="Credit Health Overview aggregated by Branch"
          >
            <TabHeader
              label="Credit Health by Branch"
              tabKey="creditHealthByBranch"
              info="Credit Health Overview aggregated by Branch"
            />
          </button>
	          <button
	            className={`tab ${activeTab === 'collectionsControlCentre' ? 'active' : ''}`}
	            onClick={() => {
	              setActiveTab('collectionsControlCentre');
	              try {
	                const newPath = '/collections/control-centre';
	                if (window.history && window.location?.pathname !== newPath) {
	                  window.history.pushState({}, '', newPath);
	                }
	              } catch (e) {
	                // Ignore history errors
	              }
	            }}
	            title="Collections-focused control centre"
	          >
	            <TabHeader
	              label="Collections Control Centre"
	              tabKey="collectionsControlCentre"
	              info="Collections performance view using loans and repayments data"
	            />
	          </button>
          <button
            className={`tab ${activeTab === 'allLoans' ? 'active' : ''}`}
            onClick={() => setActiveTab('allLoans')}
            title="View all loans in the database"
          >
            <TabHeader
              label="All Loans"
              tabKey="allLoans"
              info="View all loans in the database with filtering and export capabilities"
            />
          </button>
        </div>

		        <div className="tab-content">
	          {activeTab === 'fimrDrilldown' ? (
	            <FIMRDrilldown loans={fimrLoans} />
	          ) : activeTab === 'earlyIndicatorsDrilldown' ? (
	            <EarlyIndicatorsDrilldown loans={earlyIndicatorLoans} />
	          ) : activeTab === 'agentPerformance' ? (
	            <AgentPerformance
	              key={JSON.stringify(agentPerformanceFilter)}
	              agents={officers}
	              onViewPortfolio={handleViewOfficerPortfolio}
	              onViewLowDelayLoans={handleViewOfficerLowDelayLoans}
	              initialFilter={agentPerformanceFilter}
	            />
	          ) : activeTab === 'creditHealthByBranch' ? (
	            <CreditHealthByBranch branches={branches} />
	          ) : activeTab === 'collectionsControlCentre' ? (
	            <CollectionControlCentre onNavigateToBranch={navigateToBranch} />
	          ) : activeTab === 'branchDetail' ? (
	            <BranchDetail
	              branchSlug={branchSlug}
	              onBack={navigateBackToCollections}
		              onViewOfficerLoans={(officerId, officerName) =>
		                navigateToOfficerDetail(branchSlug, officerId, officerName)
		              }
	            />
		          ) : activeTab === 'officerDetail' ? (
		            <OfficerDetail
		              branchSlug={officerDetailContext.branchSlug || branchSlug}
		              officerId={officerDetailContext.officerId}
		              officerName={officerDetailContext.officerName}
		              onBackToOverview={navigateBackToCollections}
		              onBackToBranch={() => {
		                const slug = officerDetailContext.branchSlug || branchSlug;
		                if (slug) {
		                  navigateToBranch(slug);
		                } else {
		                  navigateBackToCollections();
		                }
		              }}
		            />
	          ) : activeTab === 'allLoans' ? (
	            <AllLoans
	              key={JSON.stringify(allLoansFilter)}
	              initialFilter={allLoansFilter}
	            />
	          ) : (
	            <DataTables
	              officers={filteredOfficers}
	              activeTab={activeTab}
	              onFilterChange={handleCreditHealthFilterChange}
	            />
	          )}
	        </div>
      </div>
    </div>
  );
}

export default App;
