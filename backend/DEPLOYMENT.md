# 🚀 Deployment Guide

Complete guide for deploying the Analytics Backend to production.

---

## 📋 Pre-Deployment Checklist

### **Security**

- [ ] Change all default passwords in `.env`
- [ ] Enable SSL for PostgreSQL (`DB_SSLMODE=require`)
- [ ] Set strong `DB_PASSWORD` and `REDIS_PASSWORD`
- [ ] Configure proper CORS origins (remove `*`)
- [ ] Enable HTTPS for API endpoints
- [ ] Setup firewall rules
- [ ] Implement authentication (JWT)
- [ ] Enable rate limiting

### **Configuration**

- [ ] Set `GIN_MODE=release`
- [ ] Configure proper `LOG_LEVEL` (info or warn)
- [ ] Set appropriate connection pool sizes
- [ ] Configure cache TTL values
- [ ] Set proper CORS allowed origins
- [ ] Configure backup schedules

### **Infrastructure**

- [ ] Provision production servers
- [ ] Setup load balancer
- [ ] Configure DNS
- [ ] Setup SSL certificates
- [ ] Configure monitoring
- [ ] Setup log aggregation
- [ ] Configure alerting

---

## 🐳 Docker Deployment (Recommended)

### **Option 1: Docker Compose (Single Server)**

#### **1. Prepare Server**

```bash
# Install Docker and Docker Compose
curl -fsSL https://get.docker.com -o get-docker.sh
sh get-docker.sh

# Install Docker Compose
sudo curl -L "https://github.com/docker/compose/releases/latest/download/docker-compose-$(uname -s)-$(uname -m)" -o /usr/local/bin/docker-compose
sudo chmod +x /usr/local/bin/docker-compose
```

#### **2. Clone Repository**

```bash
git clone <repository-url>
cd seeds-metrics/backend
```

#### **3. Configure Environment**

```bash
cp .env.example .env

# Edit .env with production values
nano .env
```

**Production .env:**
```bash
# Server
SERVER_PORT=8080
SERVER_HOST=0.0.0.0
GIN_MODE=release

# Database
DB_HOST=postgres
DB_PORT=5432
DB_USER=analytics_user
DB_PASSWORD=<STRONG_PASSWORD_HERE>
DB_NAME=analytics_db
DB_SSLMODE=require
DB_MAX_CONNECTIONS=50
DB_MAX_IDLE_CONNECTIONS=10

# Redis
REDIS_HOST=redis
REDIS_PORT=6379
REDIS_PASSWORD=<STRONG_PASSWORD_HERE>
REDIS_DB=0
REDIS_CACHE_TTL=900

# CORS
CORS_ALLOWED_ORIGINS=https://yourdomain.com
CORS_ALLOWED_METHODS=GET,POST,PUT,DELETE,OPTIONS
CORS_ALLOWED_HEADERS=Origin,Content-Type,Accept,Authorization

# Logging
LOG_LEVEL=info
LOG_FORMAT=json
```

#### **4. Update docker-compose.yml for Production**

```yaml
version: '3.8'

services:
  postgres:
    image: postgres:14-alpine
    container_name: analytics-postgres
    environment:
      POSTGRES_USER: ${DB_USER}
      POSTGRES_PASSWORD: ${DB_PASSWORD}
      POSTGRES_DB: ${DB_NAME}
    ports:
      - "127.0.0.1:5432:5432"  # Only localhost
    volumes:
      - postgres_data:/var/lib/postgresql/data
      - ./migrations:/docker-entrypoint-initdb.d
    restart: always
    deploy:
      resources:
        limits:
          cpus: '2'
          memory: 2G

  redis:
    image: redis:7-alpine
    container_name: analytics-redis
    command: redis-server --requirepass ${REDIS_PASSWORD}
    ports:
      - "127.0.0.1:6379:6379"  # Only localhost
    volumes:
      - redis_data:/data
    restart: always
    deploy:
      resources:
        limits:
          cpus: '1'
          memory: 512M

  api:
    build:
      context: .
      dockerfile: Dockerfile
    container_name: analytics-api
    env_file: .env
    ports:
      - "8080:8080"
    depends_on:
      - postgres
      - redis
    restart: always
    deploy:
      resources:
        limits:
          cpus: '2'
          memory: 1G
      replicas: 2  # Run 2 instances

volumes:
  postgres_data:
  redis_data:

networks:
  default:
    driver: bridge
```

