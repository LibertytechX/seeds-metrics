// API Service for Analytics Backend

const API_BASE_URL = import.meta.env.VITE_API_URL || 'http://localhost:8080/api/v1';

class ApiService {
  async fetchPortfolioMetrics(params = {}) {
    try {
      const queryParams = new URLSearchParams(params);
      const response = await fetch(`${API_BASE_URL}/metrics/portfolio?${queryParams}`);
      const data = await response.json();

      if (data.status === 'success') {
        return data.data;
      }
      throw new Error(data.message || 'Failed to fetch portfolio metrics');
    } catch (error) {
      console.error('Error fetching portfolio metrics:', error);
      throw error;
    }
  }

  async fetchOfficers(params = {}) {
    try {
      const queryParams = new URLSearchParams(params);
      const response = await fetch(`${API_BASE_URL}/officers?${queryParams}`);
      const data = await response.json();

      if (data.status === 'success') {
        return data.data.officers || [];
      }
      throw new Error(data.message || 'Failed to fetch officers');
    } catch (error) {
      console.error('Error fetching officers:', error);
      throw error;
    }
  }

  async fetchOfficerById(officerId) {
    try {
      const response = await fetch(`${API_BASE_URL}/officers/${officerId}`);
      const data = await response.json();

      if (data.status === 'success') {
        return data.data;
      }
      throw new Error(data.message || 'Failed to fetch officer');
    } catch (error) {
      console.error('Error fetching officer:', error);
      throw error;
    }
  }

  async fetchFIMRLoans(params = {}) {
    try {
      const queryParams = new URLSearchParams(params);
      const response = await fetch(`${API_BASE_URL}/fimr/loans?${queryParams}`);
      const data = await response.json();

      if (data.status === 'success') {
        return data.data.loans || [];
      }
      throw new Error(data.message || 'Failed to fetch FIMR loans');
    } catch (error) {
      console.error('Error fetching FIMR loans:', error);
      throw error;
    }
  }

  async fetchEarlyIndicatorLoans(params = {}) {
    try {
      const queryParams = new URLSearchParams(params);
      const response = await fetch(`${API_BASE_URL}/early-indicators/loans?${queryParams}`);
      const data = await response.json();

      if (data.status === 'success') {
        return data.data.loans || [];
      }
      throw new Error(data.message || 'Failed to fetch early indicator loans');
    } catch (error) {
      console.error('Error fetching early indicator loans:', error);
      throw error;
    }
  }

  async fetchBranches(params = {}) {
    try {
      const queryParams = new URLSearchParams(params);
      const response = await fetch(`${API_BASE_URL}/branches?${queryParams}`);
      const data = await response.json();

      if (data.status === 'success') {
        return data.data.branches || [];
      }
      throw new Error(data.message || 'Failed to fetch branches');
    } catch (error) {
      console.error('Error fetching branches:', error);
      throw error;
    }
  }

  async fetchTeamMembers() {
    try {
      const response = await fetch(`${API_BASE_URL}/team-members`);
      const data = await response.json();

      if (data.status === 'success') {
        return data.data || [];
      }
      throw new Error(data.message || 'Failed to fetch team members');
    } catch (error) {
      console.error('Error fetching team members:', error);
      throw error;
    }
  }

