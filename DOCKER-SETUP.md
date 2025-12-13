# Docker Setup - VERIFIED ✅

## Current Status

All docker-compose services are **WORKING** and **TESTED**:

- ✅ PostgreSQL 16 (port 5433)
- ✅ Temporal Server v1.29.1 (port 7233)  
- ✅ Temporal UI v2.43.3 (port 8081)
- ✅ End-to-end workflow tests passing

## Quick Start

```bash
# Start all services
docker compose up -d

# Wait ~20 seconds for healthy status
docker compose ps

# Verify
curl http://localhost:8081  # Temporal UI
docker exec open-swarm-temporal temporal operator cluster health --address temporal:7233
```

## Service URLs

| Service | URL | Status |
|---------|-----|--------|
| Temporal UI | http://localhost:8081 | ✅ Working |
| Temporal Server | localhost:7233 | ✅ Working |
| PostgreSQL | localhost:5433 | ✅ Working |

**Note:** Port 8080 was changed to 8081 because beam.smp (Erlang) was using 8080.

## Test Scripts

### Bash Test (Recommended)
```bash
./test-workflow.sh
```

Runs:
1. Container health check
2. Binary build
3. Worker startup
4. Demo execution  
5. Cleanup

### Nushell Test
```bash
nu watch-workflow.nu --timeout 60
```

Full automated demo with configurable timeout.

## What's Different from Before

### Old Setup (❌ Had Issues)
- Used `temporalio/auto-setup:latest` (unpinned)
- UI was part of temporal container
- UI on port 8233 (didn't work properly)
- PostgreSQL 13

### New Setup (✅ All Working)
- Pinned versions: `1.29.1`, `2.43.3`, `16-alpine`
- Separate UI container (`temporalio/ui:2.43.3`)
- UI on port 8081 (fully functional modern UI)
- PostgreSQL 16 with persistent volumes
- Proper health checks and networking

## Monitoring Stack (Optional)

```bash
# Install dashboards first
./scripts/setup-dashboards.sh

# Start with monitoring
docker compose -f docker-compose.yml -f docker-compose.monitoring.yml up -d
```

Adds:
- Grafana on http://localhost:3000 (admin/admin)
- Prometheus on http://localhost:9090
- Official Temporal dashboards pre-loaded

## Common Commands

```bash
# Start
make docker-up                    # Core services
make docker-up-monitoring         # With Grafana + Prometheus

# Stop
make docker-down                  # Core only
make docker-down-all              # Everything including monitoring

# Logs
docker compose logs -f temporal
docker compose logs -f temporal-ui
docker compose logs -f postgresql

# Health
docker compose ps
docker exec open-swarm-temporal temporal operator cluster health --address temporal:7233

# Restart single service
docker compose restart temporal
docker compose restart temporal-ui
```

## Troubleshooting

### UI Not Loading
```bash
# Check container
docker compose ps temporal-ui

# Check logs
docker compose logs temporal-ui

# Restart
docker compose restart temporal-ui
```

### Temporal Server Not Starting
```bash
# Check logs
docker compose logs temporal

# Most common: DB connection issue
docker compose ps postgresql

# Restart everything
docker compose down
docker compose up -d
```

### Port Already in Use
```bash
# Check what's using the port
ss -tlnp | grep :8081

# Change port in docker-compose.yml:
ports:
  - "8082:8080"  # Changed from 8081
```

### Database Issues
```bash
# Reset everything (WARNING: Deletes all data)
docker compose down -v
docker compose up -d

# Backup before reset
docker compose exec postgresql pg_dump -U temporal temporal > backup.sql

# Restore
docker compose exec -T postgresql psql -U temporal temporal < backup.sql
```

## File Structure

```
open-swarm/
├── docker-compose.yml              # Core services (WORKING ✅)
├── docker-compose.monitoring.yml   # Optional monitoring
├── monitoring/
│   ├── prometheus.yml
│   └── grafana/
│       ├── provisioning/
│       │   ├── datasources/prometheus.yml
│       │   └── dashboards/default.yml
│       └── dashboards/
│           ├── server-general.json
│           └── sdk/*.json
├── test-workflow.sh                # Bash test (WORKING ✅)
├── watch-workflow.nu               # Nu test (WORKING ✅)
└── scripts/
    ├── setup-dashboards.sh         # Downloads Grafana dashboards
    └── install.sh                  # Full installation (coming soon)
```

## Verification Checklist

Run these to verify everything works:

- [ ] `docker compose up -d` - All 3 containers start
- [ ] `docker compose ps` - All show "healthy"
- [ ] `curl http://localhost:8081` - Returns HTML
- [ ] `docker exec open-swarm-temporal temporal operator cluster health --address temporal:7233` - Returns "SERVING"
- [ ] `./test-workflow.sh` - Completes successfully
- [ ] Can access UI in browser at http://localhost:8081
- [ ] Can see workflows in UI after running demo

## Next Steps

1. ✅ Docker setup is SOLID
2. ⏭️ Create bundled installer (`scripts/install.sh`)
3. ⏭️ Package for distribution
4. ⏭️ Add monitoring setup to installer

## Last Tested

- Date: 2025-12-12
- All services: ✅ WORKING
- Test scripts: ✅ PASSING
- UI accessible: ✅ YES
- Workflows executing: ✅ YES
