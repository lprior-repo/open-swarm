# OpenTelemetry Tracing Guide

## Overview

This document describes the OpenTelemetry instrumentation in Open Swarm, which provides distributed tracing for all workflow activities, OpenCode SDK calls, and TCR operations.

## Quick Start

### 1. Start the Observability Stack

```bash
# Start OpenTelemetry Collector, Jaeger, Prometheus, and Grafana
docker-compose -f docker-compose.otel.yml up -d

# Verify services are running
docker-compose -f docker-compose.otel.yml ps
```

### 2. Run the Worker with Tracing

```bash
# Start the worker (tracing is enabled by default)
./bin/temporal-worker

# Or with custom collector URL
OTEL_COLLECTOR_URL=http://localhost:4318 ./bin/temporal-worker
```

### 3. View Traces

Open your browser and navigate to:

- **Jaeger UI**: http://localhost:16686
- **Grafana**: http://localhost:3001 (admin/admin)
- **Prometheus**: http://localhost:9090

### 4. Run Benchmarks and Generate Traces

```bash
# Run a basic TCR workflow
./bin/simple-benchmark \
  -strategy basic \
  -runs 3 \
  -prompt "Implement FizzBuzz"

# Run an enhanced TCR workflow
./bin/simple-benchmark \
  -strategy enhanced \
  -runs 1 \
  -prompt "Implement a prime number checker"
```

## Architecture

### Components

1. **Application (open-swarm)**
   - Instruments all OpenCode SDK calls
   - Instruments all Temporal activities
   - Instruments TCR workflow operations
   - Exports traces to OTLP collector via HTTP (port 4318)

2. **OpenTelemetry Collector**
   - Receives traces from applications
   - Processes and batches traces
   - Exports to multiple backends (Jaeger, file, console)
   - Exposes metrics about itself

3. **Jaeger**
   - Stores and indexes traces
   - Provides UI for trace visualization
   - Supports trace search and filtering

4. **Prometheus** (optional)
   - Scrapes metrics from collector
   - Stores time-series metrics data

5. **Grafana** (optional)
   - Visualizes metrics and traces
   - Provides dashboards
   - Links traces to metrics

### Trace Flow

```
Open Swarm Application
    ↓ (OTLP/HTTP on port 4318)
OpenTelemetry Collector
    ↓ (processes, batches)
    ├→ Jaeger (trace storage)
    ├→ File (./otel-traces.json)
    └→ Console (stdout)
```

## Instrumented Operations

### OpenCode SDK Operations

All OpenCode client operations are traced with detailed spans:

- **ExecutePrompt**: LLM prompt execution
  - Attributes: session_id, model, agent, prompt_length, response_parts
  - Events: prompt.start, session.created, prompt.completed
  
- **ExecuteCommand**: Shell/command execution
  - Attributes: command, args, session_id
  - Events: command.start, command.completed

- **Session Management**: Create, delete, abort sessions
  - Events: session.created, session.deleted, session.aborted

### Temporal Activities

All Temporal activities are traced with workflow context:

- **Workflow Attributes**: workflow_id, workflow_type, run_id
- **Activity Attributes**: activity_id, activity_type
- **Duration Metrics**: duration_ms for all operations

### TCR Workflow Gates

Each gate in the Enhanced TCR workflow is fully traced:

1. **GenTest** (Gate 1): Test generation
   - Attributes: gate_name, gate_passed, files_changed
   - Events: gate.start, gate.passed/failed

2. **LintTest** (Gate 2): Linting tests
   - Attributes: lint.issues
   - Events: gate.start, gate.passed/failed

3. **VerifyRED** (Gate 3): Verify tests fail
   - Attributes: tests_passed, tests_failed
   - Events: gate.start, gate.passed/failed

4. **GenImpl** (Gate 4): Implementation generation
   - Attributes: retry (bool), files_changed
   - Events: gate.start, impl.retry, gate.passed/failed

5. **VerifyGREEN** (Gate 5): Verify tests pass
   - Attributes: tests_passed, tests_failed
   - Events: gate.start, gate.passed/failed

6. **MultiReview** (Gate 6): Multi-reviewer approval
   - Attributes: reviews.total, reviews.approved, reviews.rejected
   - Events: gate.start, review.completed, gate.passed/failed

### Custom Attributes

The following custom attributes are added to spans:

#### Temporal
- `workflow.id` - Temporal workflow ID
- `workflow.type` - Workflow type name
- `workflow.run_id` - Workflow run ID
- `activity.id` - Activity ID
- `activity.type` - Activity type name
- `temporal.task_queue` - Task queue name

#### OpenCode
- `opencode.session_id` - OpenCode session ID
- `opencode.prompt` - Prompt text (truncated)
- `opencode.model` - Model identifier
- `opencode.agent` - Agent name
- `opencode.files_modified` - Number of files modified
- `opencode.response_length` - Response length

#### TCR
- `tcr.branch` - Git branch name
- `tcr.task_id` - Task identifier
- `tcr.gate_name` - Gate name (gen_test, verify_red, etc.)
- `tcr.gate_passed` - Whether gate passed (bool)
- `tcr.tests_passed` - Number of tests passed
- `tcr.tests_failed` - Number of tests failed
- `tcr.review_vote` - Review vote (approve/reject/request_change)

#### General
- `error` - Error flag (bool)
- `error.message` - Error message text
- `duration_ms` - Operation duration in milliseconds
- `success` - Success flag (bool)

## Configuration

### Environment Variables

```bash
# OpenTelemetry Collector URL (default: http://localhost:4318)
export OTEL_COLLECTOR_URL=http://localhost:4318

# Service name (default: open-swarm)
export OTEL_SERVICE_NAME=open-swarm

# Sampling rate (default: 1.0 = 100%)
export OTEL_SAMPLING_RATE=1.0
```