  // Transform backend data to frontend format
  transformOfficerData(officer) {
    return {
      id: officer.id,
      officerId: officer.officer_id,
      name: officer.name,
      officerName: officer.name, // AgentPerformance component uses officerName
      region: officer.region,
      branch: officer.branch,
      channel: officer.channel,
      riskScore: officer.calculatedMetrics?.riskScore || 0,
      riskBand: officer.riskBand || 'Green',
      fimr: officer.calculatedMetrics?.fimr || 0,
      slippage: officer.calculatedMetrics?.slippage || 0,
      d06Slippage: officer.calculatedMetrics?.slippage || 0, // AgentPerformance uses d06Slippage
      roll: officer.calculatedMetrics?.roll || 0,
      frr: officer.calculatedMetrics?.frr || 0,
      ayr: officer.calculatedMetrics?.ayr || 0,
      dqi: officer.calculatedMetrics?.dqi || 0,
      // NEW: Repayment behavior metrics
      avgTimelinessScore: officer.calculatedMetrics?.avgTimelinessScore || null,
      avgRepaymentHealth: officer.calculatedMetrics?.avgRepaymentHealth || null,
      avgDaysSinceLastRepayment: officer.calculatedMetrics?.avgDaysSinceLastRepayment || null,
      avgLoanAge: officer.calculatedMetrics?.avgLoanAge || null,
      repaymentDelayRate: officer.calculatedMetrics?.repaymentDelayRate || null,
      yield: officer.calculatedMetrics?.yield || 0,
      overdue15dVolume: officer.calculatedMetrics?.overdue15dVolume || 0,
      onTimeRate: officer.calculatedMetrics?.onTimeRate || 0,
      channelPurity: officer.calculatedMetrics?.channelPurity || 0,
      porr: officer.calculatedMetrics?.porr || 0,
      // Add raw metrics fields needed by CreditHealthTable and AgentPerformance
      totalPortfolio: officer.rawMetrics?.totalPortfolio || 0,
      portfolioTotal: officer.rawMetrics?.totalPortfolio || 0, // AgentPerformance uses portfolioTotal
      overdue15d: officer.rawMetrics?.overdue15d || 0,
      activeLoans: officer.rawMetrics?.disbursed || 0, // Use disbursed count as active loans
      // Add audit-related fields with defaults
      assignee: officer.assignee || 'Unassigned',
      auditStatus: officer.auditStatus || 'Assigned',
      lastAuditDate: officer.lastAuditDate || null,
      allTimeFimr: officer.calculatedMetrics?.fimr || 0, // Use current FIMR as all-time FIMR for now
      rank: officer.rank || 0,
    };
  }

  transformFIMRLoan(loan) {
    return {
      loanId: loan.loan_id,
      customerName: loan.customer_name,
      customerPhone: loan.customer_phone,
      officerName: loan.officer_name,
      officerId: loan.officer_id,
      region: loan.region,
      branch: loan.branch,
      channel: loan.channel,
      loanAmount: loan.loan_amount,
      disbursementDate: loan.disbursement_date?.split('T')[0] || loan.disbursement_date,
      firstPaymentDueDate: loan.first_payment_due_date?.split('T')[0] || loan.first_payment_due_date,
      daysSinceDue: loan.days_since_due, // Match mock data format
      amountDue1stInstallment: loan.amount_due_1st_installment,
      amountPaid: loan.amount_paid,
      outstandingBalance: loan.outstanding_balance, // Match mock data format
      currentDPD: loan.current_dpd,
      status: loan.status,
      fimrTagged: loan.fimr_tagged,
    };
  }

  transformEarlyIndicatorLoan(loan) {
    return {
      loanId: loan.loan_id,
      customerName: loan.customer_name,
      customerPhone: loan.customer_phone,
      officerName: loan.officer_name,
      officerId: loan.officer_id,
      region: loan.region,
      branch: loan.branch,
      channel: loan.channel,
      loanAmount: loan.loan_amount,
      disbursementDate: loan.disbursement_date?.split('T')[0] || loan.disbursement_date,
      currentDPD: loan.current_dpd,
      previousDPDStatus: loan.previous_dpd_status || 'Current',
      daysInCurrentStatus: loan.days_in_current_status || 0,
      amountDue: loan.amount_due,
      amountPaid: loan.amount_paid,
      outstandingBalance: loan.outstanding_balance, // Match mock data format
      status: loan.status,
      fimrTagged: loan.fimr_tagged,
      rollDirection: loan.roll_direction || 'Stable',
      lastPaymentDate: loan.last_payment_date?.split('T')[0] || loan.last_payment_date,
    };
  }

  transformBranchData(branch) {
    return {
      branch: branch.branch,
      region: branch.region,
      portfolioTotal: branch.portfolio_total,
      overdue15d: branch.overdue_15d,
      par15Ratio: branch.par15_ratio,
      ayr: branch.ayr,
      dqi: branch.dqi,
      fimr: branch.fimr,
      activeLoans: branch.active_loans,
      totalOfficers: branch.total_officers,
    };
  }
}

export const apiService = new ApiService();
export default apiService;