#### **5. Deploy**

```bash
# Build and start
docker-compose up -d --build

# Check status
docker-compose ps

# View logs
docker-compose logs -f

# Check health
curl http://localhost:8080/health
```

#### **6. Setup Nginx Reverse Proxy**

```nginx
# /etc/nginx/sites-available/analytics-api

upstream analytics_backend {
    server localhost:8080;
}

server {
    listen 80;
    server_name api.yourdomain.com;

    # Redirect to HTTPS
    return 301 https://$server_name$request_uri;
}

server {
    listen 443 ssl http2;
    server_name api.yourdomain.com;

    # SSL Configuration
    ssl_certificate /etc/letsencrypt/live/api.yourdomain.com/fullchain.pem;
    ssl_certificate_key /etc/letsencrypt/live/api.yourdomain.com/privkey.pem;

    # Security Headers
    add_header X-Frame-Options "SAMEORIGIN" always;
    add_header X-Content-Type-Options "nosniff" always;
    add_header X-XSS-Protection "1; mode=block" always;

    # Proxy to backend
    location / {
        proxy_pass http://analytics_backend;
        proxy_http_version 1.1;
        proxy_set_header Upgrade $http_upgrade;
        proxy_set_header Connection 'upgrade';
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto $scheme;
        proxy_cache_bypass $http_upgrade;
    }

    # Health check endpoint
    location /health {
        proxy_pass http://analytics_backend/health;
        access_log off;
    }
}
```

Enable and restart Nginx:
```bash
sudo ln -s /etc/nginx/sites-available/analytics-api /etc/nginx/sites-enabled/
sudo nginx -t
sudo systemctl restart nginx
```

---

## ☸️ Kubernetes Deployment (Production Scale)

### **1. Create Kubernetes Manifests**

**namespace.yaml:**
```yaml
apiVersion: v1
kind: Namespace
metadata:
  name: analytics
```

**postgres-deployment.yaml:**
```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: postgres
  namespace: analytics
spec:
  replicas: 1
  selector:
    matchLabels:
      app: postgres
  template:
    metadata:
      labels:
        app: postgres
    spec:
      containers:
      - name: postgres
        image: postgres:14-alpine
        env:
        - name: POSTGRES_USER
          valueFrom:
            secretKeyRef:
              name: db-secret
              key: username
        - name: POSTGRES_PASSWORD
          valueFrom:
            secretKeyRef:
              name: db-secret
              key: password
        - name: POSTGRES_DB
          value: analytics_db
        ports:
        - containerPort: 5432
        volumeMounts:
        - name: postgres-storage
          mountPath: /var/lib/postgresql/data
      volumes:
      - name: postgres-storage
        persistentVolumeClaim:
          claimName: postgres-pvc
```

**api-deployment.yaml:**
```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: analytics-api
  namespace: analytics
spec:
  replicas: 3
  selector:
    matchLabels:
      app: analytics-api
  template:
    metadata:
      labels:
        app: analytics-api
    spec:
      containers:
      - name: api
        image: your-registry/analytics-api:latest
        ports:
        - containerPort: 8080
        env:
        - name: DB_HOST
          value: postgres-service
        - name: DB_PASSWORD
          valueFrom:
            secretKeyRef:
              name: db-secret
              key: password
        resources:
          requests:
            memory: "256Mi"
            cpu: "250m"
          limits:
            memory: "512Mi"
            cpu: "500m"
        livenessProbe:
          httpGet:
            path: /health
            port: 8080
          initialDelaySeconds: 30
          periodSeconds: 10
        readinessProbe:
          httpGet:
            path: /health
            port: 8080
          initialDelaySeconds: 5
          periodSeconds: 5
```

