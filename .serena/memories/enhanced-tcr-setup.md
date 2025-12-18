# Enhanced TCR Setup Guide

## Overview
Open-swarm implements Enhanced TCR (Test-Commit-Revert) workflows using Temporal.io.

## Architecture
- **Temporal Workflows**: `internal/temporal/`
  - `workflow_tcr_enhanced.go` - 6-gate Enhanced TCR
  - `activities_enhanced.go` - Gate implementations
  
- **Agent Configs**: `opencode.json` agents using Haiku 4.5:
  - `test-generator` - Gate 1 test generation
  - `implementation` - Gate 4 implementation
  - `reviewer-testing`, `reviewer-functional`, `reviewer-architecture` - Gate 6 reviews

## 6-Gate System
1. Generate Tests (TDD RED)
2. Lint Tests
3. Verify RED (tests fail)
4. Generate Implementation
5. Verify GREEN (tests pass)
6. Multi-reviewer approval

## Running Enhanced TCR

### Prerequisites
- Docker running
- Temporal started via `docker compose up -d`

### Steps
1. Build worker: `go build -o /tmp/temporal-worker ./cmd/temporal-worker`
2. Run worker: `/tmp/temporal-worker`
3. Run benchmark: `go run ./cmd/benchmark-tcr -strategy enhanced -runs 3 -prompt "your task"`

## Key Files
- `docker-compose.yml` - Temporal services (PostgreSQL port: 5435)
- `cmd/temporal-worker/main.go` - Worker registration
- `cmd/benchmark-tcr/main.go` - Benchmark runner

## Helper Functions
- `getReviewerAgent(reviewType)` - Maps review type to agent name
- `getReviewFocus(reviewType)` - Gets review focus description
