#!/bin/bash

# ========================================
# Deploy User Type Filter Feature
# ========================================
# This script deploys the user_type filter feature to production
# ========================================

set -e  # Exit on error

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

echo -e "${BLUE}========================================${NC}"
echo -e "${BLUE}Deploying User Type Filter Feature${NC}"
echo -e "${BLUE}========================================${NC}"

# Configuration
PRODUCTION_SERVER="root@143.198.146.44"
BACKEND_PATH="/home/seeds-metrics-backend/backend"

echo -e "\n${YELLOW}Step 1: Syncing backend code files...${NC}"
rsync -avz --progress \
  backend/internal/models/officer.go \
  backend/internal/repository/officer_repository.go \
  backend/internal/repository/django_repository.go \
  backend/internal/repository/dashboard_repository.go \
  backend/scripts/sync_from_django.go \
  ${PRODUCTION_SERVER}:${BACKEND_PATH}/temp_sync/

echo -e "\n${YELLOW}Step 2: Moving files to correct locations on server...${NC}"
ssh ${PRODUCTION_SERVER} << 'EOF'
  cd /home/seeds-metrics-backend/backend
  
  # Create temp directory if it doesn't exist
  mkdir -p temp_sync
  
  # Move files to correct locations
  mv temp_sync/officer.go internal/models/
  mv temp_sync/officer_repository.go internal/repository/
  mv temp_sync/django_repository.go internal/repository/
  mv temp_sync/dashboard_repository.go internal/repository/
  mv temp_sync/sync_from_django.go scripts/
  
  # Clean up
  rmdir temp_sync
  
  echo "Files moved successfully"
EOF

echo -e "\n${YELLOW}Step 3: Building backend on server...${NC}"
ssh ${PRODUCTION_SERVER} << 'EOF'
  cd /home/seeds-metrics-backend/backend
  
  # Build the binary
  go build -o seeds-metrics-api ./cmd/api
  
  if [ $? -eq 0 ]; then
    echo "Build successful"
  else
    echo "Build failed"
    exit 1
  fi
EOF

echo -e "\n${YELLOW}Step 4: Deploying backend binary...${NC}"
ssh ${PRODUCTION_SERVER} << 'EOF'
  cd /home/seeds-metrics-backend/backend
  
  # Backup old binary
  if [ -f api ]; then
    mv api api.old.$(date +%Y%m%d_%H%M%S)
  fi
  
  # Deploy new binary
  mv seeds-metrics-api api
  chmod +x api
  
  echo "Binary deployed"
EOF

echo -e "\n${YELLOW}Step 5: Running database migration...${NC}"
ssh ${PRODUCTION_SERVER} << 'EOF'
  cd /home/seeds-metrics-backend/backend
  source .env
  
  # Run migration
  psql "host=$DB_HOST port=$DB_PORT dbname=$DB_NAME user=$DB_USER password=$DB_PASSWORD sslmode=$DB_SSLMODE" \
    -f migrations/017_add_user_type_to_officers.sql
  
  echo "Migration completed"
EOF

echo -e "\n${YELLOW}Step 6: Syncing user_type data from Django...${NC}"
ssh ${PRODUCTION_SERVER} << 'EOF'
  cd /home/seeds-metrics-backend/backend
  
  # Run sync script
  ./sync_from_django
  
  echo "Data sync completed"
EOF

echo -e "\n${YELLOW}Step 7: Restarting backend service...${NC}"
ssh ${PRODUCTION_SERVER} << 'EOF'
  sudo systemctl restart seeds-metrics-api
  sleep 3
  sudo systemctl status seeds-metrics-api --no-pager -l
EOF

echo -e "\n${YELLOW}Step 8: Testing API endpoints...${NC}"
echo -e "${BLUE}Testing /api/v1/filters/user-types...${NC}"
curl -s "https://metrics.seedsandpennies.com/api/v1/filters/user-types" | jq '.'

echo -e "\n${BLUE}Testing /api/v1/officers with user_type filter...${NC}"
curl -s "https://metrics.seedsandpennies.com/api/v1/officers?user_type=MERCHANT&page=1&limit=5" | \
  jq '.data.officers[] | {officer_id, officer_name, user_type}'

echo -e "\n${GREEN}========================================${NC}"
echo -e "${GREEN}Backend Deployment Complete!${NC}"
echo -e "${GREEN}========================================${NC}"

echo -e "\n${YELLOW}Step 9: Building frontend...${NC}"
cd metrics-dashboard
npm run build

echo -e "\n${YELLOW}Step 10: Deploying frontend...${NC}"
rsync -avz --delete --progress dist/ ${PRODUCTION_SERVER}:/home/seeds-metrics-frontend/

echo -e "\n${GREEN}========================================${NC}"
echo -e "${GREEN}Frontend Deployment Complete!${NC}"
echo -e "${GREEN}========================================${NC}"

echo -e "\n${BLUE}Testing complete deployment...${NC}"
echo -e "Opening https://metrics.seedsandpennies.com in browser..."
echo -e "Please navigate to the Officer Performance tab and test the User Type filter."

echo -e "\n${GREEN}✅ Deployment Complete!${NC}"
echo -e "\n${YELLOW}Next Steps:${NC}"
echo -e "1. Open https://metrics.seedsandpennies.com"
echo -e "2. Navigate to 'Agent Performance' tab"
echo -e "3. Click 'Filters' button"
echo -e "4. Test the 'User Type' multi-select filter"
echo -e "5. Verify filtering works correctly"

