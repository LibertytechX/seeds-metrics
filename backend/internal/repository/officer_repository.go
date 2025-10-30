package repository

import (
	"context"
	"fmt"
	"time"

	"github.com/seeds-metrics/analytics-backend/internal/models"
	"github.com/seeds-metrics/analytics-backend/pkg/database"
)

type OfficerRepository struct {
	db *database.DB
}

func NewOfficerRepository(db *database.DB) *OfficerRepository {
	return &OfficerRepository{db: db}
}

// Create inserts a new officer
func (r *OfficerRepository) Create(ctx context.Context, input *models.OfficerInput) error {
	query := `
		INSERT INTO officers (
			officer_id, officer_name, officer_phone,
			region, branch, employment_status, hire_date,
			created_at, updated_at
		) VALUES (
			$1, $2, $3, $4, $5, $6, $7, NOW(), NOW()
		)
		ON CONFLICT (officer_id) DO UPDATE SET
			officer_name = EXCLUDED.officer_name,
			officer_phone = EXCLUDED.officer_phone,
			region = EXCLUDED.region,
			branch = EXCLUDED.branch,
			employment_status = EXCLUDED.employment_status,
			hire_date = EXCLUDED.hire_date,
			updated_at = NOW()
	`

	// Set default employment status if not provided
	employmentStatus := "Active"
	if input.EmploymentStatus != "" {
		employmentStatus = input.EmploymentStatus
	}

	// Parse hire date if provided
	var hireDate *time.Time
	if input.HireDate != nil {
		parsed, err := time.Parse("2006-01-02", *input.HireDate)
		if err != nil {
			return fmt.Errorf("invalid hire_date format: %w", err)
		}
		hireDate = &parsed
	}

	_, err := r.db.ExecContext(ctx, query,
		input.OfficerID, input.OfficerName, input.OfficerPhone,
		input.Region, input.Branch, employmentStatus, hireDate,
	)

	return err
}

// GetByID retrieves an officer by ID
func (r *OfficerRepository) GetByID(ctx context.Context, officerID string) (*models.Officer, error) {
	query := `
		SELECT
			officer_id, officer_name, officer_phone,
			region, branch, employment_status, hire_date,
			created_at, updated_at
		FROM officers
		WHERE officer_id = $1
	`

	var officer models.Officer
	err := r.db.QueryRowContext(ctx, query, officerID).Scan(
		&officer.OfficerID, &officer.OfficerName, &officer.OfficerPhone,
		&officer.Region, &officer.Branch, &officer.EmploymentStatus, &officer.HireDate,
		&officer.CreatedAt, &officer.UpdatedAt,
	)

	if err != nil {
		return nil, err
	}

	return &officer, nil
}

// List retrieves all officers
func (r *OfficerRepository) List(ctx context.Context) ([]*models.Officer, error) {
	query := `
		SELECT
			officer_id, officer_name, officer_phone,
			region, branch, employment_status, hire_date,
			created_at, updated_at
		FROM officers
		ORDER BY officer_name
	`

	rows, err := r.db.QueryContext(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var officers []*models.Officer
	for rows.Next() {
		var officer models.Officer
		err := rows.Scan(
			&officer.OfficerID, &officer.OfficerName, &officer.OfficerPhone,
			&officer.Region, &officer.Branch, &officer.EmploymentStatus, &officer.HireDate,
			&officer.CreatedAt, &officer.UpdatedAt,
		)
		if err != nil {
			return nil, err
		}
		officers = append(officers, &officer)
	}

	return officers, nil
}
