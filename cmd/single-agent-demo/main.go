// Copyright (c) 2025 Open Swarm Contributors
//
// This software is released under the MIT License.
// See LICENSE file in the repository for details.

// Package main demonstrates a single agent orchestration example.
package main

import (
	"context"
	"log"
	"os"
	"strings"
	"time"

	"open-swarm/internal/agent"
	"open-swarm/internal/infra"
)

const (
	defaultTimeoutMinutes = 5
	portRangeStart        = 8000
	portRangeEnd          = 8100
)

func main() {
	logDemoIntro()

	cwd := getWorkingDir()
	ctx, cancel := context.WithTimeout(context.Background(), defaultTimeoutMinutes*time.Minute)
	defer cancel()

	portMgr, serverMgr := setupInfra()
	port := allocatePort(portMgr)
	defer releasePort(portMgr, port)

	bootStart := time.Now()
	serverHandle := bootServer(ctx, cwd, portMgr, serverMgr, port)
	defer shutdownServerDefer(serverMgr, serverHandle)

	client := createClient(serverHandle, port)
	result := executeTask(ctx, client)
	verifyResults(ctx, client)
	performFinalHealthCheck(ctx, serverMgr, serverHandle)

	log.Println("\n✨ Demo completed successfully!")
	log.Printf("Total execution time: %v", time.Since(bootStart))
	log.Println("   Server will shutdown automatically...")
}

func logDemoIntro() {
	log.Println("🚀 Single OpenCode Agent Demo")
	log.Println("================================")
	log.Println("This demo shows the complete flow:")
	log.Println("  1. Setup infrastructure (port manager, server manager)")
	log.Println("  2. Allocate unique port for agent")
	log.Println("  3. Boot OpenCode server with health check")
	log.Println("  4. Create SDK client")
	log.Println("  5. Execute task via agent")
	log.Println("  6. Verify results")
	log.Println()
}

func getWorkingDir() string {
	cwd, err := os.Getwd()
	if err != nil {
		log.Fatalf("Failed to get working directory: %v", err)
	}
	log.Printf("Working directory: %s\n", cwd)
	return cwd
}

func setupInfra() (*infra.PortManager, *infra.ServerManager) {
	log.Println("\n📦 Step 1: Setting up infrastructure...")
	portMgr := infra.NewPortManager(portRangeStart, portRangeEnd)
	serverMgr := infra.NewServerManager()
	log.Printf("   ✅ Port range: %d-%d (%d ports available)", portRangeStart, portRangeEnd, portMgr.AvailableCount())
	log.Printf("   ✅ Server manager ready (health timeout: 10s)")
	return portMgr, serverMgr
}

func allocatePort(portMgr *infra.PortManager) int {
	log.Println("\n🔌 Step 2: Allocating port...")
	port, err := portMgr.Allocate()
	if err != nil {
		log.Fatalf("Failed to allocate port: %v", err)
	}
	log.Printf("   ✅ Allocated port: %d", port)
	log.Printf("   📊 Ports in use: %d, Available: %d", portMgr.AllocatedCount(), portMgr.AvailableCount())
	return port
}

func releasePort(portMgr *infra.PortManager, port int) {
	if err := portMgr.Release(port); err != nil {
		log.Printf("⚠️  Warning: Failed to release port %d: %v", port, err)
	}
}

func bootServer(ctx context.Context, cwd string, portMgr *infra.PortManager, serverMgr *infra.ServerManager, port int) *infra.ServerHandle {
	log.Printf("\n🖥️  Step 3: Booting OpenCode server on port %d...", port)
	log.Println("   (This will wait for server health check...)")
	bootStart := time.Now()
	serverHandle, err := serverMgr.BootServer(ctx, cwd, "demo-agent", port)
	if err != nil {
		log.Fatalf("Failed to boot server: %v", err)
	}
	bootDuration := time.Since(bootStart)
	log.Printf("   ✅ Server running at %s (PID: %d)", serverHandle.BaseURL, serverHandle.PID)
	log.Printf("   ⏱️  Boot time: %v", bootDuration)

	if !serverMgr.IsHealthy(ctx, serverHandle) {
		log.Fatal("Server health check failed after boot")
	}
	log.Println("   ✅ Health check passed")
	return serverHandle
}

