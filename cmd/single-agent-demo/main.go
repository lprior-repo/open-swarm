// Copyright (c) 2025 Open Swarm Contributors
//
// This software is released under the MIT License.
// See LICENSE file in the repository for details.

package main

import (
	"context"
	"log"
	"os"
	"time"

	"open-swarm/internal/agent"
	"open-swarm/internal/infra"
)

func main() {
	log.Println("🚀 Single OpenCode Agent Demo")
	log.Println("================================")

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	// Get current working directory
	cwd, err := os.Getwd()
	if err != nil {
		log.Fatalf("❌ Failed to get working directory: %v", err)
	}

	// 1. Setup Infrastructure
	log.Println("\n📦 Setting up infrastructure...")
	portMgr := infra.NewPortManager(8000, 8100)
	serverMgr := infra.NewServerManager()

	// 2. Allocate Port
	log.Println("🔌 Allocating port...")
	port, err := portMgr.Allocate()
	if err != nil {
		log.Fatalf("❌ Failed to allocate port: %v", err)
	}
	defer portMgr.Release(port)
	log.Printf("   ✅ Allocated port: %d", port)

	// 3. Boot OpenCode Server
	log.Printf("\n🖥️  Booting OpenCode server on port %d...", port)
	serverHandle, err := serverMgr.BootServer(ctx, cwd, "demo-agent", port)
	if err != nil {
		log.Fatalf("❌ Failed to boot server: %v", err)
	}
	defer serverMgr.Shutdown(serverHandle)
	log.Printf("   ✅ Server running at %s (PID: %d)", serverHandle.BaseURL, serverHandle.PID)

	// 4. Create SDK Client
	log.Println("\n🔗 Creating SDK client...")
	client := agent.NewClient(serverHandle.BaseURL, port)
	log.Printf("   ✅ Client connected to %s", client.GetBaseURL())

	// 5. Execute a simple task
	log.Println("\n🎯 Executing task...")
	prompt := "Create a simple hello.txt file with the message 'Hello from OpenCode agent!'"

	result, err := client.ExecutePrompt(ctx, prompt, &agent.PromptOptions{
		Title: "Demo Task",
		// Don't specify model - use default configured in OpenCode
		NoReply: false,
	})
	if err != nil {
		log.Fatalf("❌ Task execution failed: %v", err)
	}

	log.Println("\n📊 Task Results:")
	log.Printf("   Session ID: %s", result.SessionID)
	log.Printf("   Message ID: %s", result.MessageID)

	// Print all result parts
	for i, part := range result.Parts {
		log.Printf("   Part %d [%s]: %s", i+1, part.Type, part.Text)
		if part.ToolName != "" {
			log.Printf("     Tool: %s", part.ToolName)
		}
	}

	// 6. Verify file was created
	log.Println("\n✅ Verifying results...")
	if _, err := os.Stat("hello.txt"); err == nil {
		content, _ := os.ReadFile("hello.txt")
		log.Printf("   ✅ File created: hello.txt")
		log.Printf("   Content: %s", string(content))
		// Cleanup
		os.Remove("hello.txt")
	} else {
		log.Println("   ⚠️  File not found (agent may have used different approach)")
	}

	log.Println("\n✨ Demo completed successfully!")
	log.Println("   Server will shutdown automatically...")
}
