package handlers

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/seeds-metrics/analytics-backend/internal/models"
	"github.com/seeds-metrics/analytics-backend/internal/repository"
	"github.com/seeds-metrics/analytics-backend/pkg/database"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// MockDashboardRepository is a mock implementation of DashboardRepository
type MockDashboardRepository struct {
	mock.Mock
}

func (m *MockDashboardRepository) GetBranches(filters map[string]interface{}) ([]*models.DashboardBranchMetrics, error) {
	args := m.Called(filters)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*models.DashboardBranchMetrics), args.Error(1)
}

// TestGetBranchesWithoutFilters tests the GetBranches endpoint without any filters
func TestGetBranchesWithoutFilters(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockRepo := new(MockDashboardRepository)
	mockBranches := []*models.DashboardBranchMetrics{
		{
			Branch:         "Lekki",
			Region:         "Lagos",
			PortfolioTotal: 1000000,
			Overdue15d:     50000,
			Par15Ratio:     0.05,
			ActiveLoans:    100,
			TotalOfficers:  5,
		},
	}

	mockRepo.On("GetBranches", mock.MatchedBy(func(filters map[string]interface{}) bool {
		return len(filters) == 0
	})).Return(mockBranches, nil)

	handler := &DashboardHandler{
		dashboardRepo: mockRepo,
	}

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/branches", nil)
	c, _ := gin.CreateTestContext(w)
	c.Request = req

	handler.GetBranches(c)

	assert.Equal(t, http.StatusOK, w.Code)
	mockRepo.AssertCalled(t, "GetBranches", mock.MatchedBy(func(filters map[string]interface{}) bool {
		return len(filters) == 0
	}))
}

// TestGetBranchesWithRegionFilter tests the GetBranches endpoint with region filter
func TestGetBranchesWithRegionFilter(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockRepo := new(MockDashboardRepository)
	mockBranches := []*models.DashboardBranchMetrics{
		{
			Branch:         "Lekki",
			Region:         "Lagos",
			PortfolioTotal: 1000000,
			Overdue15d:     50000,
			Par15Ratio:     0.05,
			ActiveLoans:    100,
			TotalOfficers:  5,
		},
	}

	mockRepo.On("GetBranches", mock.MatchedBy(func(filters map[string]interface{}) bool {
		region, ok := filters["region"].(string)
		return ok && region == "Lagos"
	})).Return(mockBranches, nil)

	handler := &DashboardHandler{
		dashboardRepo: mockRepo,
	}

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/branches?region=Lagos", nil)
	c, _ := gin.CreateTestContext(w)
	c.Request = req

	handler.GetBranches(c)

	assert.Equal(t, http.StatusOK, w.Code)
	mockRepo.AssertCalled(t, "GetBranches", mock.MatchedBy(func(filters map[string]interface{}) bool {
		region, ok := filters["region"].(string)
		return ok && region == "Lagos"
	}))
}

// TestGetBranchesWithBranchFilter tests the GetBranches endpoint with branch filter
func TestGetBranchesWithBranchFilter(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockRepo := new(MockDashboardRepository)
	mockBranches := []*models.DashboardBranchMetrics{
		{
			Branch:         "Lekki",
			Region:         "Lagos",
			PortfolioTotal: 1000000,
			Overdue15d:     50000,
			Par15Ratio:     0.05,
			ActiveLoans:    100,
			TotalOfficers:  5,
		},
	}

	mockRepo.On("GetBranches", mock.MatchedBy(func(filters map[string]interface{}) bool {
		branch, ok := filters["branch"].(string)
		return ok && branch == "Lekki"
	})).Return(mockBranches, nil)

	handler := &DashboardHandler{
		dashboardRepo: mockRepo,
	}

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/branches?branch=Lekki", nil)
	c, _ := gin.CreateTestContext(w)
	c.Request = req

	handler.GetBranches(c)

	assert.Equal(t, http.StatusOK, w.Code)
	mockRepo.AssertCalled(t, "GetBranches", mock.MatchedBy(func(filters map[string]interface{}) bool {
		branch, ok := filters["branch"].(string)
		return ok && branch == "Lekki"
	}))
}

// TestGetBranchesWithChannelFilter tests the GetBranches endpoint with channel filter
func TestGetBranchesWithChannelFilter(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockRepo := new(MockDashboardRepository)
	mockBranches := []*models.DashboardBranchMetrics{}

	mockRepo.On("GetBranches", mock.MatchedBy(func(filters map[string]interface{}) bool {
		channel, ok := filters["channel"].(string)
		return ok && channel == "AGENT"
	})).Return(mockBranches, nil)

	handler := &DashboardHandler{
		dashboardRepo: mockRepo,
	}

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/branches?channel=AGENT", nil)
	c, _ := gin.CreateTestContext(w)
	c.Request = req

	handler.GetBranches(c)

	assert.Equal(t, http.StatusOK, w.Code)
	mockRepo.AssertCalled(t, "GetBranches", mock.MatchedBy(func(filters map[string]interface{}) bool {
		channel, ok := filters["channel"].(string)
		return ok && channel == "AGENT"
	}))
}

// TestGetBranchesWithUserTypeFilter tests the GetBranches endpoint with user_type filter
func TestGetBranchesWithUserTypeFilter(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockRepo := new(MockDashboardRepository)
	mockBranches := []*models.DashboardBranchMetrics{}

	mockRepo.On("GetBranches", mock.MatchedBy(func(filters map[string]interface{}) bool {
		userType, ok := filters["user_type"].(string)
		return ok && userType == "AGENT"
	})).Return(mockBranches, nil)

	handler := &DashboardHandler{
		dashboardRepo: mockRepo,
	}

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/branches?user_type=AGENT", nil)
	c, _ := gin.CreateTestContext(w)
	c.Request = req

	handler.GetBranches(c)

	assert.Equal(t, http.StatusOK, w.Code)
	mockRepo.AssertCalled(t, "GetBranches", mock.MatchedBy(func(filters map[string]interface{}) bool {
		userType, ok := filters["user_type"].(string)
		return ok && userType == "AGENT"
	}))
}

// TestGetBranchesWithMultipleFilters tests the GetBranches endpoint with multiple filters
func TestGetBranchesWithMultipleFilters(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockRepo := new(MockDashboardRepository)
	mockBranches := []*models.DashboardBranchMetrics{}

	mockRepo.On("GetBranches", mock.MatchedBy(func(filters map[string]interface{}) bool {
		region, regionOk := filters["region"].(string)
		channel, channelOk := filters["channel"].(string)
		return regionOk && region == "Lagos" && channelOk && channel == "AGENT"
	})).Return(mockBranches, nil)

	handler := &DashboardHandler{
		dashboardRepo: mockRepo,
	}

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/branches?region=Lagos&channel=AGENT", nil)
	c, _ := gin.CreateTestContext(w)
	c.Request = req

	handler.GetBranches(c)

	assert.Equal(t, http.StatusOK, w.Code)
	mockRepo.AssertCalled(t, "GetBranches", mock.MatchedBy(func(filters map[string]interface{}) bool {
		region, regionOk := filters["region"].(string)
		channel, channelOk := filters["channel"].(string)
		return regionOk && region == "Lagos" && channelOk && channel == "AGENT"
	}))
}

