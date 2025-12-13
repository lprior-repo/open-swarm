// Copyright (c) 2025 Open Swarm Contributors
//
// This software is released under the MIT License.
// See LICENSE file in the repository for details.

package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"

	"open-swarm/internal/agent"
	"open-swarm/internal/infra"
	"open-swarm/internal/workflow"
)

const (
	// MaxConcurrentAgents is the maximum number of concurrent agents that can run.
	MaxConcurrentAgents = 50
	// PortRangeMin is the minimum port in the agent port range.
	PortRangeMin = 8000
	// PortRangeMax is the maximum port in the agent port range.
	PortRangeMax = 9000
)

// Config holds the configuration for the reactor.
type Config struct {
	RepoDir      string
	WorktreeBase string
	Branch       string
	MaxAgents    int
}

func main() {
	repoDir, worktreeBase, branch, maxAgents, taskID, taskDesc, taskPrompt := parseFlags()

	config := &Config{
		RepoDir:      repoDir,
		WorktreeBase: worktreeBase,
		Branch:       branch,
		MaxAgents:    maxAgents,
	}

	logConfiguration(config)

	if taskID == "" || taskPrompt == "" {
		log.Fatal("❌ Error: --task and --prompt are required")
	}

	portManager, serverManager, worktreeManager := initializeInfrastructure(config)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	setupShutdownHandler(ctx, cancel)

	activities := workflow.NewActivities(portManager, serverManager, worktreeManager)
	cell, err := bootstrapCell(ctx, activities, config.Branch)
	if err != nil {
		log.Fatalf("❌ Failed to bootstrap cell: %v", err)
	}

	defer teardownCell(ctx, activities, cell)

	task := &agent.TaskContext{
		TaskID:      taskID,
		Description: taskDesc,
		Prompt:      taskPrompt,
	}

	executeAndHandleTask(ctx, activities, cell, task)

	log.Println("\n✅ Reactor execution complete")
}

func parseFlags() (string, string, string, int, string, string, string) {
	var (
		repoDir      = flag.String("repo", ".", "Git repository directory")
		worktreeBase = flag.String("worktrees", "./worktrees", "Base directory for worktrees")
		branch       = flag.String("branch", "main", "Branch to use for worktrees")
		maxAgents    = flag.Int("max-agents", MaxConcurrentAgents, "Maximum concurrent agents")
		taskID       = flag.String("task", "", "Task ID to execute")
		taskDesc     = flag.String("desc", "", "Task description")
		taskPrompt   = flag.String("prompt", "", "Task prompt")
		_ = flag.Bool("parallel", false, "Run tasks in parallel mode (not yet implemented)")
	)
	flag.Parse()
	return *repoDir, *worktreeBase, *branch, *maxAgents, *taskID, *taskDesc, *taskPrompt
}

func logConfiguration(config *Config) {
	log.Printf("🚀 Reactor-SDK v6.0.0 - Enterprise Agent Orchestrator")
	log.Printf("📊 Configuration:")
	log.Printf("   Repository: %s", config.RepoDir)
	log.Printf("   Worktree Base: %s", config.WorktreeBase)
	log.Printf("   Branch: %s", config.Branch)
	log.Printf("   Max Agents: %d", config.MaxAgents)
	log.Printf("   Port Range: %d-%d", PortRangeMin, PortRangeMax)
}

func initializeInfrastructure(config *Config) (*infra.PortManager, *infra.ServerManager, *infra.WorktreeManager) {
	log.Println("🔧 Initializing infrastructure...")
	portManager := infra.NewPortManager(PortRangeMin, PortRangeMax)
	serverManager := infra.NewServerManager()
	worktreeManager := infra.NewWorktreeManager(config.RepoDir, config.WorktreeBase)

	log.Println("🧹 Cleaning up existing worktrees...")
	if err := worktreeManager.CleanupAll(); err != nil {
		log.Printf("⚠️  Warning: Failed to cleanup worktrees: %v", err)
	}

	return portManager, serverManager, worktreeManager
}

func setupShutdownHandler(ctx context.Context, cancel context.CancelFunc) {
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

	go func() {
		<-sigChan
		log.Println("\n🛑 Shutdown signal received, cleaning up...")
		cancel()
	}()
}

func bootstrapCell(ctx context.Context, activities *workflow.Activities, branch string) (*workflow.CellBootstrap, error) {
	log.Println("⚙️  Initializing workflow engine...")
	log.Println("🎯 Starting task execution...")
	log.Println("📦 Bootstrapping agent cell...")
	cell, err := activities.BootstrapCell(ctx, "primary", branch)
	if err != nil {
		return nil, err
	}
	log.Printf("✅ Cell bootstrapped on port %d", cell.Port)
	log.Printf("📁 Worktree: %s", cell.WorktreePath)
	return cell, nil
}

func teardownCell(ctx context.Context, activities *workflow.Activities, cell *workflow.CellBootstrap) {
	if cell != nil {
		log.Println("🧹 Tearing down cell...")
		if err := activities.TeardownCell(ctx, cell); err != nil {
			log.Printf("⚠️  Warning: Failed to teardown cell: %v", err)
		}
	}
}

func executeAndHandleTask(ctx context.Context, activities *workflow.Activities, cell *workflow.CellBootstrap, task *agent.TaskContext) {
	log.Println("⚙️  Executing task...")
	result, err := activities.ExecuteTask(ctx, cell, task)
	if err != nil {
		log.Fatalf("❌ Task execution failed: %v", err)
	}

	if !result.Success {
		log.Printf("❌ Task failed: %s", result.ErrorMessage)
		os.Exit(1)
	}

	log.Println("✅ Task completed successfully")
	log.Printf("📝 Output:\n%s", result.Output)

	logModifiedFiles(result.FilesModified)
	handleTestsAndFinal(ctx, activities, cell, task)
}

func logModifiedFiles(files []string) {
	if len(files) > 0 {
		log.Println("📁 Modified files:")
		for _, file := range files {
			log.Printf("   - %s", file)
		}
	}
}

func handleTestsAndFinal(ctx context.Context, activities *workflow.Activities, cell *workflow.CellBootstrap, task *agent.TaskContext) {
	log.Println("🧪 Running tests...")
	testsPassed, err := activities.RunTests(ctx, cell)
	if err != nil {
		log.Printf("⚠️  Test execution failed: %v", err)
		testsPassed = false
	}

	if testsPassed {
		handleTestPassed(ctx, activities, cell, task)
	} else {
		handleTestFailed(ctx, activities, cell)
	}
}

func handleTestPassed(ctx context.Context, activities *workflow.Activities, cell *workflow.CellBootstrap, task *agent.TaskContext) {
	log.Println("✅ Tests passed")
	log.Println("💾 Committing changes...")
	commitMsg := fmt.Sprintf("Task %s: %s\n\n🤖 Generated by Reactor-SDK", task.TaskID, task.Description)
	if err := activities.CommitChanges(ctx, cell, commitMsg); err != nil {
		log.Printf("⚠️  Commit failed: %v", err)
	} else {
		log.Println("✅ Changes committed")
	}
}

func handleTestFailed(ctx context.Context, activities *workflow.Activities, cell *workflow.CellBootstrap) {
	log.Println("❌ Tests failed")
	log.Println("↩️  Reverting changes...")
	if err := activities.RevertChanges(ctx, cell); err != nil {
		log.Printf("⚠️  Revert failed: %v", err)
	} else {
		log.Println("✅ Changes reverted")
	}
}
