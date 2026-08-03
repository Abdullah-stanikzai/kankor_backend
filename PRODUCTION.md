# Kankor Exam Platform — Production Deployment Guide

## Architecture Overview

```
┌──────────────────────────────────────────────────┐
│                    VPS (Host)                     │
│                                                   │
│  ┌─────────────────────────────────────────────┐ │
│  │         Docker Compose Stack                 │ │
│  │                                              │ │
│  │  ┌──────────┐  ┌────────────┐  ┌─────────┐ │ │
│  │  │ Backend  │  │    PDF     │  │  Postgre│ │ │
│  │  │   :8081  │──│ Extractor  │  │   SQL   │ │ │
│  │  │  (Go)    │  │   :5001    │  │  :5433  │ │ │
│  │  └──────────┘  └────────────┘  └─────────┘ │ │
│  │       │               │              │       │ │
│  │       └───────────────┴──────────────┘       │ │
│  │              Docker Network                  │ │
│  └─────────────────────────────────────────────┘ │
│                                                   │
│  Existing project (other ports)                   │
└──────────────────────────────────────────────────┘
```

## Port Mapping (avoiding conflicts with existing VPS services)

| Service         | Host Port | Container Port |
|-----------------|-----------|----------------|
| Backend API     | `8081`    | `8080`         |
| PostgreSQL      | `5433`    | `5432`         |
| PDF Extractor   | `5001`    | `5000`         |

---

## 1. Initial Setup (First Time)

### 1.1 Prerequisites

```bash
# Install Docker (if not installed)
curl -fsSL https://get.docker.com | sh
sudo usermod -aG docker $USER
# Log out and back in for group change to take effect

# Install Docker Compose plugin
sudo apt-get update && sudo apt-get install docker-compose-plugin

# Verify installation
docker --version
docker compose version
```

### 1.2 Clone & Configure

```bash
# Clone the repository
cd /opt
git clone <your-repo-url> kankor-backend
cd kankor-backend

# Create environment file from template
cp .env.production .env

# EDIT .env with your actual values:
nano .env
#   - DB_PASSWORD: Generate a strong password
#   - JWT_SECRET:   Generate with: openssl rand -hex 32
#   - GEMINI_API_KEY: Your Google Gemini API key
```

### 1.3 Generate Secrets

```bash
# Generate a secure JWT secret
openssl rand -hex 32

# Generate a secure DB password
openssl rand -base64 24
```

### 1.4 Build & Start

```bash
# Build all images
docker compose build

# Start all services (detached)
docker compose up -d

# Check if everything is running
docker compose ps

# View logs
docker compose logs -f
```

### 1.5 Verify Deployment

```bash
# Health check the backend
curl http://localhost:8081/health
# Expected: {"status":"ok","message":"Kankor Exam Platform API is running"}

# Health check PDF extractor
curl http://localhost:5001/health
# Expected: {"status":"healthy","service":"pdf-extractor",...}

# Check database
docker compose exec postgres psql -U kankor_user -d kankor_db -c "\dt"
```

---

## 2. Daily Operations

### 2.1 Start / Stop

```bash
# Start all services
docker compose up -d

# Stop all services (preserves data)
docker compose down

# Stop and remove volumes (⚠ DESTROYS ALL DATA)
docker compose down -v

# Restart a specific service
docker compose restart backend
docker compose restart pdf-extractor
```

### 2.2 View Logs

```bash
# All services
docker compose logs -f

# Specific service
docker compose logs -f backend
docker compose logs -f postgres
docker compose logs -f pdf-extractor

# Last 100 lines
docker compose logs --tail=100 backend

# Timestamped logs
docker compose logs -f --timestamps
```

### 2.3 Service Status

```bash
# Check all containers
docker compose ps

# Resource usage
docker stats

# Check disk usage
docker system df
```

---

## 3. Updates & Deployment

### 3.1 Update Application Code

```bash
cd /opt/kankor-backend

# Pull latest code
git pull origin main

# Rebuild and restart (zero-downtime for DB)
docker compose up -d --build backend
docker compose up -d --build pdf-extractor

# Or rebuild everything
docker compose up -d --build

# Clean up old images
docker image prune -f
```

### 3.2 Full Redeploy

```bash
cd /opt/kankor-backend
git pull origin main
docker compose down
docker compose build --no-cache
docker compose up -d
docker compose logs -f
```

---

## 4. Database Management

### 4.1 Connect to Database

```bash
# Via docker exec
docker compose exec postgres psql -U kankor_user -d kankor_db

# From host (using mapped port 5433)
psql -h localhost -p 5433 -U kankor_user -d kankor_db
```

### 4.2 Backup & Restore

```bash
# Create backup
docker compose exec -T postgres pg_dump -U kankor_user -d kankor_db > backup_$(date +%Y%m%d_%H%M%S).sql

# Restore from backup
docker compose exec -T postgres psql -U kankor_user -d kankor_db < backup_20250101_120000.sql

# Automated daily backup (add to crontab)
# 0 2 * * * cd /opt/kankor-backend && docker compose exec -T postgres pg_dump -U kankor_user -d kankor_db > /backups/kankor_$(date +\%Y\%m\%d).sql
```

### 4.3 Database Migrations

```bash
# Run a migration script
docker compose exec -T postgres psql -U kankor_user -d kankor_db < migrations/xxx.sql

# Reset database (⚠ DESTROYS ALL DATA)
docker compose down -v
docker compose up -d
```

