/**
 * Tests for TDD Enforcer Plugin Self-Protection
 *
 * Following TDD: Write ONE test at a time, see it fail, make it pass
 */

import { describe, test, expect } from "bun:test"

describe("TDD Enforcer Self-Protection", () => {
  test("should detect opencode.json as protected", () => {
    const filePath = "/home/user/project/opencode.json"
    expect(isProtectedPath(filePath)).toBe(true)
  })
})

// Helper function to be implemented
function isProtectedPath(filePath: string): boolean {
  throw new Error("Not implemented - RED phase")
}
