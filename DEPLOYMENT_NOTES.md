# Seeds Metrics - Deployment Notes

## Production Deployment

### Frontend Deployment

**IMPORTANT**: The NGINX root is set to `/home/seeds-metrics-frontend/` (NOT `/home/seeds-metrics-frontend/dist/`)

#### Correct Deployment Command:
```bash
# Build the frontend
cd metrics-dashboard
npm run build

# Deploy to production (from repository root)
scp -r metrics-dashboard/dist/* root@143.198.146.44:/home/seeds-metrics-frontend/
```

#### Common Mistake:
❌ **WRONG**: `scp -r metrics-dashboard/dist/* root@143.198.146.44:/home/seeds-metrics-frontend/dist/`

This deploys to the wrong directory. NGINX serves from `/home/seeds-metrics-frontend/`, not `/home/seeds-metrics-frontend/dist/`.

### NGINX Configuration

**Location**: `/etc/nginx/sites-available/seeds-metrics`

**Key Settings**:
- Root directory: `/home/seeds-metrics-frontend/`
- Static assets (JS, CSS) are cached for 1 year with `immutable` flag
- HTML files are not cached (`no-store, no-cache`)
- API requests to `/api/v1/*` are proxied to `localhost:8080`

**Cache Headers**:
```nginx
# Cache static assets (JS, CSS, images) for 1 year
location ~* \.(js|css|png|jpg|jpeg|gif|ico|svg|woff|woff2|ttf|eot)$ {
    expires 1y;
    add_header Cache-Control "public, immutable";
}

# Do not cache HTML files (for immediate updates)
location ~* \.html$ {
    expires -1;
    add_header Cache-Control "no-store, no-cache, must-revalidate, proxy-revalidate, max-age=0";
}
```

### Backend Deployment

**Location**: `/home/seeds-metrics-backend/backend/`

**Service**: `seeds-metrics-api.service` (systemd)

**Deployment Steps**:
1. SSH to server: `ssh root@143.198.146.44`
2. Navigate to backend directory: `cd /home/seeds-metrics-backend/backend/`
3. Pull latest changes: `git pull origin main`
4. Build: `go build -o seeds-metrics-api cmd/api/main.go`
5. Restart service: `systemctl restart seeds-metrics-api.service`
6. Check status: `systemctl status seeds-metrics-api.service`

### Database Migrations

**Connection String**:
```bash
postgresql://metricsuser:PASSWORD@private-generaldb-do-user-9489371-0.k.db.ondigitalocean.com:25060/seedsmetrics?sslmode=require
```

**Run Migration**:
```bash
psql 'postgresql://metricsuser:PASSWORD@private-generaldb-do-user-9489371-0.k.db.ondigitalocean.com:25060/seedsmetrics?sslmode=require' \
  -f /path/to/migration.sql
```

### Troubleshooting

#### Issue: New frontend changes not appearing on production

**Symptoms**:
- Changes work locally but not on production
- Browser shows old version even after hard refresh

**Root Causes**:
1. **Wrong deployment directory**: Files deployed to `/home/seeds-metrics-frontend/dist/` instead of `/home/seeds-metrics-frontend/`
2. **Browser cache**: Static assets cached for 1 year with `immutable` flag
3. **Old index.html**: NGINX serving old index.html that references old asset files

**Solution**:
1. Verify deployment directory:
   ```bash
   ssh root@143.198.146.44 'ls -lh /home/seeds-metrics-frontend/'
   ```

2. Check which assets index.html references:
   ```bash
   ssh root@143.198.146.44 'cat /home/seeds-metrics-frontend/index.html | grep -E "(index-.*\.js|index-.*\.css)"'
   ```

3. Verify production website serves correct index.html:
   ```bash
   curl -s https://metrics.seedsandpennies.com/ | grep -E "(index-.*\.js|index-.*\.css)"
   ```

4. If files are in wrong location, copy them:
   ```bash
   ssh root@143.198.146.44 'cp -r /home/seeds-metrics-frontend/dist/* /home/seeds-metrics-frontend/'
   ```

5. Clear browser cache:
   - Hard refresh: Cmd+Shift+R (Mac) or Ctrl+Shift+R (Windows/Linux)
   - Or clear browser cache completely

### Production URLs

- **Website**: https://metrics.seedsandpennies.com
- **API**: https://metrics.seedsandpennies.com/api/v1
- **Swagger**: https://metrics.seedsandpennies.com/swagger/index.html

### Server Details

- **IP**: 143.198.146.44
- **User**: root
- **SSH**: `ssh root@143.198.146.44`
- **OS**: Ubuntu (DigitalOcean Droplet)

### Git Repository

- **URL**: git@github.com:LibertytechX/seeds-metrics.git
- **Branch**: main

---

**Last Updated**: 2025-11-06
**Updated By**: Augment Agent

