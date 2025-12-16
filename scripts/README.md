# Infrastructure Management Scripts

Comprehensive scripts for managing Open Swarm infrastructure, including Docker containers, observability stack, and benchmarking.

## Quick Start

```bash
# Setup everything (cleanup + start all services + verify)
./scripts/infra.sh setup

# Full cycle with benchmarks
./scripts/infra.sh full --benchmark --strategy enhanced --runs 5

# Just verify current services
./scripts/infra.sh verify
```

## Main Script: `infra.sh`

The unified infrastructure management script that handles:
- 🧹 Cleanup of zombie/leftover Docker resources
- 🚀 Standing up fresh infrastructure with latest images
- 🔍 Health verification of all services
- 📊 Running benchmarks
- 🔄 Updating Docker image versions

### Commands

| Command | Description |
|---------|-------------|
| `setup` | Clean and setup all infrastructure (default) |
| `cleanup` | Only cleanup resources |
| `verify` | Only verify health of running services |
| `benchmark` | Run benchmarks |
| `full` | Full cycle: cleanup + setup + verify + benchmark |
| `update-versions` | Update Docker Compose files to latest image versions |
| `help` | Show help message |

### Options

| Option | Description |
|--------|-------------|
| `--remove-volumes` | Remove Docker volumes (deletes persistent data) |
| `--no-observability` | Skip observability stack (OTEL, Jaeger, Prometheus, Grafana) |
| `--benchmark` | Run benchmark after setup |
| `--strategy <type>` | Benchmark strategy: `basic` or `enhanced` (default: basic) |
| `--runs <number>` | Number of benchmark runs (default: 3) |
| `--prompt <text>` | Benchmark prompt (default: "Implement a simple hello world function") |

### Usage Examples

#### Basic Setup

```bash
# Setup core services only (Temporal + PostgreSQL + UI)
./scripts/infra.sh setup --no-observability

# Setup everything including observability
./scripts/infra.sh setup
```

#### Cleanup

```bash
# Cleanup containers and networks (preserve volumes)
./scripts/infra.sh cleanup

# Cleanup everything including volumes (DESTRUCTIVE)
./scripts/infra.sh cleanup --remove-volumes
```

#### Verification

```bash
# Verify all services are healthy
./scripts/infra.sh verify

# Verify only core services
./scripts/infra.sh verify --no-observability
```

#### Benchmarking

```bash
# Run basic benchmark (3 runs)
./scripts/infra.sh benchmark

# Run enhanced benchmark (10 runs with custom prompt)
./scripts/infra.sh benchmark \
  --strategy enhanced \
  --runs 10 \
  --prompt "Implement a thread-safe LRU cache with TTL support"

# Setup + benchmark in one command
./scripts/infra.sh setup --benchmark --strategy basic --runs 5
```

#### Full Cycle

```bash
# Complete refresh: cleanup + setup + verify
./scripts/infra.sh full

# Complete refresh + benchmark
./scripts/infra.sh full \
  --benchmark \
  --strategy enhanced \
  --runs 5 \
  --prompt "Add rate limiting to API endpoints"

# Complete refresh + delete all data
./scripts/infra.sh full --remove-volumes
```

#### Update Versions

```bash
# Update all Docker Compose files to latest versions
./scripts/infra.sh update-versions

# Then review changes
git diff docker-compose*.yml
```

## Docker Image Versions

The script uses the latest stable versions of all services:

| Service | Version | Description |
|---------|---------|-------------|
| PostgreSQL | `17-alpine` | Latest PostgreSQL database |
| Temporal Server | `1.25.2` | Latest Temporal workflow engine |
| Temporal UI | `2.33.1` | Latest Temporal web UI |
| OTEL Collector | `0.115.1` | Latest OpenTelemetry collector |
| Jaeger | `2.3.0` | Latest Jaeger tracing |
| Prometheus | `v3.0.1` | Latest Prometheus metrics |
| Grafana | `11.4.0` | Latest Grafana dashboards |
| Node Exporter | `v1.8.2` | Latest Prometheus node exporter |

### Overriding Versions

You can override versions using environment variables:

```bash
POSTGRES_VERSION=16-alpine \
TEMPORAL_VERSION=1.24.0 \
./scripts/infra.sh setup
```

## Service URLs

After running `setup` or `verify`, you'll see:

| Service | URL | Credentials |
|---------|-----|-------------|
| Temporal UI | http://localhost:8081 | None |
| Jaeger (Traces) | http://localhost:16686 | None |
| Grafana | http://localhost:3001 | admin/admin |
| Prometheus | http://localhost:9090 | None |
| OTEL HTTP | http://localhost:4318 | None |
| OTEL gRPC | http://localhost:4317 | None |
| PostgreSQL | localhost:5433 | temporal/temporal |
| Temporal Server | localhost:7233 | None |

