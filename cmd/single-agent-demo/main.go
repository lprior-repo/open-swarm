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

	// Get current working directory
	cwd, err := os.Getwd()
	if err != nil {
		log.Fatalf("❌ Failed to get working directory: %v", err)
	}
	log.Printf("Working directory: %s\n", cwd)

	ctx, cancel := context.WithTimeout(context.Background(), defaultTimeoutMinutes*time.Minute)
	defer cancel()

	// 1. Setup Infrastructure
	log.Println("\n📦 Step 1: Setting up infrastructure...")
	portMgr := infra.NewPortManager(portRangeStart, portRangeEnd)
	serverMgr := infra.NewServerManager()
	log.Printf("   ✅ Port range: %d-%d (%d ports available)", portRangeStart, portRangeEnd, portMgr.AvailableCount())
	log.Printf("   ✅ Server manager ready (health timeout: 10s)")

	// 2. Allocate Port
	log.Println("\n🔌 Step 2: Allocating port...")
	port, err := portMgr.Allocate()
	if err != nil {
		log.Fatalf("❌ Failed to allocate port: %v", err)
	}
	defer func() {
		if err := portMgr.Release(port); err != nil {
			log.Printf("⚠️  Warning: Failed to release port %d: %v", port, err)
		}
	}()
	log.Printf("   ✅ Allocated port: %d", port)
	log.Printf("   📊 Ports in use: %d, Available: %d", portMgr.AllocatedCount(), portMgr.AvailableCount())

	// 3. Boot OpenCode Server
	log.Printf("\n🖥️  Step 3: Booting OpenCode server on port %d...", port)
	log.Println("   (This will wait for server health check...)")
	bootStart := time.Now()
	serverHandle, err := serverMgr.BootServer(ctx, cwd, "demo-agent", port)
	if err != nil {
		log.Fatalf("❌ Failed to boot server: %v", err)
	}
	defer func() {
		log.Println("\n🛑 Shutting down server...")
		if err := serverMgr.Shutdown(serverHandle); err != nil {
			log.Printf("⚠️  Warning: Server shutdown error: %v", err)
		} else {
			log.Println("   ✅ Server shutdown complete")
		}
	}()
	bootDuration := time.Since(bootStart)
	log.Printf("   ✅ Server running at %s (PID: %d)", serverHandle.BaseURL, serverHandle.PID)
	log.Printf("   ⏱️  Boot time: %v", bootDuration)

	// Verify server is healthy
	if !serverMgr.IsHealthy(ctx, serverHandle) {
		log.Fatal("❌ Server health check failed after boot")
	}
	log.Println("   ✅ Health check passed")

	// 4. Create SDK Client
	log.Println("\n🔗 Step 4: Creating SDK client...")
	client := agent.NewClient(serverHandle.BaseURL, port)
	log.Printf("   ✅ Client connected to %s", client.GetBaseURL())

	// 5. Execute a simple task
	log.Println("\n🎯 Step 5: Executing task...")
	prompt := "Create a simple hello.txt file with the message 'Hello from OpenCode agent!'"
	log.Printf("   Prompt: %s", prompt)

	taskStart := time.Now()
	result, err := client.ExecutePrompt(ctx, prompt, &agent.PromptOptions{
		Title: "Demo Task",
		// Don't specify model - use default configured in OpenCode
		NoReply: false,
	})
	if err != nil {
		log.Fatalf("❌ Task execution failed: %v", err)
	}
	taskDuration := time.Since(taskStart)

	log.Println("\n📊 Task Results:")
	log.Printf("   Session ID: %s", result.SessionID)
	log.Printf("   Message ID: %s", result.MessageID)
	log.Printf("   Duration: %v", taskDuration)
	log.Printf("   Response parts: %d", len(result.Parts))

	// Print all result parts
	for i, part := range result.Parts {
		if part.Type == "text" && part.Text != "" {
			log.Printf("   Part %d [%s]: %.100s...", i+1, part.Type, part.Text)
		}
		if part.ToolName != "" {
			log.Printf("   Part %d [tool]: %s", i+1, part.ToolName)
		}
	}

	// 6. Verify file was created
	log.Println("\n✅ Step 6: Verifying results...")

	// Give agent a moment to complete
	time.Sleep(1 * time.Second)

	// Use GetFileStatus() proactively to see what files were created
	files, err := client.GetFileStatus(ctx)
	if err != nil {
		log.Fatalf("❌ Failed to get file status: %v", err)
	}

	if len(files) == 0 {
		log.Fatal("❌ No files created by agent - task failed")
	}

	log.Printf("   📁 Files created by agent: %d", len(files))
	for _, f := range files {
		log.Printf("      - %s", f.Path)
	}

	// Read the first file created
	content, err := client.ReadFile(ctx, files[0].Path)
	if err != nil {
		log.Fatalf("❌ Failed to read %s: %v", files[0].Path, err)
	}

	// Validate content contains expected text
	if !strings.Contains(content, "Hello") {
		log.Fatalf("❌ File content doesn't match expected. Got: %s", content)
	}

	log.Printf("   ✅ File verified: %s", files[0].Path)
	log.Printf("   📄 Content: %s", content)

	// Final health check
	log.Println("\n🏥 Final health check...")
	if serverMgr.IsHealthy(ctx, serverHandle) {
		log.Println("   ✅ Server still healthy")
	} else {
		log.Println("   ⚠️  Server health check failed")
	}

	log.Println("\n✨ Demo completed successfully!")
	log.Printf("Total execution time: %v", time.Since(bootStart))
	log.Println("   Server will shutdown automatically...")
}
