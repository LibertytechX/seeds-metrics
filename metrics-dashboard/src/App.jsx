import { useState, useEffect } from 'react';
import { Header } from './components/Header';
import { KPIStrip } from './components/KPIStrip';
import { DataTables } from './components/DataTables';
import FIMRDrilldown from './components/FIMRDrilldown';
import EarlyIndicatorsDrilldown from './components/EarlyIndicatorsDrilldown';
import AgentPerformance from './components/AgentPerformance';
import CreditHealthByBranch from './components/CreditHealthByBranch';
import AllLoans from './components/AllLoans';
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
  const [filters, setFilters] = useState({
    dateRange: 'week',
    branch: '',
    includeWatch: false,
    dqiCpToggle: true,
    showRedOnly: false,
  });

  const [activeTab, setActiveTab] = useState('performance');
  const [lastRefresh, setLastRefresh] = useState(new Date());

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

        // Fetch all data in parallel
        const [
          portfolioData,
          officersData,
          fimrData,
          earlyIndicatorData,
          branchesData,
        ] = await Promise.all([
          apiService.fetchPortfolioMetrics(),
          apiService.fetchOfficers(),
          apiService.fetchFIMRLoans(),
          apiService.fetchEarlyIndicatorLoans(),
          apiService.fetchBranches(),
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
  }, [useRealData]);

  // Filter officers based on current filters
  const filteredOfficers = officers.filter((officer) => {
    if (filters.branch && officer.branch !== filters.branch) return false;
    if (filters.showRedOnly && officer.riskScore >= 40) return false;
    return true;
  });

  const handleFilterChange = (newFilters) => {
    setFilters(newFilters);
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
      />

      <KPIStrip portfolioMetrics={portfolioMetrics} />

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
            <AgentPerformance agents={officers} />
          ) : activeTab === 'creditHealthByBranch' ? (
            <CreditHealthByBranch branches={branches} />
          ) : activeTab === 'allLoans' ? (
            <AllLoans />
          ) : (
            <DataTables officers={filteredOfficers} activeTab={activeTab} />
          )}
        </div>
      </div>
    </div>
  );
}

export default App;