## Health Checks

The script performs comprehensive health checks:

1. **Container Health**: Verifies all Docker containers are running
2. **HTTP Endpoints**: Checks HTTP services are responding
3. **Port Availability**: Ensures services are listening on correct ports

If any service fails health checks, the script will report which services are unhealthy.

## Cleanup Details

The cleanup process removes:

- ✅ All Open Swarm Docker containers
- ✅ Open Swarm Docker networks
- ✅ Temporary files (`otel-data/`, `worktrees/*`)
- ✅ Unused Docker resources (via `docker system prune`)
- ⚠️ Docker volumes (only with `--remove-volumes` flag)

### What Gets Preserved

By default, these are **preserved** (unless you use `--remove-volumes`):

- PostgreSQL data (`postgresql-data` volume)
- Prometheus data (`prometheus-data` volume)
- Grafana data (`grafana-data` volume)

This allows you to refresh infrastructure without losing:
- Temporal workflow history
- Grafana dashboards
- Prometheus metrics

## Architecture

```
infra.sh
├── cleanup
│   ├── Stop & remove containers
│   ├── Remove networks
│   ├── Remove volumes (optional)
│   ├── Prune Docker resources
│   └── Clean temp files
├── setup
│   ├── Setup core (Temporal + PostgreSQL + UI)
│   └── Setup observability (OTEL + Jaeger + Prometheus + Grafana)
├── verify
│   ├── Check containers running
│   ├── Check HTTP endpoints
│   └── Report health status
└── benchmark
    ├── Build binaries
    └── Run benchmark workflow
```

## Integration with Makefile

The script integrates with the existing Makefile:

```bash
# Via Makefile
make docker-up              # Uses docker-compose directly
make docker-up-monitoring   # Uses docker-compose with monitoring
make docker-down            # Stop services

# Via infra.sh (recommended)
./scripts/infra.sh setup    # Cleanup + setup + verify
./scripts/infra.sh full     # Complete cycle
```

**Recommendation**: Use `infra.sh` for full infrastructure management, and Makefile for quick container operations.

## Troubleshooting

### Containers Won't Start

```bash
# Check Docker daemon
docker info

# View logs
docker logs open-swarm-temporal
docker logs open-swarm-postgresql

# Full cleanup and retry
./scripts/infra.sh cleanup --remove-volumes
./scripts/infra.sh setup
```

### Health Checks Failing

```bash
# Check container status
docker ps -a --filter "name=open-swarm"

# Check specific service logs
docker logs open-swarm-temporal-ui

# Restart failed service
docker restart open-swarm-temporal
./scripts/infra.sh verify
```

### Port Conflicts

If ports are already in use:

1. Find what's using the port:
   ```bash
   lsof -i :8081
   netstat -tulpn | grep 8081
   ```

2. Stop the conflicting service or change ports in `docker-compose.yml`

### Benchmark Fails

```bash
# Ensure services are healthy
./scripts/infra.sh verify

# Build binaries manually
make build

# Check worker is running
ps aux | grep temporal-worker

# Run benchmark with verbose output
./bin/benchmark-tcr -strategy basic -runs 1 -prompt "test"
```

## Development Workflow

### Daily Development

```bash
# Morning: Start everything
./scripts/infra.sh setup

# Work...

# Evening: Stop everything (preserve data)
docker compose down
docker compose -f docker-compose.otel.yml down
```

### Testing Changes

```bash
# Test with fresh state
./scripts/infra.sh full --remove-volumes

# Test specific component
./scripts/infra.sh setup --no-observability
```

### Before Committing

```bash
# Verify everything works
./scripts/infra.sh full --benchmark --runs 3

# If passing, commit
git add .
git commit -m "Your changes"
```

## CI/CD Integration

```yaml
# Example GitHub Actions workflow
name: Infrastructure Test
on: [push]
jobs:
  test:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v3
      - name: Setup infrastructure
        run: ./scripts/infra.sh setup
      - name: Run tests
        run: make test
      - name: Cleanup
        run: ./scripts/infra.sh cleanup
```

## Related Documentation

- [Docker Setup Guide](../DOCKER-SETUP.md)
- [Telemetry Guide](../TELEMETRY.md)
- [Benchmark Documentation](../cmd/benchmark-tcr/README.md)
- [Makefile Commands](../Makefile)

## Support

For issues or questions:

1. Check service logs: `docker logs <container-name>`
2. Verify Docker is running: `docker info`
3. Review documentation: `./scripts/infra.sh --help`
4. Open an issue with full output from `./scripts/infra.sh verify`
