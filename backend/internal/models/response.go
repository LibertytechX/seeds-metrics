package models

import "time"

// APIResponse represents a standard API response
type APIResponse struct {
	Status  string      `json:"status"`
	Message string      `json:"message,omitempty"`
	Data    interface{} `json:"data,omitempty"`
	Error   *APIError   `json:"error,omitempty"`
}

// APIError represents an error response
type APIError struct {
	Code    string                 `json:"code"`
	Message string                 `json:"message"`
	Details map[string]interface{} `json:"details,omitempty"`
}

// PaginatedResponse represents a paginated API response
type PaginatedResponse struct {
	Status     string      `json:"status"`
	Data       interface{} `json:"data"`
	Pagination Pagination  `json:"pagination"`
}

// Pagination represents pagination metadata
type Pagination struct {
	Total      int `json:"total"`
	Page       int `json:"page"`
	PageSize   int `json:"page_size"`
	TotalPages int `json:"total_pages"`
}

// ETLSyncRequest represents a batch sync request
type ETLSyncRequest struct {
	SyncTimestamp time.Time       `json:"sync_timestamp" binding:"required"`
	SyncType      string          `json:"sync_type" binding:"required"` // "incremental" or "full"
	Data          ETLSyncData     `json:"data" binding:"required"`
	Metadata      ETLSyncMetadata `json:"metadata"`
}

// ETLSyncData represents the data payload in a sync request
type ETLSyncData struct {
	Loans      []LoanInput      `json:"loans"`
	Repayments []RepaymentInput `json:"repayments"`
}

// ETLSyncMetadata represents metadata in a sync request
type ETLSyncMetadata struct {
	TotalLoans      int    `json:"total_loans"`
	TotalRepayments int    `json:"total_repayments"`
	SourceSystem    string `json:"source_system"`
	ETLVersion      string `json:"etl_version"`
}

// ETLSyncResponse represents the response to a sync request
type ETLSyncResponse struct {
	Status                 string                 `json:"status"`
	SyncID                 string                 `json:"sync_id"`
	Timestamp              time.Time              `json:"timestamp"`
	Results                ETLSyncResults         `json:"results"`
	ComputedFieldsUpdated  ComputedFieldsUpdate   `json:"computed_fields_updated"`
	NextSyncRecommended    time.Time              `json:"next_sync_recommended"`
	Errors                 []ETLSyncError         `json:"errors,omitempty"`
}

// ETLSyncResults represents the results of a sync operation
type ETLSyncResults struct {
	Loans      ETLEntityResult `json:"loans"`
	Repayments ETLEntityResult `json:"repayments"`
}

// ETLEntityResult represents the result for a specific entity type
type ETLEntityResult struct {
	Inserted int `json:"inserted"`
	Updated  int `json:"updated"`
	Failed   int `json:"failed"`
}

// ComputedFieldsUpdate represents information about computed field updates
type ComputedFieldsUpdate struct {
	LoansAffected      int `json:"loans_affected"`
	ComputationTimeMs  int `json:"computation_time_ms"`
}

// ETLSyncError represents an error during sync
type ETLSyncError struct {
	EntityType   string `json:"entity_type"`
	EntityID     string `json:"entity_id"`
	ErrorCode    string `json:"error_code"`
	ErrorMessage string `json:"error_message"`
	Field        string `json:"field,omitempty"`
}

// HealthCheckResponse represents the health check response
type HealthCheckResponse struct {
	Status    string                 `json:"status"`
	Timestamp time.Time              `json:"timestamp"`
	Services  map[string]ServiceHealth `json:"services"`
}

// ServiceHealth represents the health status of a service
type ServiceHealth struct {
	Status  string `json:"status"`
	Message string `json:"message,omitempty"`
}