### **2. Deploy to Kubernetes**

```bash
# Create namespace
kubectl apply -f namespace.yaml

# Create secrets
kubectl create secret generic db-secret \
  --from-literal=username=analytics_user \
  --from-literal=password=<STRONG_PASSWORD> \
  -n analytics

# Deploy services
kubectl apply -f postgres-deployment.yaml
kubectl apply -f redis-deployment.yaml
kubectl apply -f api-deployment.yaml

# Check status
kubectl get pods -n analytics
kubectl get services -n analytics
```

---

## 📊 Monitoring Setup

### **Prometheus + Grafana**

```yaml
# docker-compose.monitoring.yml
version: '3.8'

services:
  prometheus:
    image: prom/prometheus:latest
    volumes:
      - ./prometheus.yml:/etc/prometheus/prometheus.yml
      - prometheus_data:/prometheus
    ports:
      - "9090:9090"

  grafana:
    image: grafana/grafana:latest
    volumes:
      - grafana_data:/var/lib/grafana
    ports:
      - "3000:3000"
    environment:
      - GF_SECURITY_ADMIN_PASSWORD=<STRONG_PASSWORD>

volumes:
  prometheus_data:
  grafana_data:
```

---

## 🔄 Backup Strategy

### **Database Backups**

```bash
# Create backup script
cat > /opt/scripts/backup-db.sh << 'EOF'
#!/bin/bash
BACKUP_DIR="/backups/postgres"
DATE=$(date +%Y%m%d_%H%M%S)
FILENAME="analytics_db_${DATE}.sql.gz"

docker exec analytics-postgres pg_dump -U analytics_user analytics_db | gzip > "${BACKUP_DIR}/${FILENAME}"

# Keep only last 7 days
find ${BACKUP_DIR} -name "*.sql.gz" -mtime +7 -delete
EOF

chmod +x /opt/scripts/backup-db.sh

# Add to crontab (daily at 2 AM)
0 2 * * * /opt/scripts/backup-db.sh
```

---

## 🚨 Health Monitoring

### **Setup Health Check Monitoring**

```bash
# Create monitoring script
cat > /opt/scripts/health-check.sh << 'EOF'
#!/bin/bash
HEALTH_URL="http://localhost:8080/health"
ALERT_EMAIL="admin@yourdomain.com"

response=$(curl -s -o /dev/null -w "%{http_code}" ${HEALTH_URL})

if [ "$response" != "200" ]; then
    echo "API health check failed! HTTP ${response}" | mail -s "Analytics API Down" ${ALERT_EMAIL}
fi
EOF

chmod +x /opt/scripts/health-check.sh

# Add to crontab (every 5 minutes)
*/5 * * * * /opt/scripts/health-check.sh
```

---

## 📈 Scaling

### **Horizontal Scaling**

```bash
# Scale API instances
docker-compose up -d --scale api=3

# Or in Kubernetes
kubectl scale deployment analytics-api --replicas=5 -n analytics
```

### **Vertical Scaling**

Update resource limits in docker-compose.yml or Kubernetes manifests.

---

## 🔧 Maintenance

### **Update Application**

```bash
# Pull latest code
git pull origin main

# Rebuild and restart
docker-compose down
docker-compose up -d --build

# Check logs
docker-compose logs -f api
```

### **Database Migrations**

```bash
# Run new migrations
docker exec -it analytics-postgres psql -U analytics_user -d analytics_db -f /path/to/new_migration.sql
```

---

## 📞 Support

For production issues:
1. Check logs: `docker-compose logs -f`
2. Check health: `curl http://localhost:8080/health`
3. Check database: `docker exec -it analytics-postgres psql -U analytics_user -d analytics_db`
4. Review monitoring dashboards

---

**Production deployment complete! 🎉**

