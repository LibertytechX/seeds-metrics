import { useState } from 'react';
import { Header } from './components/Header';
import { KPIStrip } from './components/KPIStrip';
import { DataTables } from './components/DataTables';
import FIMRDrilldown from './components/FIMRDrilldown';
import EarlyIndicatorsDrilldown from './components/EarlyIndicatorsDrilldown';
import AgentPerformance from './components/AgentPerformance';
import CreditHealthByBranch from './components/CreditHealthByBranch';
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
  const [lastRefresh] = useState(new Date());

  // Filter officers based on current filters
  const filteredOfficers = mockOfficers.filter((officer) => {
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

  return (
    <div className="app">
      <Header
        filters={filters}
        onFilterChange={handleFilterChange}
        onExport={handleExport}
        lastRefresh={lastRefresh}
      />

      <KPIStrip portfolioMetrics={mockPortfolioMetrics} />

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
        </div>

        <div className="tab-content">
          {activeTab === 'fimrDrilldown' ? (
            <FIMRDrilldown loans={mockFIMRLoans} />
          ) : activeTab === 'earlyIndicatorsDrilldown' ? (
            <EarlyIndicatorsDrilldown loans={mockEarlyIndicatorLoans} />
          ) : activeTab === 'agentPerformance' ? (
            <AgentPerformance agents={mockAgentPerformance} />
          ) : activeTab === 'creditHealthByBranch' ? (
            <CreditHealthByBranch branches={mockBranchData} />
          ) : (
            <DataTables officers={filteredOfficers} activeTab={activeTab} />
          )}
        </div>
      </div>
    </div>
  );
}

export default App;
