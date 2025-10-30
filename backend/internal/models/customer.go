package models

import "time"

// Customer represents a customer
type Customer struct {
	CustomerID       string     `json:"customer_id" db:"customer_id"`
	CustomerName     string     `json:"customer_name" db:"customer_name"`
	CustomerPhone    *string    `json:"customer_phone,omitempty" db:"customer_phone"`
	CustomerEmail    *string    `json:"customer_email,omitempty" db:"customer_email"`
	DateOfBirth      *time.Time `json:"date_of_birth,omitempty" db:"date_of_birth"`
	Gender           *string    `json:"gender,omitempty" db:"gender"`
	State            *string    `json:"state,omitempty" db:"state"`
	LGA              *string    `json:"lga,omitempty" db:"lga"`
	Address          *string    `json:"address,omitempty" db:"address"`
	KYCStatus        *string    `json:"kyc_status,omitempty" db:"kyc_status"`
	KYCVerifiedDate  *time.Time `json:"kyc_verified_date,omitempty" db:"kyc_verified_date"`
	CreatedAt        time.Time  `json:"created_at" db:"created_at"`
	UpdatedAt        time.Time  `json:"updated_at" db:"updated_at"`
}

// CustomerInput represents the input payload for creating/updating a customer
type CustomerInput struct {
	CustomerID      string  `json:"customer_id" binding:"required"`
	CustomerName    string  `json:"customer_name" binding:"required"`
	CustomerPhone   *string `json:"customer_phone"`
	CustomerEmail   *string `json:"customer_email"`
	DateOfBirth     *string `json:"date_of_birth"` // YYYY-MM-DD format
	Gender          *string `json:"gender"`
	State           *string `json:"state"`
	LGA             *string `json:"lga"`
	Address         *string `json:"address"`
	KYCStatus       *string `json:"kyc_status"`
	KYCVerifiedDate *string `json:"kyc_verified_date"` // YYYY-MM-DD format
}

