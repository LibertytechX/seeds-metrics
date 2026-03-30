# Seeds Metrics Codebase Index

This document provides a high-level overview of the `seeds-metrics` project structure, its core components, and where to find key documentation.

## Project Overview
The `seeds-metrics` project is a fully-featured Loan Officer Metrics Dashboard with a React frontend and a Go-based Analytics Backend. It is designed to track 7 core metrics for loan officers (e.g., FIMR, D0-6 Slippage, FRR, AYR, DQI, Risk Score) and provide interactive dashboards with colour-coded risk indicators.

## Directory Structure

### `/metrics-dashboard` (Frontend)
A React 18 application built with Vite and styled with CSS3. It provides the UI for the Loan Officer Metrics Dashboard.
* `src/components/`: Contains UI components like `Header.jsx`, `KPIStrip.jsx`, and `DataTables.jsx`.
* `src/utils/`: Contains metric calculation logic (`metrics.js`) and mock data (`mockData.js`).
* **Key Documentation**:
  * `README.md`: Overview of the frontend project and its features.
  * `README_DASHBOARD.md`: Complete feature documentation.
  * `IMPLEMENTATION_SUMMARY.md`: Technical architecture of the frontend.
  * `QUICK_START.md`: 2-minute setup guide.

### `/backend` (Analytics Backend)
A high-performance RESTful API service built with Go (Golang) designed for ETL data ingestion and metric calculations. It uses PostgreSQL for data storage and triggers for auto-computing fields, with Redis for caching.
* `cmd/api/`: Main application entry point (`main.go`).
* `internal/`: Contains core business logic, handlers, models, and database access layer (`repository`).
* `pkg/`: Shared utilities, database, and cache connections.
* `migrations/`: SQL migration scripts.
* `scripts/`: Helper scripts for testing and database setup.
* **Key Documentation**:
  * `README.md`: Overview of the API service, Docker setup, and local development.
  * `API_ENDPOINTS.md`: Detailed list of API endpoints.

### `/` (Root directory)
The root directory contains comprehensive documentation, architecture designs, and database schemas.
* **Key Technical Specifications**:
  * `BACKEND_ARCHITECTURE.md`: The complete technical specification for the backend (over 2,100 lines).
  * `DATABASE_SCHEMA_QUICK_REFERENCE.md`: Quick reference for the 12 PostgreSQL tables.
  * `SQL_MIGRATION_SCRIPTS.sql`: Ready-to-run SQL scripts to set up the database.
  * `ETL_DATA_FLOW_SPECIFICATION.md`: Critical document detailing ETL data flow and field computations.
  * `PYTHON_CONSIDERATIONS.md`: Analysis of using Python vs Go.
* **Other Important Documents**:
  * `build guide.txt`: Business requirements for the dashboard.
  * `style guide.txt`: UI/UX specifications.
  * `BACKEND_IMPLEMENTATION_SUMMARY.md`: Executive summary and implementation guide.

## Tech Stack
* **Frontend**: React 18, Vite, CSS3, Lucide React, Recharts.
* **Backend**: Go (Golang), PostgreSQL 14+, Redis 7+.
* **Infrastructure**: Docker, Docker Compose.
