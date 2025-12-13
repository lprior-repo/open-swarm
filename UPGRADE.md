# Temporal Stack Upgrade Guide

## 🚀 What's New

### Latest Versions
- **Temporal Server**: v1.29.1 (latest stable)
- **Temporal UI**: v2.43.3 (modern web UI with enhanced features)
- **PostgreSQL**: 16-alpine (latest)
- **Monitoring Stack**: Prometheus v2.54.1 + Grafana v11.4.0

### New Features

#### 1. **Separate Modern UI Container**
- Dedicated `temporalio/ui:2.43.3` container
- Available at: http://localhost:8080
- Features:
  - 🔍 Saved Views for frequently used queries
  - 🚨 Auto-flagging of failed workflows
  - 📊 Better search and filtering
  - 🎨 Modern, responsive interface
  - 📝 Workflow progress indicators

#### 2. **Full Observability Stack**
- **Prometheus** (http://localhost:9090)
  - Metrics collection from Temporal server
  - Custom metrics from workers
  - System metrics via node-exporter
  
- **Grafana** (http://localhost:3000)
  - Pre-configured dashboards
  - Temporal server metrics
  - SDK worker metrics
  - Custom dashboards support

#### 3. **Admin Tools Container**
- On-demand admin container
- Full temporal CLI access
- Useful for debugging and administration

## 📋 Upgrade Steps

### 1. Backup Current Data (Optional)
```bash
docker compose exec postgresql pg_dump -U temporal temporal > temporal-backup.sql
```

### 2. Stop Old Services
```bash
docker compose down
```

### 3. Pull Latest Configuration
Your updated `docker-compose.yml` is ready!

### 4. Download Grafana Dashboards
```bash
./scripts/setup-dashboards.sh
```

### 5. Start Upgraded Stack
```bash
# Core services only
docker compose up -d

# With monitoring
docker compose -f docker-compose.yml -f docker-compose.monitoring.yml up -d
```

### 6. Verify Services

```bash
# Check all services are healthy
docker compose ps

# Test Temporal Server
curl http://localhost:7233

# Test Temporal UI
curl http://localhost:8080

# Test Prometheus (if monitoring enabled)
curl http://localhost:9090/-/healthy

# Test Grafana (if monitoring enabled)
curl http://localhost:3000/api/health
```

## 🌐 Service URLs

| Service | URL | Default Credentials |
|---------|-----|---------------------|
| Temporal UI | http://localhost:8080 | None |
| Temporal Server (gRPC) | localhost:7233 | None |
| Grafana | http://localhost:3000 | admin/admin |
| Prometheus | http://localhost:9090 | None |
| PostgreSQL | localhost:5433 | temporal/temporal |

## 🔧 Configuration Changes

### docker-compose.yml
- ✅ Pinned versions (no more `:latest`)
- ✅ Separate UI container
- ✅ PostgreSQL 16 with persistent volumes
- ✅ Proper networking
- ✅ Container names for easier management
- ✅ Admin tools profile

### New Files
- `docker-compose.monitoring.yml` - Optional monitoring stack
- `monitoring/prometheus.yml` - Prometheus config
- `monitoring/grafana/` - Grafana provisioning and dashboards
- `scripts/setup-dashboards.sh` - Dashboard installer

## 🎯 Quick Commands

```bash
# Start everything
make docker-up-monitoring

# Start core only
make docker-up

# View logs
docker compose logs -f temporal
docker compose logs -f temporal-ui

# Access admin tools
docker compose --profile admin run temporal-admin-tools

# Inside admin tools container:
temporal operator namespace list
temporal workflow list

# Reload Prometheus config (when monitoring running)
curl -X POST http://localhost:9090/-/reload

# Backup database
docker compose exec postgresql pg_dump -U temporal temporal > backup.sql

# Restore database
docker compose exec -T postgresql psql -U temporal temporal < backup.sql
```

## 📊 Grafana Dashboard Setup

1. Visit http://localhost:3000
2. Login with `admin/admin`
3. Navigate to **Dashboards → Browse**
4. Find **Temporal** folder
5. Available dashboards:
   - Server General Metrics
   - Server Advanced Metrics
   - Frontend Service Metrics
   - History Service Metrics
   - Matching Service Metrics
   - Worker Metrics (Go, Java, Python, TypeScript)

## 🐛 Troubleshooting

### Temporal UI not accessible
```bash
docker compose logs temporal-ui
# Check if temporal server is up first
docker compose ps temporal
```

### Grafana dashboards not showing
```bash
# Re-run dashboard setup
./scripts/setup-dashboards.sh

# Restart Grafana
docker compose restart grafana
```

### Port conflicts
If ports are already in use, edit `docker-compose.yml`:
```yaml
ports:
  - "8081:8080"  # Changed from 8080:8080
```

### Database migration issues
```bash
# Check temporal logs
docker compose logs temporal

# Reset if needed (WARNING: loses all data)
docker compose down -v
docker compose up -d
```

## 🔄 Rollback

If you need to rollback:

```bash
# Restore old docker-compose.yml from git
git checkout HEAD~1 docker-compose.yml

# Restart
docker compose down
docker compose up -d
```

## 📚 Learn More

- [Temporal UI Docs](https://docs.temporal.io/web-ui)
- [Temporal Metrics](https://docs.temporal.io/references/cluster-metrics)
- [Grafana Dashboards Repo](https://github.com/temporalio/dashboards)
- [Temporal Best Practices](https://docs.temporal.io/best-practices)

## ✨ What's Next?

1. **Enable SDK Metrics** - Update your workers to export Prometheus metrics
2. **Custom Dashboards** - Create dashboards for your specific workflows
3. **Alerting** - Set up Grafana alerts for critical metrics
4. **High Availability** - Scale Temporal services for production
