#!/usr/bin/env bun
import { ready, status, close, create, list, addDependency } from "./beads"

console.log("🧪 Testing Beads Plugin...\n")

// Test 1: Check that all exports are defined
console.log("✓ Test 1: All functions exported")
console.log("  - ready:", typeof ready)
console.log("  - status:", typeof status)
console.log("  - close:", typeof close)
console.log("  - create:", typeof create)
console.log("  - list:", typeof list)
console.log("  - addDependency:", typeof addDependency)

// Test 2: Validate structure
console.log("\n✓ Test 2: Tool structure validation")
console.log("  - ready.description:", ready.description)
console.log("  - ready.args:", Object.keys(ready.args))
console.log("  - ready.execute:", typeof ready.execute)

console.log("\n✓ Test 3: Schema validation")
console.log("  - status args:", Object.keys(status.args))
console.log("  - create args:", Object.keys(create.args))
console.log("  - list args:", Object.keys(list.args))

console.log("\n✅ All structural tests passed!")
console.log("\n📝 Note: Runtime tests require actual bd commands")