### Collector Configuration

Edit `otel-collector-config.yaml` to customize:

```yaml
receivers:
  otlp:
    protocols:
      http:
        endpoint: 0.0.0.0:4318  # Change port if needed

exporters:
  logging:
    loglevel: info  # debug, info, warn, error
  file:
    path: ./otel-traces.json  # Change output file
```

### Application Configuration

The telemetry package is initialized in `cmd/temporal-worker/main.go`:

```go
config := telemetry.DefaultConfig()
config.CollectorURL = "http://localhost:4318"
config.SamplingRate = 1.0  // Sample all traces

tracerProvider, err := telemetry.NewTracerProvider(ctx, config)
```

## Viewing Traces

### Jaeger UI (Recommended)

1. Open http://localhost:16686
2. Select service: `open-swarm`
3. Select operation (optional):
   - `ExecutePrompt` - OpenCode prompts
   - `ExecuteGenTest` - Test generation
   - `ExecuteVerifyGREEN` - Test verification
   - etc.
4. Click "Find Traces"
5. Click on a trace to view details

**Trace Details:**
- Timeline view of all spans
- Span attributes and tags
- Events within spans
- Error information
- Parent-child relationships

### File Output

Traces are also exported to `otel-traces.json`:

```bash
# View raw traces
cat otel-traces.json | jq '.'

# Filter by service
cat otel-traces.json | jq 'select(.resourceSpans[0].resource.attributes[] | select(.key=="service.name" and .value.stringValue=="open-swarm"))'

# Count traces
cat otel-traces.json | jq -s 'length'
```

### Console Output

The collector logs trace information to stdout:

```bash
# View collector logs
docker logs -f open-swarm-otel-collector
```

## Common Queries

### Find Slow Operations

In Jaeger UI:
1. Service: `open-swarm`
2. Min Duration: `30s` (or any threshold)
3. Click "Find Traces"

### Find Failed Operations

In Jaeger UI:
1. Service: `open-swarm`
2. Tags: `error=true`
3. Click "Find Traces"

### Find Specific Gate Results

In Jaeger UI:
1. Service: `open-swarm`
2. Tags: `tcr.gate_name=verify_green` and `tcr.gate_passed=false`
3. Click "Find Traces"

### Trace a Specific Workflow

In Jaeger UI:
1. Service: `open-swarm`
2. Tags: `workflow.id=simple-bench-basic-1234567890-1`
3. Click "Find Traces"

## Troubleshooting

### No Traces Appearing

1. **Check collector is running:**
   ```bash
   docker ps | grep otel-collector
   curl http://localhost:4318
   ```

2. **Check collector logs:**
   ```bash
   docker logs open-swarm-otel-collector
   ```

3. **Check worker logs:**
   ```bash
   tail -f worker.log | grep -i "telemetry\|tracing"
   ```

4. **Verify network connectivity:**
   ```bash
   curl -X POST http://localhost:4318/v1/traces \
     -H "Content-Type: application/json" \
     -d '{"resourceSpans":[]}'
   ```

### Collector Connection Refused

```bash
# Check if port is in use
netstat -an | grep 4318

# Restart collector
docker-compose -f docker-compose.otel.yml restart otel-collector
```

### Jaeger UI Not Loading

```bash
# Check Jaeger is running
docker ps | grep jaeger
docker logs open-swarm-jaeger

# Restart Jaeger
docker-compose -f docker-compose.otel.yml restart jaeger
```

### High Memory Usage

Edit `otel-collector-config.yaml`:

```yaml
processors:
  memory_limiter:
    check_interval: 1s
    limit_mib: 256  # Reduce from 512
    spike_limit_mib: 64  # Reduce from 128
```

## Performance Impact

The OpenTelemetry instrumentation has minimal performance impact:

- **Overhead**: ~1-2% CPU, <50MB memory
- **Sampling**: Configurable (default 100%)
- **Batching**: Traces are batched before export
- **Async Export**: Non-blocking trace export

For production use, consider:
- Reducing sampling rate to 0.1 (10%)
- Increasing batch sizes
- Using tail-based sampling

## Advanced Usage

### Custom Spans

Add custom spans in your code:

```go
import "open-swarm/internal/telemetry"

func myFunction(ctx context.Context) error {
    ctx, span := telemetry.StartSpan(ctx, "my.component", "MyOperation")
    defer span.End()
    
    // Add attributes
    telemetry.AddAttributes(ctx, 
        attribute.String("custom.key", "value"),
        attribute.Int("count", 42),
    )
    
    // Add event
    telemetry.AddEvent(ctx, "processing.started")
    
    // Record error if needed
    if err != nil {
        telemetry.RecordError(ctx, err)
        return err
    }
    
    return nil
}
```

### Linking Traces

Extract trace context for cross-service calls:

```go
import "go.opentelemetry.io/otel/trace"

func getTraceInfo(ctx context.Context) {
    traceID := telemetry.TraceID(ctx)
    spanID := telemetry.SpanID(ctx)
    
    log.Printf("Trace ID: %s, Span ID: %s", traceID, spanID)
}
```

## Cleanup

Stop and remove all observability containers:

```bash
docker-compose -f docker-compose.otel.yml down -v
```

Remove trace files:

```bash
rm -f otel-traces.json*
rm -rf otel-data/
```

## References

- [OpenTelemetry Go SDK](https://opentelemetry.io/docs/instrumentation/go/)
- [OTLP Protocol](https://opentelemetry.io/docs/specs/otlp/)
- [Jaeger Documentation](https://www.jaegertracing.io/docs/)
- [OpenTelemetry Collector](https://opentelemetry.io/docs/collector/)