#!/usr/bin/env nu
# Open Swarm - Temporal Workflow Demo
# Watch workflows execute in real-time via Temporal UI

def main [] {
    print "🚀 Open Swarm - Temporal Workflow Demo"
    print "======================================"
    print ""

    # Check if Temporal is running
    let temporal_running = (
        try {
            http get http://localhost:8233 | is-not-empty
        } catch {
            false
        }
    )

    if not $temporal_running {
        print "❌ Temporal UI not accessible at http://localhost:8233"
        print ""
        print "Start it with:"
        print "  make docker-up"
        print ""
        exit 1
    }

    print "✅ Temporal UI is running at http://localhost:8233"
    print ""

    # Start the worker in background
    print "🔧 Starting Temporal worker..."
    let worker = (
        run-external --redirect-stdout --redirect-stderr "go" "run" "cmd/temporal-worker/main.go" 
        | complete
        | get pid
    )

    # Wait for worker to start
    sleep 3sec

    # Run the demo
    print ""
    print "▶️  Running workflow demo..."
    
    try {
        run-external "go" "run" "cmd/workflow-demo/main.go"
    } catch {
        print $"⚠️  Demo completed: ($in)"
    }

    # Cleanup
    try {
        ps | where pid == $worker | each { |proc| kill $proc.pid }
    } catch {
        # Worker might already be stopped
    }

    print ""
    print "✅ Demo complete!"
    print "   View workflow history: http://localhost:8233"
}
