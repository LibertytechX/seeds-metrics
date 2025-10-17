import { useState } from 'react';
import { Header } from './components/Header';
import { KPIStrip } from './components/KPIStrip';
import { DataTables } from './components/DataTables';
import { TabHeader } from './components/Tooltip';
import { formatTabTooltip } from './utils/metricInfo';
import { mockOfficers, mockPortfolioMetrics } from './utils/mockData';
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
        </div>

        <div className="tab-content">
          <DataTables officers={filteredOfficers} activeTab={activeTab} />
        </div>
      </div>
    </div>
  );
}

export default App;
