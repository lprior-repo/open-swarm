// Copyright (c) 2025 Open Swarm Contributors
//
// This software is released under the MIT License.
// See LICENSE file in the repository for details.

package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"go.temporal.io/sdk/client"
	"go.temporal.io/sdk/worker"

	"open-swarm/internal/telemetry"
	"open-swarm/internal/temporal"
	"open-swarm/pkg/dag"
)

const (
	maxConcurrentActivityExecutionSize      = 50 // Match MaxConcurrentAgents
	maxConcurrentWorkflowTaskExecutionSize  = 10
	maxConcurrentLocalActivityExecutionSize = 100
	workerStopTimeout                       = 30 * time.Second
)

func main() {
	log.Println("🚀 Reactor-SDK Temporal Worker v6.1.0")

	// Initialize OpenTelemetry tracing
	log.Println("🔭 Initializing OpenTelemetry tracing...")
	ctx := context.Background()

	// Configure telemetry - use environment variables if available
	config := telemetry.DefaultConfig()
	if collectorURL := os.Getenv("OTEL_COLLECTOR_URL"); collectorURL != "" {
		config.CollectorURL = collectorURL
	}
	if serviceName := os.Getenv("OTEL_SERVICE_NAME"); serviceName != "" {
		config.ServiceName = serviceName
	}

	tracerProvider, err := telemetry.NewTracerProvider(ctx, config)
	if err != nil {
		log.Printf("⚠️  Failed to initialize tracing (continuing without): %v", err)
	} else {
		log.Printf("✅ OpenTelemetry tracing initialized (collector: http://%s)", config.CollectorURL)
		defer func() {
			log.Println("🔭 Shutting down tracing...")
			shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
			defer cancel()
			if err := tracerProvider.Shutdown(shutdownCtx); err != nil {
				log.Printf("⚠️  Error shutting down tracer provider: %v", err)
			} else {
				log.Println("✅ Tracing shutdown complete")
			}
		}()
	}

	// Initialize global infrastructure managers (singleton pattern)
	log.Println("🔧 Initializing global managers...")
	temporal.InitializeGlobals(8000, 9000, ".", "./worktrees")

	// Connect to Temporal server
	c, err := client.Dial(client.Options{
		HostPort: client.DefaultHostPort, // localhost:7233
	})
	if err != nil {
		log.Fatalln("❌ Unable to create Temporal client:", err)
	}

	defer func() {
		if c != nil {
			c.Close()
		}
	}()

	log.Println("✅ Connected to Temporal server")

	// Create worker on task queue
	w := worker.New(c, "reactor-task-queue", worker.Options{
		MaxConcurrentActivityExecutionSize:      maxConcurrentActivityExecutionSize,
		MaxConcurrentWorkflowTaskExecutionSize:  maxConcurrentWorkflowTaskExecutionSize,
		MaxConcurrentLocalActivityExecutionSize: maxConcurrentLocalActivityExecutionSize,
		WorkerStopTimeout:                       workerStopTimeout,
	})

	// Register workflows
	w.RegisterWorkflow(temporal.TCRWorkflow)
	w.RegisterWorkflow(temporal.EnhancedTCRWorkflow)
	w.RegisterWorkflow(temporal.BenchmarkWorkflow)
	w.RegisterWorkflow(dag.TddDagWorkflow)

	// Register activities
	cellActivities := temporal.NewCellActivities()
	enhancedActivities := temporal.NewEnhancedActivities()
	shellActivities := &temporal.ShellActivities{}
	agentActivities := temporal.NewAgentActivities()
	dagActivities := &dag.ShellActivities{}

	w.RegisterActivity(cellActivities.BootstrapCell)
	w.RegisterActivity(cellActivities.ExecuteTask)
	w.RegisterActivity(cellActivities.RunTests)
	w.RegisterActivity(cellActivities.CommitChanges)
	w.RegisterActivity(cellActivities.RevertChanges)
	w.RegisterActivity(cellActivities.TeardownCell)
	w.RegisterActivity(enhancedActivities.AcquireFileLocks)
	w.RegisterActivity(enhancedActivities.ReleaseFileLocks)
	w.RegisterActivity(enhancedActivities.ExecuteGenTest)
	w.RegisterActivity(enhancedActivities.ExecuteLintTest)
	w.RegisterActivity(enhancedActivities.ExecuteVerifyRED)
	w.RegisterActivity(enhancedActivities.ExecuteGenImpl)
	w.RegisterActivity(enhancedActivities.ExecuteVerifyGREEN)
	w.RegisterActivity(enhancedActivities.ExecuteMultiReview)
	w.RegisterActivity(shellActivities.RunScript)
	w.RegisterActivity(shellActivities.RunScriptInDir)
	w.RegisterActivity(agentActivities.InvokeAgent)
	w.RegisterActivity(agentActivities.StreamedInvokeAgent)
	w.RegisterActivity(dagActivities)

	log.Println("📋 Registered workflows and activities")
	log.Println("⚙️  Worker listening on task queue: reactor-task-queue")

	// Start worker
	errChan := make(chan error, 1)
	go func() {
		errChan <- w.Run(worker.InterruptCh())
	}()

	// Wait for interrupt signal
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

	select {
	case err := <-errChan:
		log.Println("❌ Worker error:", err)
		os.Exit(1)
	case <-sigChan:
		log.Println("\n🛑 Shutdown signal received")
	}

	log.Println("✅ Worker stopped")
}