func shutdownServerDefer(serverMgr *infra.ServerManager, serverHandle *infra.ServerHandle) {
	log.Println("\n🛑 Shutting down server...")
	if err := serverMgr.Shutdown(serverHandle); err != nil {
		log.Printf("⚠️  Warning: Server shutdown error: %v", err)
	} else {
		log.Println("   ✅ Server shutdown complete")
	}
}

func createClient(serverHandle *infra.ServerHandle, port int) *agent.Client {
	log.Println("\n🔗 Step 4: Creating SDK client...")
	client := agent.NewClient(serverHandle.BaseURL, port)
	log.Printf("   ✅ Client connected to %s", client.GetBaseURL())
	return client
}

func executeTask(ctx context.Context, client *agent.Client) *agent.PromptResult {
	log.Println("\n🎯 Step 5: Executing task...")
	prompt := "Create a simple hello.txt file with the message 'Hello from OpenCode agent!'"
	log.Printf("   Prompt: %s", prompt)

	taskStart := time.Now()
	result, err := client.ExecutePrompt(ctx, prompt, &agent.PromptOptions{
		Title:   "Demo Task",
		NoReply: false,
	})
	if err != nil {
		log.Fatalf("Task execution failed: %v", err)
	}
	taskDuration := time.Since(taskStart)

	logTaskResults(result, taskDuration)
	return result
}

func logTaskResults(result *agent.PromptResult, duration time.Duration) {
	log.Println("\n📊 Task Results:")
	log.Printf("   Session ID: %s", result.SessionID)
	log.Printf("   Message ID: %s", result.MessageID)
	log.Printf("   Duration: %v", duration)
	log.Printf("   Response parts: %d", len(result.Parts))

	for i, part := range result.Parts {
		if part.Type == "text" && part.Text != "" {
			log.Printf("   Part %d [%s]: %.100s...", i+1, part.Type, part.Text)
		}
		if part.ToolName != "" {
			log.Printf("   Part %d [tool]: %s", i+1, part.ToolName)
		}
	}
}

func verifyResults(ctx context.Context, client *agent.Client) {
	log.Println("\n✅ Step 6: Verifying results...")
	time.Sleep(1 * time.Second)

	files, err := client.GetFileStatus(ctx)
	if err != nil {
		log.Fatalf("Failed to get file status: %v", err)
	}

	if len(files) == 0 {
		log.Fatal("No files created by agent - task failed")
	}

	log.Printf("   📁 Files created by agent: %d", len(files))
	for _, f := range files {
		log.Printf("      - %s", f.Path)
	}

	verifyFileContent(ctx, client, files)
}

func verifyFileContent(ctx context.Context, client *agent.Client, files []interface{}) {
	if len(files) == 0 {
		return
	}

	// Extract path from first file
	var filePath string
	if f, ok := files[0].(map[string]interface{}); ok {
		if p, ok := f["Path"].(string); ok {
			filePath = p
		}
	}

	if filePath == "" {
		return
	}

	content, err := client.ReadFile(ctx, filePath)
	if err != nil {
		log.Fatalf("Failed to read %s: %v", filePath, err)
	}

	if !strings.Contains(content, "Hello") {
		log.Fatalf("File content doesn't match expected. Got: %s", content)
	}

	log.Printf("   ✅ File verified: %s", filePath)
	log.Printf("   📄 Content: %s", content)
}

func performFinalHealthCheck(ctx context.Context, serverMgr *infra.ServerManager, serverHandle *infra.ServerHandle) {
	log.Println("\n🏥 Final health check...")
	if serverMgr.IsHealthy(ctx, serverHandle) {
		log.Println("   ✅ Server still healthy")
	} else {
		log.Println("   ⚠️  Server health check failed")
	}
}
