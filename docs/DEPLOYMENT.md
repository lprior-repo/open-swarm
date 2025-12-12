# Open Swarm Deployment Guide

This guide covers deploying Open Swarm in development, staging, and production environments.

## Table of Contents

1. [Prerequisites](#prerequisites)
2. [Development Setup](#development-setup)
3. [Production Deployment](#production-deployment)
4. [Scaling](#scaling)
5. [Monitoring](#monitoring)
6. [Security](#security)

---

## Prerequisites

### System Requirements

**Minimum Hardware:**
- CPU: 2 cores
- RAM: 4GB (development), 8GB+ (production)
- Disk: 20GB available space
- Network: Reliable internet connection

**Supported Operating Systems:**
- Linux (Ubuntu 20.04+, CentOS 8+, Debian 11+)
- macOS (12+)
- Windows (WSL2 recommended)

### Required Software

#### 1. Go 1.25+

```bash
# Verify installation
go version
# Should output: go version go1.25.x linux/amd64 (or darwin/amd64)
```

**Installation:** See https://go.dev/dl/

#### 2. Docker and Docker Compose

```bash
# Install Docker (Ubuntu/Debian)
sudo apt-get update
sudo apt-get install -y docker.io docker-compose

# Verify installation
docker --version
docker-compose --version

# Add your user to docker group (optional, requires logout/login)
sudo usermod -aG docker $USER
```

**Installation:** See https://docs.docker.com/engine/install/

#### 3. Git

```bash
# Install Git
sudo apt-get install -y git

# Verify installation
git --version
```

#### 4. OpenCode (SST)

```bash
# Install OpenCode
curl -fsSL https://opencode.ai/install | bash

# Verify installation
opencode --version
```

#### 5. Agent Mail MCP Server

```bash
# Install Agent Mail
curl -fsSL "https://raw.githubusercontent.com/Dicklesworthstone/mcp_agent_mail/main/scripts/install.sh?$(date +%s)" | bash -s -- --yes

# Verify installation (starts server)
am --version
# Press Ctrl+C to stop
```

#### 6. Beads

```bash
# Install Beads
curl -fsSL https://raw.githubusercontent.com/steveyegge/beads/main/scripts/install.sh | bash

# Verify installation
bd --version
```

#### 7. PostgreSQL Client (Optional, for database management)

```bash
# Install PostgreSQL client tools
sudo apt-get install -y postgresql-client

# Verify installation
psql --version
```

### Environment Variables

Create a `.env` file in the project root for sensitive configuration:

```bash
# API Keys
ANTHROPIC_API_KEY=sk-ant-...                 # For Claude API (if using Anthropic)
OPENCODE_PROVIDER=opencode-zen                # Provider selection
OPENCODE_API_KEY=...                          # Provider-specific key

# Database
DATABASE_URL=postgresql://user:pass@localhost:5432/open_swarm

# Deployment
ENVIRONMENT=development                        # development|staging|production
LOG_LEVEL=info                                 # debug|info|warn|error
PORT=8080                                      # Application port
```

**Security Note:** Never commit `.env` to version control. It's already in `.gitignore`.

---

## Development Setup

### Local Installation

```bash
# 1. Clone the repository
git clone <repository-url>
cd open-swarm

# 2. Install Go dependencies
go mod download

# 3. Create environment file
cp .env.example .env
# Edit .env with your settings

# 4. Initialize Beads (if not already done)
bd init

# 5. Build the binaries
go build -o bin/open-swarm ./cmd/open-swarm
go build -o bin/reactor ./cmd/reactor
go build -o bin/temporal-worker ./cmd/temporal-worker
go build -o bin/reactor-client ./cmd/reactor-client

# 6. Start Temporal services (in separate terminal)
docker-compose up -d

# 7. Verify services are healthy
docker-compose ps

# 8. Run tests
go test ./...

# 9. Start Agent Mail server (in separate terminal)
am

# 10. Start your application
./bin/open-swarm --config opencode.json
```

### Docker Compose for Development

The included `docker-compose.yml` provides Temporal infrastructure:

```bash
# Start Temporal and PostgreSQL
docker-compose up -d

# View logs
docker-compose logs -f

# Stop services
docker-compose down

# Reset database (warning: deletes all data)
docker-compose down -v
docker-compose up -d
```

**Services:**
- **PostgreSQL:** Port 5432 (credentials in docker-compose.yml)
- **Temporal Server:** Port 7233 (gRPC)
- **Temporal Web UI:** Port 8233 (http://localhost:8233)

### Development Workflow

```bash
# Terminal 1: Services
docker-compose up -d
am  # Start Agent Mail server

# Terminal 2: Main application
go build -o bin/open-swarm ./cmd/open-swarm
./bin/open-swarm --config opencode.json

# Terminal 3: Reactor (if needed)
go build -o bin/reactor ./cmd/reactor
./bin/reactor --task "task-id" --desc "Task description"

# Terminal 4: Tests and development
go test -v ./...
gofmt -w .
golangci-lint run

# Terminal 5: Agent coordination
opencode
/session-start
```

### Verification Checklist

- [ ] Go 1.25+ installed
- [ ] Docker and Docker Compose running
- [ ] PostgreSQL accessible on localhost:5432
- [ ] Temporal Server healthy (port 7233)
- [ ] Agent Mail server running (port 8765)
- [ ] Tests passing (`go test ./...`)
- [ ] Binaries built successfully
- [ ] Beads initialized (`bd list` returns issues)

---

## Production Deployment

### Pre-Deployment Checklist

Before deploying to production:

```bash
# 1. Run all tests
go test -v ./...

# 2. Build with optimizations
go build -ldflags="-s -w" -o bin/open-swarm ./cmd/open-swarm
go build -ldflags="-s -w" -o bin/reactor ./cmd/reactor
go build -ldflags="-s -w" -o bin/temporal-worker ./cmd/temporal-worker

# 3. Run security checks
go list -json -m all | nancy sleuth
golangci-lint run

# 4. Run benchmarks (if applicable)
go test -bench=. ./...

# 5. Verify configuration
opencode.json is present and valid
.env contains all required variables
Database connection string is correct
```

### Docker-Based Deployment

#### Build Production Docker Image

Create `Dockerfile` in project root:

```dockerfile
# Multi-stage build
FROM golang:1.25-alpine AS builder

WORKDIR /build
COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o open-swarm ./cmd/open-swarm
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o reactor ./cmd/reactor
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o temporal-worker ./cmd/temporal-worker

# Final image
FROM alpine:latest
RUN apk --no-cache add ca-certificates
WORKDIR /app

COPY --from=builder /build/open-swarm .
COPY --from=builder /build/reactor .
COPY --from=builder /build/temporal-worker .
COPY opencode.json .
COPY .opencode/ ./.opencode/

EXPOSE 8080
HEALTHCHECK --interval=30s --timeout=3s --start-period=5s --retries=3 \
    CMD wget --quiet --tries=1 --spider http://localhost:8080/health || exit 1

CMD ["./open-swarm", "--config", "opencode.json"]
```

**Build the image:**

```bash
docker build -t open-swarm:latest .
docker tag open-swarm:latest open-swarm:v1.0.0
```

#### Production Docker Compose

Create `docker-compose.production.yml`:

```yaml
version: '3.8'

services:
  postgresql:
    image: postgres:13-alpine
    container_name: open-swarm-db
    environment:
      POSTGRES_DB: open_swarm
      POSTGRES_USER: ${DB_USER:-swarm}
      POSTGRES_PASSWORD: ${DB_PASSWORD}
    volumes:
      - postgres_data:/var/lib/postgresql/data
      - ./scripts/init.sql:/docker-entrypoint-initdb.d/init.sql
    ports:
      - "5432:5432"
    healthcheck:
      test: ["CMD-SHELL", "pg_isready -U ${DB_USER:-swarm}"]
      interval: 10s
      timeout: 5s
      retries: 5
    restart: unless-stopped
    networks:
      - open-swarm-network
    labels:
      com.example.description: "PostgreSQL database for Open Swarm"

  temporal:
    image: temporalio/auto-setup:latest
    container_name: open-swarm-temporal
    depends_on:
      postgresql:
        condition: service_healthy
    environment:
      - DB=postgresql
      - DB_PORT=5432
      - POSTGRES_USER=${DB_USER:-swarm}
      - POSTGRES_PWD=${DB_PASSWORD}
      - POSTGRES_SEEDS=postgresql
      - TEMPORAL_CORS_ORIGINS=*
    ports:
      - "7233:7233"
      - "8233:8233"
    healthcheck:
      test: ["CMD", "tctl", "--address", "temporal:7233", "cluster", "health"]
      interval: 10s
      timeout: 5s
      retries: 10
      start_period: 30s
    restart: unless-stopped
    networks:
      - open-swarm-network
    labels:
      com.example.description: "Temporal server for Open Swarm"

  open-swarm:
    build: .
    container_name: open-swarm-app
    depends_on:
      temporal:
        condition: service_healthy
    environment:
      ENVIRONMENT: production
      LOG_LEVEL: ${LOG_LEVEL:-info}
      PORT: 8080
      TEMPORAL_HOST: temporal
      TEMPORAL_PORT: 7233
      DATABASE_URL: postgresql://${DB_USER:-swarm}:${DB_PASSWORD}@postgresql:5432/open_swarm
      ANTHROPIC_API_KEY: ${ANTHROPIC_API_KEY}
    ports:
      - "8080:8080"
    volumes:
      - ./logs:/app/logs
      - opencode_cache:/home/opencode/.cache
    healthcheck:
      test: ["CMD", "wget", "--quiet", "--tries=1", "--spider", "http://localhost:8080/health"]
      interval: 30s
      timeout: 3s
      retries: 3
      start_period: 5s
    restart: unless-stopped
    networks:
      - open-swarm-network
    labels:
      com.example.description: "Open Swarm application server"

  agent-mail:
    image: mcp-agent-mail:latest
    container_name: open-swarm-agent-mail
    environment:
      AGENT_MAIL_PORT: 8765
      DATABASE_URL: postgresql://${DB_USER:-swarm}:${DB_PASSWORD}@postgresql:5432/agent_mail
    ports:
      - "8765:8765"
    healthcheck:
      test: ["CMD", "curl", "-f", "http://localhost:8765/health"]
      interval: 30s
      timeout: 3s
      retries: 3
    restart: unless-stopped
    networks:
      - open-swarm-network
    labels:
      com.example.description: "Agent Mail MCP server"

volumes:
  postgres_data:
    driver: local
  opencode_cache:
    driver: local

networks:
  open-swarm-network:
    driver: bridge
```

**Deploy:**

```bash
# Create production environment file
cp .env.example .env.production
# Edit .env.production with production values

# Deploy using docker-compose
docker-compose -f docker-compose.production.yml -p open-swarm up -d

# Monitor
docker-compose -f docker-compose.production.yml logs -f open-swarm

# Stop
docker-compose -f docker-compose.production.yml down
```

### Systemd Service Files

Create systemd service files for bare-metal deployment.

#### Main Application Service

Create `/etc/systemd/system/open-swarm.service`:

```ini
[Unit]
Description=Open Swarm - Multi-Agent Coordination Framework
Documentation=https://github.com/example/open-swarm
After=network.target temporal.service postgresql.service agent-mail.service
Wants=temporal.service postgresql.service agent-mail.service

[Service]
Type=simple
User=openswarm
Group=openswarm
WorkingDirectory=/opt/open-swarm
EnvironmentFile=/etc/open-swarm/open-swarm.env

# Environment variables
Environment="ENVIRONMENT=production"
Environment="LOG_LEVEL=info"
Environment="PORT=8080"

ExecStart=/opt/open-swarm/bin/open-swarm --config /etc/open-swarm/opencode.json

# Process management
Restart=on-failure
RestartSec=10
KillMode=mixed
KillSignal=SIGTERM
TimeoutStopSec=30

# Security
NoNewPrivileges=true
PrivateTmp=true
ProtectSystem=strict
ProtectHome=yes
ReadWritePaths=/opt/open-swarm/logs /opt/open-swarm/data

# Resource limits
LimitNOFILE=65536
LimitNPROC=32768
MemoryMax=2G
CPUQuota=200%

[Install]
WantedBy=multi-user.target
```

#### Temporal Worker Service

Create `/etc/systemd/system/open-swarm-temporal-worker.service`:

```ini
[Unit]
Description=Open Swarm Temporal Worker
After=temporal.service
Wants=temporal.service

[Service]
Type=simple
User=openswarm
Group=openswarm
WorkingDirectory=/opt/open-swarm
EnvironmentFile=/etc/open-swarm/temporal-worker.env

Environment="TEMPORAL_HOST=localhost"
Environment="TEMPORAL_PORT=7233"
Environment="LOG_LEVEL=info"

ExecStart=/opt/open-swarm/bin/temporal-worker

Restart=on-failure
RestartSec=10
KillMode=mixed
KillSignal=SIGTERM

[Install]
WantedBy=multi-user.target
```

#### PostgreSQL Service

Create `/etc/systemd/system/open-swarm-postgresql.service`:

```ini
[Unit]
Description=Open Swarm PostgreSQL Database
Documentation=https://www.postgresql.org/docs/
After=network.target
Before=open-swarm.service

[Service]
Type=notify
User=postgres
ExecStart=/usr/lib/postgresql/13/bin/postgres -D /var/lib/postgresql/13/main -c config_file=/etc/postgresql/13/main/postgresql.conf
ExecReload=/bin/kill -HUP $MAINPID
KillMode=mixed
Restart=on-failure
RestartSec=10

[Install]
WantedBy=multi-user.target
```

#### Temporal Server Service

Create `/etc/systemd/system/open-swarm-temporal.service`:

```ini
[Unit]
Description=Open Swarm Temporal Server
After=open-swarm-postgresql.service
Wants=open-swarm-postgresql.service

[Service]
Type=simple
User=temporal
WorkingDirectory=/opt/temporal
EnvironmentFile=/etc/open-swarm/temporal.env

Environment="DB=postgresql"
Environment="POSTGRES_SEEDS=localhost"
Environment="TEMPORAL_CORS_ORIGINS=*"

ExecStart=/opt/temporal/bin/temporal-server

Restart=on-failure
RestartSec=10
KillMode=mixed

[Install]
WantedBy=multi-user.target
```

### Installation

```bash
# 1. Create service user
sudo useradd --system --home /opt/open-swarm --shell /bin/bash openswarm

# 2. Copy binaries and configuration
sudo mkdir -p /opt/open-swarm/{bin,logs,data}
sudo cp bin/* /opt/open-swarm/bin/
sudo cp opencode.json /etc/open-swarm/
sudo cp .opencode /etc/open-swarm/

# 3. Create environment files
sudo mkdir -p /etc/open-swarm
sudo cp .env.production /etc/open-swarm/open-swarm.env
sudo chmod 600 /etc/open-swarm/open-swarm.env

# 4. Copy service files
sudo cp /tmp/open-swarm*.service /etc/systemd/system/

# 5. Set permissions
sudo chown -R openswarm:openswarm /opt/open-swarm
sudo chown -R openswarm:openswarm /etc/open-swarm
sudo chmod 644 /etc/systemd/system/open-swarm*.service

# 6. Enable services
sudo systemctl daemon-reload
sudo systemctl enable open-swarm-postgresql.service
sudo systemctl enable open-swarm-temporal.service
sudo systemctl enable open-swarm.service

# 7. Start services
sudo systemctl start open-swarm-postgresql.service
sleep 10
sudo systemctl start open-swarm-temporal.service
sleep 10
sudo systemctl start open-swarm.service

# 8. Verify
sudo systemctl status open-swarm.service
```

### Service Management

```bash
# Start services
sudo systemctl start open-swarm

# Stop services
sudo systemctl stop open-swarm

# Restart services
sudo systemctl restart open-swarm

# View logs
sudo journalctl -u open-swarm -f

# Check status
sudo systemctl status open-swarm
sudo systemctl list-units --type=service --state=running | grep open-swarm
```

---

## Scaling

### Horizontal Scaling

#### Load Balancer Configuration

Use Nginx as a reverse proxy:

```nginx
# /etc/nginx/sites-available/open-swarm
upstream open_swarm_backend {
    least_conn;
    server localhost:8080 weight=1 max_fails=3 fail_timeout=30s;
    server localhost:8081 weight=1 max_fails=3 fail_timeout=30s;
    server localhost:8082 weight=1 max_fails=3 fail_timeout=30s;
}

server {
    listen 80;
    server_name open-swarm.example.com;

    client_max_body_size 100M;

    location / {
        proxy_pass http://open_swarm_backend;
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto $scheme;

        # WebSocket support
        proxy_http_version 1.1;
        proxy_set_header Upgrade $http_upgrade;
        proxy_set_header Connection "upgrade";

        # Timeouts for long-running tasks
        proxy_connect_timeout 60s;
        proxy_send_timeout 300s;
        proxy_read_timeout 300s;
    }

    location /health {
        proxy_pass http://open_swarm_backend;
        access_log off;
    }
}
```

**Enable:**

```bash
sudo ln -s /etc/nginx/sites-available/open-swarm /etc/nginx/sites-enabled/
sudo nginx -t
sudo systemctl restart nginx
```

#### Multiple Instance Deployment

Deploy multiple application instances on different ports:

```bash
# Instance 1
PORT=8080 ./bin/open-swarm --config opencode.json

# Instance 2 (different terminal)
PORT=8081 ./bin/open-swarm --config opencode.json

# Instance 3 (different terminal)
PORT=8082 ./bin/open-swarm --config opencode.json
```

Or using systemd:

```bash
# Create instances
sudo cp /etc/systemd/system/open-swarm.service /etc/systemd/system/open-swarm@.service

# Edit @instance version to support variables
# Environment="PORT=%i080"

# Enable instances
sudo systemctl enable open-swarm@8080.service
sudo systemctl enable open-swarm@8081.service
sudo systemctl enable open-swarm@8082.service

# Start all
sudo systemctl start open-swarm@{8080,8081,8082}.service

# Check status
sudo systemctl status open-swarm@*.service
```

### Database Scaling

#### Connection Pooling

Configure connection pooling in `opencode.json`:

```json
{
  "database": {
    "max_connections": 50,
    "min_idle": 5,
    "max_lifetime": 1800,
    "idle_timeout": 300
  }
}
```

#### Read Replicas

For read-heavy workloads, configure PostgreSQL read replicas:

```bash
# On primary (master)
# Edit postgresql.conf:
# wal_level = replica
# max_wal_senders = 3
# wal_keep_size = 1GB

# On replica (standby)
pg_basebackup -h master.example.com -D /var/lib/postgresql/13/main -U replicator -v -P

# Recovery config
echo "primary_conninfo = 'host=master.example.com user=replicator password=secret'" > recovery.signal
```

Update application connection string for read operations:

```json
{
  "database": {
    "write_url": "postgresql://user:pass@master.example.com:5432/open_swarm",
    "read_url": "postgresql://user:pass@replica.example.com:5432/open_swarm"
  }
}
```

### Caching

#### In-Process Caching

Configure caching in application code:

```bash
# Cache Temporal workflow results
# TTL: 5 minutes for hot workflows
# Size limit: 100MB

# OpenCode responses
# TTL: 1 hour for stable responses
# Size limit: 500MB
```

#### Distributed Caching (Redis)

For multi-instance deployments:

```bash
# Start Redis
docker run -d --name redis -p 6379:6379 redis:7-alpine

# Configure in application
# REDIS_URL=redis://localhost:6379/0
```

### Auto-Scaling

#### With Kubernetes

Deploy using Kubernetes for automatic scaling:

```yaml
# deployment.yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: open-swarm
spec:
  replicas: 3
  strategy:
    type: RollingUpdate
    rollingUpdate:
      maxSurge: 1
      maxUnavailable: 0
  selector:
    matchLabels:
      app: open-swarm
  template:
    metadata:
      labels:
        app: open-swarm
    spec:
      containers:
      - name: open-swarm
        image: open-swarm:v1.0.0
        ports:
        - containerPort: 8080
        env:
        - name: ENVIRONMENT
          value: production
        - name: TEMPORAL_HOST
          value: temporal
        resources:
          requests:
            memory: "512Mi"
            cpu: "500m"
          limits:
            memory: "2Gi"
            cpu: "2000m"
        livenessProbe:
          httpGet:
            path: /health
            port: 8080
          initialDelaySeconds: 10
          periodSeconds: 30
        readinessProbe:
          httpGet:
            path: /ready
            port: 8080
          initialDelaySeconds: 5
          periodSeconds: 10

---
apiVersion: autoscaling/v2
kind: HorizontalPodAutoscaler
metadata:
  name: open-swarm-hpa
spec:
  scaleTargetRef:
    apiVersion: apps/v1
    kind: Deployment
    name: open-swarm
  minReplicas: 3
  maxReplicas: 10
  metrics:
  - type: Resource
    resource:
      name: cpu
      target:
        type: Utilization
        averageUtilization: 70
  - type: Resource
    resource:
      name: memory
      target:
        type: Utilization
        averageUtilization: 80
```

---

## Monitoring

### Application Metrics

#### Health Check Endpoint

```bash
# Basic health check
curl http://localhost:8080/health

# Response
{
  "status": "healthy",
  "timestamp": "2024-01-15T10:30:00Z",
  "components": {
    "database": "healthy",
    "temporal": "healthy",
    "agent_mail": "healthy"
  }
}
```

#### Readiness Check

```bash
# Readiness check
curl http://localhost:8080/ready

# Response
{
  "ready": true,
  "reason": "All dependencies available"
}
```

### Logging

#### Log Levels

Set `LOG_LEVEL` environment variable:

```bash
export LOG_LEVEL=debug    # Verbose logging
export LOG_LEVEL=info     # Normal logging (recommended)
export LOG_LEVEL=warn     # Warnings only
export LOG_LEVEL=error    # Errors only
```

#### Log Format

Logs are output as JSON for easy parsing:

```json
{
  "timestamp": "2024-01-15T10:30:00Z",
  "level": "info",
  "component": "workflow.engine",
  "message": "Workflow execution started",
  "task_id": "open-swarm-axu.1.12",
  "duration_ms": 150,
  "trace_id": "abc123def456"
}
```

#### Log Aggregation

##### Using ELK Stack (Elasticsearch, Logstash, Kibana)

1. **Deploy ELK:**

```yaml
version: '3.8'
services:
  elasticsearch:
    image: docker.elastic.co/elasticsearch/elasticsearch:8.5.0
    environment:
      - discovery.type=single-node
      - xpack.security.enabled=false
    ports:
      - "9200:9200"

  logstash:
    image: docker.elastic.co/logstash/logstash:8.5.0
    volumes:
      - ./logstash.conf:/usr/share/logstash/pipeline/logstash.conf
    ports:
      - "5000:5000"

  kibana:
    image: docker.elastic.co/kibana/kibana:8.5.0
    ports:
      - "5601:5601"
```

2. **Configure Logstash (logstash.conf):**

```
input {
  tcp {
    port => 5000
    codec => json
  }
}

filter {
  if [type] == "open-swarm" {
    mutate {
      add_field => { "[@metadata][index_name]" => "open-swarm-%{+YYYY.MM.dd}" }
    }
  }
}

output {
  elasticsearch {
    hosts => ["elasticsearch:9200"]
    index => "%{[@metadata][index_name]}"
  }
}
```

3. **Configure Application Logging:**

Update application to send logs to Logstash on port 5000.

### Monitoring Tools

#### Prometheus Metrics

Expose metrics for Prometheus scraping:

```bash
# Endpoint: /metrics
curl http://localhost:8080/metrics

# Output (Prometheus format)
# HELP open_swarm_workflows_total Total workflows executed
# TYPE open_swarm_workflows_total counter
open_swarm_workflows_total{status="completed"} 1042
open_swarm_workflows_total{status="failed"} 23
open_swarm_workflows_total{status="retried"} 5
```

#### Grafana Dashboard

Create a Grafana dashboard to visualize metrics:

```json
{
  "dashboard": {
    "title": "Open Swarm Metrics",
    "panels": [
      {
        "title": "Workflows Per Second",
        "targets": [
          {
            "expr": "rate(open_swarm_workflows_total[5m])"
          }
        ]
      },
      {
        "title": "Average Execution Time",
        "targets": [
          {
            "expr": "histogram_quantile(0.95, open_swarm_workflow_duration_seconds_bucket)"
          }
        ]
      },
      {
        "title": "Error Rate",
        "targets": [
          {
            "expr": "rate(open_swarm_workflows_total{status=\"failed\"}[5m])"
          }
        ]
      }
    ]
  }
}
```

### Alerting

#### Alert Rules

Create alert rules in `/etc/prometheus/rules/open-swarm.yml`:

```yaml
groups:
  - name: open-swarm
    rules:
      - alert: HighErrorRate
        expr: rate(open_swarm_workflows_total{status="failed"}[5m]) > 0.05
        for: 5m
        labels:
          severity: warning
        annotations:
          summary: "High error rate detected"
          description: "Error rate is {{ $value }} errors/sec"

      - alert: DatabaseDown
        expr: up{job="postgresql"} == 0
        for: 1m
        labels:
          severity: critical
        annotations:
          summary: "PostgreSQL is down"

      - alert: TemporalServerDown
        expr: up{job="temporal"} == 0
        for: 1m
        labels:
          severity: critical
        annotations:
          summary: "Temporal server is down"
```

#### Notification Channels

Configure where alerts are sent:

```yaml
# Alertmanager config (/etc/prometheus/alertmanager.yml)
global:
  resolve_timeout: 5m

route:
  receiver: 'default'
  group_by: ['alertname', 'cluster', 'service']
  group_wait: 10s
  group_interval: 10s
  repeat_interval: 12h
  routes:
    - match:
        severity: critical
      receiver: 'pagerduty'
    - match:
        severity: warning
      receiver: 'slack'

receivers:
  - name: 'default'
    email_configs:
      - to: 'ops@example.com'

  - name: 'slack'
    slack_configs:
      - api_url: 'https://hooks.slack.com/services/YOUR/WEBHOOK/URL'
        channel: '#alerts'

  - name: 'pagerduty'
    pagerduty_configs:
      - service_key: 'YOUR_PAGERDUTY_SERVICE_KEY'
```

### Operational Dashboards

#### Service Status Page

```bash
# Command to check all services
sudo systemctl status 'open-swarm*' --all

# Expected output
● open-swarm.service - Open Swarm...
    Loaded: loaded (/etc/systemd/system/open-swarm.service; enabled; vendor preset: enabled)
    Active: active (running) since Mon 2024-01-15 10:30:00 UTC; 12h ago

● open-swarm-postgresql.service - Open Swarm PostgreSQL...
    Loaded: loaded (/etc/systemd/system/open-swarm-postgresql.service; enabled; vendor preset: enabled)
    Active: active (running) since Mon 2024-01-15 10:29:00 UTC; 12h ago

● open-swarm-temporal.service - Open Swarm Temporal Server...
    Loaded: loaded (/etc/systemd/system/open-swarm-temporal.service; enabled; vendor preset: enabled)
    Active: active (running) since Mon 2024-01-15 10:28:00 UTC; 12h ago
```

---

## Security

### Network Security

#### Firewall Configuration

```bash
# Enable UFW (Ubuntu)
sudo ufw enable

# Allow SSH
sudo ufw allow 22/tcp

# Allow HTTP/HTTPS (if behind load balancer)
sudo ufw allow 80/tcp
sudo ufw allow 443/tcp

# Allow Temporal (internal only)
sudo ufw allow from 10.0.0.0/8 to any port 7233

# Allow PostgreSQL (internal only)
sudo ufw allow from 10.0.0.0/8 to any port 5432

# Deny everything else
sudo ufw default deny incoming
sudo ufw default allow outgoing
```

#### SSL/TLS Configuration

```bash
# Obtain certificate (Let's Encrypt)
sudo certbot certonly --standalone -d open-swarm.example.com

# Configure Nginx
server {
    listen 443 ssl http2;
    ssl_certificate /etc/letsencrypt/live/open-swarm.example.com/fullchain.pem;
    ssl_certificate_key /etc/letsencrypt/live/open-swarm.example.com/privkey.pem;

    # Strong SSL settings
    ssl_protocols TLSv1.2 TLSv1.3;
    ssl_ciphers HIGH:!aNULL:!MD5;
    ssl_prefer_server_ciphers on;
    ssl_session_timeout 1d;
    ssl_session_cache shared:SSL:50m;
    ssl_session_tickets off;

    # HSTS
    add_header Strict-Transport-Security "max-age=31536000; includeSubDomains" always;
}
```

### Authentication & Authorization

#### API Key Authentication

Store API keys securely:

```bash
# Generate API key
openssl rand -hex 32

# Store in environment
export OPENCODE_API_KEY=<generated-key>

# Validate in requests
curl -H "Authorization: Bearer $OPENCODE_API_KEY" http://localhost:8080/api/tasks
```

#### Service-to-Service Authentication

Configure mutual TLS (mTLS) for internal service communication:

```bash
# Generate certificates
cfssl gencert -initca ca.json | cfssljson -bare ca
cfssl gencert -config=ca-config.json -profile=server server.json | cfssljson -bare server
cfssl gencert -config=ca-config.json -profile=client client.json | cfssljson -bare client

# Configure Temporal client
# Use ca.pem, server.pem, server-key.pem for secure connection
```

### Data Security

#### Database Encryption

Enable PostgreSQL data encryption:

```bash
# Enable at-rest encryption
# In postgresql.conf:
# ssl = on
# ssl_cert_file = '/etc/postgresql/server.crt'
# ssl_key_file = '/etc/postgresql/server.key'
# password_encryption = scram-sha-256

# Restart PostgreSQL
sudo systemctl restart postgresql
```

#### Secrets Management

Use environment variables for sensitive data:

```bash
# Store secrets in secure location
sudo mkdir -p /etc/open-swarm/secrets
sudo chmod 700 /etc/open-swarm/secrets

# Example: API key
echo "sk-ant-xxxxx" | sudo tee /etc/open-swarm/secrets/anthropic_api_key
sudo chmod 600 /etc/open-swarm/secrets/anthropic_api_key

# Load in systemd service
EnvironmentFile=/etc/open-swarm/secrets/*
```

Or use HashiCorp Vault:

```bash
# Initialize Vault
vault operator init
vault operator unseal

# Store secrets
vault kv put secret/open-swarm \
  anthropic_api_key="sk-ant-xxxxx" \
  db_password="secure-password"

# Retrieve in application
# Vault auto-rotate credentials
# Audit all access
```

### Access Control

#### User Roles

Implement role-based access control (RBAC):

```json
{
  "roles": {
    "admin": {
      "permissions": ["read:*", "write:*", "delete:*", "manage:users"]
    },
    "developer": {
      "permissions": ["read:*", "write:workflows", "write:tasks"]
    },
    "viewer": {
      "permissions": ["read:*"]
    }
  }
}
```

#### API Rate Limiting

Prevent abuse with rate limiting:

```bash
# Per-IP limit: 1000 requests/hour
# Per-API-key limit: 10000 requests/hour
# Burst limit: 100 requests/second

# Configure in Nginx
limit_req_zone $binary_remote_addr zone=general:10m rate=100r/s;
limit_req zone=general burst=200 nodelay;
```

### Audit Logging

Enable comprehensive audit logging:

```bash
# Log all authentication attempts
# Log all configuration changes
# Log all data access
# Retain logs for 90 days

# Enable in application
export AUDIT_LOGGING=true
export AUDIT_LOG_RETENTION=90
```

Sample audit log entry:

```json
{
  "timestamp": "2024-01-15T10:30:00Z",
  "event_type": "api_call",
  "user_id": "user-123",
  "action": "create_workflow",
  "resource": "workflow-456",
  "status": "success",
  "ip_address": "192.168.1.1",
  "user_agent": "OpenCode/1.0"
}
```

### Vulnerability Management

#### Dependency Updates

Keep dependencies up to date:

```bash
# Check for vulnerable dependencies
go list -json -m all | nancy sleuth

# Update dependencies
go get -u ./...

# Run security audit
govulncheck ./...
```

#### Regular Security Audits

```bash
# 1. Dependency audit
go list -json -m all | nancy sleuth

# 2. Code scanning
golangci-lint run --enable gosec

# 3. Container scanning
trivy image open-swarm:latest

# 4. Infrastructure as Code scanning
trivy config docker-compose.yml
```

### Compliance

#### GDPR Compliance

Implement data privacy features:

```bash
# Data retention policies
# User consent management
# Data export functionality
# Right to be forgotten (data deletion)
# Audit trails for data access
```

#### SOC 2 Compliance

Required controls:

```bash
# Change management procedures
# Access control logging
# Backup and disaster recovery
# Security incident response
# Regular security assessments
# Employee training and awareness
```

---

## Troubleshooting

### Common Issues

#### Service Won't Start

```bash
# Check logs
sudo journalctl -u open-swarm -n 50
tail -f /var/log/open-swarm.log

# Verify configuration
/opt/open-swarm/bin/open-swarm --validate-config

# Check dependencies
sudo systemctl status open-swarm-postgresql.service
sudo systemctl status open-swarm-temporal.service
```

#### Database Connection Issues

```bash
# Test connection
psql -h localhost -U swarm -d open_swarm

# Check PostgreSQL status
sudo systemctl status postgresql
sudo -u postgres psql -c "SELECT 1"

# Verify environment variables
echo $DATABASE_URL
```

#### Temporal Server Issues

```bash
# Check Temporal health
curl -v http://localhost:8233/health

# View Temporal logs
docker logs open-swarm-temporal

# Restart Temporal
sudo systemctl restart open-swarm-temporal.service
```

### Performance Optimization

#### Database Query Optimization

```bash
# Enable query logging
# ALTER SYSTEM SET log_min_duration_statement = 1000;  -- Log queries > 1s
# SELECT pg_reload_conf();

# Analyze query plans
EXPLAIN ANALYZE SELECT * FROM workflows WHERE status = 'completed';

# Add missing indexes
CREATE INDEX idx_workflows_status ON workflows(status);
CREATE INDEX idx_workflows_created_at ON workflows(created_at DESC);
```

#### Connection Pool Tuning

Adjust in `/etc/open-swarm/open-swarm.env`:

```bash
# Increase pool size for high concurrency
DATABASE_MAX_CONNECTIONS=100
DATABASE_MIN_IDLE=10
```

---

## References

- [Go Installation Guide](https://go.dev/dl/)
- [Docker Documentation](https://docs.docker.com/)
- [PostgreSQL Documentation](https://www.postgresql.org/docs/)
- [Temporal Documentation](https://temporal.io/docs/)
- [Nginx Documentation](https://nginx.org/en/docs/)
- [Systemd Documentation](https://www.freedesktop.org/software/systemd/man/)
- [Let's Encrypt](https://letsencrypt.org/)
- [Prometheus Documentation](https://prometheus.io/docs/)
- [Grafana Documentation](https://grafana.com/docs/)

---

## Support

For issues or questions:

1. Check the [README.md](/README.md) for general information
2. Review [AGENTS.md](/AGENTS.md) for development workflows
3. Check existing [Beads issues](https://github.com/example/open-swarm/issues)
4. File a new issue with:
   - Environment details (OS, Go version, Docker version)
   - Error messages and logs
   - Steps to reproduce
   - Expected vs actual behavior
