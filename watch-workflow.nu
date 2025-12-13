#!/usr/bin/env nu
# Open Swarm - AI Agent Automation Demo
# Complete demo with 5-minute timeout showing entire system

def main [
    --timeout: int = 300  # 5 minute timeout (in seconds)
] {
    print "🤖 Open Swarm - AI Agent Automation Demo"
    print "=========================================="
    print ""
    print $"⏱️  Timeout: ($timeout) seconds"
    print ""

    # Check if Temporal is running
    let temporal_running = (
        try {
            http get http://localhost:8081 | is-not-empty
        } catch {
            false
        }
    )

    if not $temporal_running {
        print "❌ Temporal UI not accessible at http://localhost:8081"
        print ""
        print "Starting Temporal with Docker Compose..."
        print ""
        
        try {
            run-external "docker" "compose" "up" "-d"
            print "⏳ Waiting for Temporal to be ready (30 seconds)..."
            sleep 30sec
        } catch {
            print "❌ Failed to start Docker Compose"
            print "   Run manually: make docker-up"
            exit 1
        }
    }

    print "✅ Temporal UI is running at http://localhost:8081"
    print ""

    # Build binaries first for faster execution
    print "🔨 Building binaries..."
    print ""
    
    try {
        run-external "go" "build" "-o" "bin/temporal-worker" "./cmd/temporal-worker"
        run-external "go" "build" "-o" "bin/agent-automation-demo" "./cmd/agent-automation-demo"
        print "✅ Binaries built successfully"
    } catch {
        print "❌ Failed to build binaries"
        exit 1
    }
    
    print ""
    
    # Start the worker in background using bash
    print "🔧 Starting Temporal worker in background..."
    print ""
    
    # Start worker as background process via bash
    run-external "bash" "-c" "./bin/temporal-worker > /tmp/worker.log 2>&1 & echo $! > /tmp/worker.pid"
    
    let worker_pid = (open /tmp/worker.pid | str trim | into int)
    print $"✅ Worker started \(PID: ($worker_pid)\)"
    
    # Wait for worker to initialize
    sleep 3sec

    print ""
    print "🚀 Starting AI Agent Automation Demo..."
    print ""
    print "This will demonstrate:"
    print "  • Multi-agent parallel execution"
    print "  • DAG-based dependency management"
    print "  • TDD workflow (Test-Commit-Revert)"
    print "  • Real-time visualization in Temporal UI"
    print ""
    print "👀 Open http://localhost:8081 to watch workflows execute!"
    print ""
    print "═══════════════════════════════════════"
    print ""

    # Run the comprehensive demo
    let demo_result = (
        do -i {
            run-external "timeout" $"($timeout)s" "./bin/agent-automation-demo"
        } | complete
    )

    print ""
    print "═══════════════════════════════════════"
    
    if $demo_result.exit_code == 0 {
        print "✅ Demo completed successfully!"
    } else if $demo_result.exit_code == 124 {
        print $"⏱️  Demo timed out after ($timeout) seconds"
        print "   (This is normal if workflows are still running)"
    } else {
        print $"⚠️  Demo exited with code: ($demo_result.exit_code)"
    }

    # Stop the worker
    print ""
    print "🛑 Stopping worker..."
    
    try {
        run-external "kill" ($worker_pid | into string)
        print $"✅ Worker stopped \(PID: ($worker_pid)\)"
    } catch {
        print "⚠️  Worker may have already stopped"
    }
    
    # Clean up pid file
    try {
        rm /tmp/worker.pid
    }

    print ""
    print "📊 View all workflows: http://localhost:8081"
    print ""
    print "🔍 Workflow features you can explore:"
    print "   • Timeline view (see activity execution order)"
    print "   • Parallel execution (multiple agents working simultaneously)"
    print "   • Dependency resolution (tasks waiting for prerequisites)"
    print "   • Signal handling (TDD loop completions)"
    print "   • Event history (complete audit trail)"
    print ""
    print "📋 Worker logs available at: /tmp/worker.log"
    print ""
}
