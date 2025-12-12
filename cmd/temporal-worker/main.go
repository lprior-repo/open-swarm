package main

import (
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"go.temporal.io/sdk/client"
	"go.temporal.io/sdk/worker"

	"open-swarm/internal/temporal"
)

func main() {
	log.Println("🚀 Reactor-SDK Temporal Worker v6.1.0")

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
	defer c.Close()

	log.Println("✅ Connected to Temporal server")

	// Create worker on task queue
	w := worker.New(c, "reactor-task-queue", worker.Options{
		MaxConcurrentActivityExecutionSize:      50, // Match MaxConcurrentAgents
		MaxConcurrentWorkflowTaskExecutionSize:  10,
		MaxConcurrentLocalActivityExecutionSize: 100,
		WorkerStopTimeout:                       30 * time.Second,
	})

	// Register workflows
	w.RegisterWorkflow(temporal.TCRWorkflow)
	w.RegisterWorkflow(temporal.TddDagWorkflow)

	// Register activities
	cellActivities := temporal.NewCellActivities()
	shellActivities := &temporal.ShellActivities{}

	w.RegisterActivity(cellActivities.BootstrapCell)
	w.RegisterActivity(cellActivities.ExecuteTask)
	w.RegisterActivity(cellActivities.RunTests)
	w.RegisterActivity(cellActivities.CommitChanges)
	w.RegisterActivity(cellActivities.RevertChanges)
	w.RegisterActivity(cellActivities.TeardownCell)
	w.RegisterActivity(shellActivities.RunScript)
	w.RegisterActivity(shellActivities.RunScriptInDir)

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
		log.Fatalln("❌ Worker error:", err)
	case <-sigChan:
		log.Println("\n🛑 Shutdown signal received")
	}

	log.Println("✅ Worker stopped")
}
