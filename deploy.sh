#!/bin/bash

# ========================================
# Seeds Metrics Dashboard Deployment Script
# ========================================
# This script deploys both backend and frontend to production
# Production URL: https://metrics.seedsandpennies.com
# ========================================

set -e  # Exit on error

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Configuration
PRODUCTION_SERVER="143.198.146.44"
PRODUCTION_USER="root"
BACKEND_PATH="/home/seeds-metrics-backend/backend"
FRONTEND_PATH="/home/seeds-metrics-frontend"
BACKEND_PORT="8080"

# Functions
print_header() {
    echo -e "${BLUE}========================================${NC}"
    echo -e "${BLUE}$1${NC}"
    echo -e "${BLUE}========================================${NC}"
}

print_success() {
    echo -e "${GREEN}✅ $1${NC}"
}

print_error() {
    echo -e "${RED}❌ $1${NC}"
}

print_warning() {
    echo -e "${YELLOW}⚠️  $1${NC}"
}

print_info() {
    echo -e "${BLUE}ℹ️  $1${NC}"
}

# Check if SSH key is configured
check_ssh() {
    print_info "Checking SSH connection to production server..."
    if ssh -o BatchMode=yes -o ConnectTimeout=5 ${PRODUCTION_USER}@${PRODUCTION_SERVER} echo "SSH OK" 2>&1 | grep -q "SSH OK"; then
        print_success "SSH connection successful"
        return 0
    else
        print_error "Cannot connect to production server via SSH"
        print_info "Please ensure:"
        print_info "  1. SSH key is added to the server"
        print_info "  2. Server hostname is correct: ${PRODUCTION_SERVER}"
        print_info "  3. User is correct: ${PRODUCTION_USER}"
        return 1
    fi
}

# Deploy Backend
deploy_backend() {
    print_header "DEPLOYING BACKEND"

    print_info "Deploying backend to production server..."

    # Deploy backend: pull code, build on server, restart systemd service
    ssh ${PRODUCTION_USER}@${PRODUCTION_SERVER} << 'ENDSSH'
        set -e
        cd /home/seeds-metrics-backend/backend

        echo "📥 Pulling latest code from GitHub..."
        git pull origin main

        echo "🔨 Building backend binary on server..."
        /usr/local/go/bin/go build -o seeds-metrics-api cmd/api/main.go

        echo "✅ Build successful"
        ls -lh seeds-metrics-api

        echo "🔄 Restarting backend service..."
        systemctl restart seeds-metrics-api.service

        echo "⏳ Waiting for service to start..."
        sleep 3

        echo "📊 Checking service status..."
        systemctl status seeds-metrics-api.service --no-pager -l

        echo "🧪 Testing API endpoint..."
        curl -s http://localhost:8080/api/v1/health || echo "Health check endpoint not available"
ENDSSH

    print_success "Backend deployed and restarted successfully"
}

# Deploy Frontend
deploy_frontend() {
    print_header "DEPLOYING FRONTEND"

    print_info "Building frontend locally..."
    cd metrics-dashboard

    # Install dependencies
    print_info "Installing npm dependencies..."
    npm install

    # Build for production
    print_info "Building React app for production..."
    npm run build

    if [ ! -d "dist" ]; then
        print_error "Frontend build failed - dist directory not found"
        exit 1
    fi

    print_success "Frontend built successfully"

    # Deploy to production server
    print_info "Deploying frontend to production server..."
    scp -r dist/* ${PRODUCTION_USER}@${PRODUCTION_SERVER}:${FRONTEND_PATH}/

    print_success "Frontend deployed successfully"
    cd ..
}

# Run database migrations
run_migrations() {
    print_header "RUNNING DATABASE MIGRATIONS"

    print_info "Applying database migrations on production..."
    ssh ${PRODUCTION_USER}@${PRODUCTION_SERVER} << 'ENDSSH'
        cd /home/seeds-metrics-backend/backend/migrations

        # Check if apply_all_fixes.sh exists
        if [ -f "apply_all_fixes.sh" ]; then
            chmod +x apply_all_fixes.sh
            ./apply_all_fixes.sh
        else
            echo "No migration script found, skipping..."
        fi
ENDSSH

    print_success "Database migrations completed"
}

# Verify deployment
verify_deployment() {
    print_header "VERIFYING DEPLOYMENT"

    # Check backend
    print_info "Checking backend API..."
    sleep 3  # Wait for backend to fully start

    BACKEND_STATUS=$(curl -s -o /dev/null -w "%{http_code}" https://metrics.seedsandpennies.com/api/v1/metrics/portfolio)

    if [ "$BACKEND_STATUS" = "200" ]; then
        print_success "Backend API is responding (HTTP 200)"
    else
        print_warning "Backend API returned HTTP ${BACKEND_STATUS}"
    fi

    # Check frontend
    print_info "Checking frontend..."
    FRONTEND_STATUS=$(curl -s -o /dev/null -w "%{http_code}" https://metrics.seedsandpennies.com/)

    if [ "$FRONTEND_STATUS" = "200" ]; then
        print_success "Frontend is accessible (HTTP 200)"
    else
        print_warning "Frontend returned HTTP ${FRONTEND_STATUS}"
    fi

    print_success "Deployment verification complete"
}

# Main deployment flow
main() {
    print_header "SEEDS METRICS DEPLOYMENT"
    print_info "Target: https://metrics.seedsandpennies.com"
    print_info "Starting deployment at $(date)"
    echo ""

    # Check SSH connection
    if ! check_ssh; then
        print_error "Deployment aborted - SSH connection failed"
        exit 1
    fi

    # Ask for confirmation
    echo ""
    print_warning "This will deploy to PRODUCTION server: ${PRODUCTION_SERVER}"
    read -p "Continue? (yes/no): " -r
    echo ""

    if [[ ! $REPLY =~ ^[Yy][Ee][Ss]$ ]]; then
        print_info "Deployment cancelled"
        exit 0
    fi

    # Deploy backend
    deploy_backend
    echo ""

    # Deploy frontend
    deploy_frontend
    echo ""

    # Run migrations (optional)
    read -p "Run database migrations? (yes/no): " -r
    echo ""
    if [[ $REPLY =~ ^[Yy][Ee][Ss]$ ]]; then
        run_migrations
        echo ""
    fi

    # Verify deployment
    verify_deployment
    echo ""

    print_header "DEPLOYMENT COMPLETE"
    print_success "Backend deployed to: https://metrics.seedsandpennies.com/api/v1"
    print_success "Frontend deployed to: https://metrics.seedsandpennies.com"
    print_success "Swagger docs: https://metrics.seedsandpennies.com/swagger/index.html"
    print_info "Deployment completed at $(date)"
}

# Run main function
main