---

## 5. Troubleshooting

### 5.1 Common Issues

**Backend can't connect to database:**
```bash
# Check if postgres is healthy
docker compose ps postgres

# Check backend logs
docker compose logs backend | grep -i "database\|connection"

# Verify env vars in container
docker compose exec backend env | grep DB_
```

**PDF extraction not working:**
```bash
# Check PDF extractor health
curl http://localhost:5001/health

# Check PDF extractor logs
docker compose logs pdf-extractor

# Verify GEMINI_API_KEY is set
docker compose exec backend env | grep GEMINI
```

**Port conflicts:**
```bash
# Check what's using a port
sudo lsof -i :8081
sudo lsof -i :5433
sudo lsof -i :5001

# Change ports in .env if needed:
#   KANKOR_API_PORT=8082
#   KANKOR_DB_PORT=5434
#   KANKOR_PDF_PORT=5002
```

### 5.2 Quick Diagnostics

```bash
# Full health check
echo "=== Backend ===" && curl -s http://localhost:8081/health | python3 -m json.tool
echo "=== PDF Extractor ===" && curl -s http://localhost:5001/health | python3 -m json.tool
echo "=== PG Status ===" && docker compose exec postgres pg_isready -U kankor_user -d kankor_db

# Check container resource usage
docker stats --no-stream

# View recent errors only
docker compose logs --tail=50 | grep -i "error\|fatal\|panic"
```

### 5.3 Emergency Commands

```bash
# Force recreate a failing service
docker compose up -d --force-recreate backend

# Enter a running container for debugging
docker compose exec backend sh
docker compose exec pdf-extractor bash

# View PostgreSQL logs
docker compose logs postgres

# Full system cleanup (⚠ removes all unused Docker data)
docker system prune -a --volumes
```

---

## 6. Performance & Monitoring

### 6.1 Set up a simple health check cron

```bash
# Add to crontab: crontab -e
# Every 5 minutes: check if backend is up
*/5 * * * * curl -sf http://localhost:8081/health || echo "Kankor backend DOWN at $(date)" >> /var/log/kankor-alerts.log
```

### 6.2 Resource Limits (add to docker-compose.yml under each service)

```yaml
deploy:
  resources:
    limits:
      cpus: '1'
      memory: '512M'
    reservations:
      cpus: '0.25'
      memory: '128M'
```

---

## 7. Security Checklist

- [ ] Changed `DB_PASSWORD` from default
- [ ] Set a strong `JWT_SECRET` (64+ characters)
- [ ] Added real `GEMINI_API_KEY`
- [ ] Firewall only exposes backend port (8081) publicly; hide 5433 and 5001
- [ ] Set up SSL/TLS with Nginx reverse proxy (e.g., using certbot)
- [ ] Regular database backups configured
- [ ] `.env` file has `600` permissions: `chmod 600 .env`

### Nginx Reverse Proxy Example

```nginx
server {
    listen 443 ssl;
    server_name api.yourdomain.com;

    ssl_certificate     /etc/letsencrypt/live/api.yourdomain.com/fullchain.pem;
    ssl_certificate_key /etc/letsencrypt/live/api.yourdomain.com/privkey.pem;

    location / {
        proxy_pass http://127.0.0.1:8081;
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto $scheme;
    }

    client_max_body_size 20M;
}
```

---

## 8. Quick Reference Card

```bash
# ─── Start / Stop ───────────────────────
docker compose up -d              # Start all
docker compose down               # Stop all (keep data)
docker compose down -v            # Stop + DELETE all data

# ─── View ───────────────────────────────
docker compose ps                 # Service status
docker compose logs -f            # Live logs
docker compose logs -f backend    # Backend logs only

# ─── Rebuild ────────────────────────────
docker compose up -d --build      # Rebuild + start
docker compose build --no-cache   # Clean rebuild
docker compose up -d --build backend  # Rebuild only backend

# ─── Database ───────────────────────────
docker compose exec postgres psql -U kankor_user -d kankor_db   # SQL shell
docker compose exec -T postgres pg_dump -U kankor_user -d kankor_db > backup.sql  # Backup

# ─── Health ─────────────────────────────
curl http://localhost:8081/health        # Backend health
curl http://localhost:5001/health        # PDF extractor health
docker compose exec postgres pg_isready -U kankor_user -d kankor_db  # DB health
```

---

## 9. Environment Variable Reference

| Variable           | Required | Default          | Description                         |
|--------------------|----------|------------------|-------------------------------------|
| `DB_NAME`          | No       | `kankor_db`      | PostgreSQL database name            |
| `DB_USER`          | No       | `kankor_user`    | PostgreSQL user                     |
| `DB_PASSWORD`      | **Yes**  | —                | PostgreSQL password                 |
| `KANKOR_DB_PORT`   | No       | `5433`           | Host port for PostgreSQL            |
| `KANKOR_API_PORT`  | No       | `8081`           | Host port for backend API           |
| `KANKOR_PDF_PORT`  | No       | `5001`           | Host port for PDF extractor         |
| `JWT_SECRET`       | **Yes**  | —                | JWT signing secret (min 32 chars)   |
| `APP_ENV`          | No       | `production`     | Application environment             |
| `GEMINI_API_KEY`   | **Yes**  | —                | Google Gemini API key               |
