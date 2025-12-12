import { tool } from "@opencode-ai/plugin"
import { $ } from "bun"
import { existsSync } from "fs"
import { join, dirname, basename } from "path"

/**
 * TDD Guard - Enforces Test-Driven Development workflow
 * 
 * Validates:
 * 1. Test file exists before implementation
 * 2. Test fails initially (Red)
 * 3. Implementation makes test pass (Green)
 * 4. Test is atomic, small, and deterministic
 * 5. Uses testify for assertions
 * 6. Full test suite passes
 */

interface TDDValidationResult {
  success: boolean
  phase: "red" | "green" | "refactor" | "error"
  message: string
  details?: {
    testFile?: string
    implementationFile?: string
    testExists?: boolean
    testFails?: boolean
    testPasses?: boolean
    isAtomic?: boolean
    usesTestify?: boolean
    suitePasses?: boolean
  }
}

export const validateTDD = tool({
  description: "Validate TDD workflow for a Go file change. Ensures test exists first, fails (Red), then passes (Green) with minimal implementation.",
  args: {
    filePath: tool.schema.string().describe("Path to the Go implementation file (not the test file)"),
    skipRedCheck: tool.schema.boolean().optional().default(false).describe("Skip the Red phase check (test must fail first)")
  },
  async execute(args): Promise<TDDValidationResult> {
    try {
      const implFile = args.filePath
      
      // Validate file is a Go file (not a test)
      if (!implFile.endsWith(".go")) {
        return {
          success: false,
          phase: "error",
          message: "File must be a .go file"
        }
      }

      if (implFile.endsWith("_test.go")) {
        return {
          success: false,
          phase: "error",
          message: "Cannot validate test files directly. Provide the implementation file path."
        }
      }

      // Determine test file path
      const testFile = implFile.replace(/\.go$/, "_test.go")
      const dir = dirname(implFile)
      const implBasename = basename(implFile, ".go")

      // PHASE 1: Check test file exists
      const testExists = existsSync(testFile)
      if (!testExists) {
        return {
          success: false,
          phase: "error",
          message: `❌ TDD VIOLATION: Test file must exist BEFORE implementation.\n\nExpected: ${testFile}\n\nCreate the test first following TDD workflow:\n1. Write failing test\n2. Implement minimal code to pass\n3. Refactor if needed`,
          details: {
            implementationFile: implFile,
            testFile,
            testExists: false
          }
        }
      }

      // PHASE 2: Check test uses testify
      const testContent = await Bun.file(testFile).text()
      const usesTestify = testContent.includes("github.com/stretchr/testify")
      
      if (!usesTestify) {
        return {
          success: false,
          phase: "error",
          message: `⚠️  Test should use testify for assertions.\n\nAdd: import "github.com/stretchr/testify/assert"`,
          details: {
            implementationFile: implFile,
            testFile,
            testExists: true,
            usesTestify: false
          }
        }
      }

      // PHASE 3: Check test is atomic (single test function, focused)
      const testFunctions = testContent.match(/func Test\w+\(t \*testing\.T\)/g) || []
      const isAtomic = testFunctions.length <= 3 // Allow small number of focused tests
      
      if (!isAtomic) {
        return {
          success: false,
          phase: "error",
          message: `⚠️  Test file has ${testFunctions.length} test functions. Keep tests atomic and focused.\n\nBreak large test files into smaller, focused test files.`,
          details: {
            implementationFile: implFile,
            testFile,
            testExists: true,
            usesTestify,
            isAtomic: false
          }
        }
      }

      // PHASE 4: RED - Test must fail initially (unless skipped)
      if (!args.skipRedCheck) {
        const redResult = await $`go test ${testFile} -v`.nothrow().quiet()
        const testFails = redResult.exitCode !== 0

        if (!testFails) {
          return {
            success: false,
            phase: "error",
            message: `❌ TDD VIOLATION: Test passes without implementation (RED phase failed).\n\nThe test must fail first to validate it's testing the right thing.\n\nEither:\n1. Your test isn't testing correctly\n2. Implementation already exists\n3. Test is not properly isolated`,
            details: {
              implementationFile: implFile,
              testFile,
              testExists: true,
              usesTestify,
              isAtomic,
              testFails: false
            }
          }
        }
      }

      // PHASE 5: GREEN - Test must pass with implementation
      const greenResult = await $`go test ${testFile} -v`.nothrow().quiet()
      const testPasses = greenResult.exitCode === 0

      if (!testPasses) {
        const output = greenResult.stderr.toString() || greenResult.stdout.toString()
        return {
          success: false,
          phase: "red",
          message: `🔴 RED: Test still failing.\n\nImplement minimal code to make test pass.\n\nTest output:\n${output.slice(0, 500)}`,
          details: {
            implementationFile: implFile,
            testFile,
            testExists: true,
            usesTestify,
            isAtomic,
            testFails: true,
            testPasses: false
          }
        }
      }

      // PHASE 6: Full test suite must pass
      const pkgDir = dirname(implFile)
      const suiteResult = await $`go test ${pkgDir}/... -v`.nothrow().quiet()
      const suitePasses = suiteResult.exitCode === 0

      if (!suitePasses) {
        const output = suiteResult.stderr.toString() || suiteResult.stdout.toString()
        return {
          success: false,
          phase: "green",
          message: `⚠️  Individual test passes but test suite has failures.\n\nFix failing tests in the package:\n${output.slice(0, 500)}`,
          details: {
            implementationFile: implFile,
            testFile,
            testExists: true,
            usesTestify,
            isAtomic,
            testFails: args.skipRedCheck ? undefined : true,
            testPasses: true,
            suitePasses: false
          }
        }
      }

      // SUCCESS: All TDD phases validated
      return {
        success: true,
        phase: "green",
        message: `✅ TDD WORKFLOW VALIDATED\n\n✓ Test file exists: ${testFile}\n✓ Uses testify assertions\n✓ Test is atomic and focused\n${args.skipRedCheck ? '' : '✓ Test failed initially (RED)\n'}✓ Test passes with implementation (GREEN)\n✓ Full test suite passes\n\nReady to commit!`,
        details: {
          implementationFile: implFile,
          testFile,
          testExists: true,
          usesTestify: true,
          isAtomic: true,
          testFails: args.skipRedCheck ? undefined : true,
          testPasses: true,
          suitePasses: true
        }
      }

    } catch (error) {
      return {
        success: false,
        phase: "error",
        message: `Error during TDD validation: ${error.message}`
      }
    }
  }
})

export const checkTestCoverage = tool({
  description: "Check test coverage for a Go package and ensure it meets minimum threshold",
  args: {
    packagePath: tool.schema.string().describe("Package path (e.g., ./internal/api)"),
    minCoverage: tool.schema.number().optional().default(80).describe("Minimum coverage percentage (default: 80%)")
  },
  async execute(args) {
    try {
      const result = await $`go test ${args.packagePath} -cover -coverprofile=/tmp/coverage.out`.nothrow()
      
      if (result.exitCode !== 0) {
        return `❌ Tests failed. Fix failing tests before checking coverage.`
      }

      const output = result.stdout.toString()
      const coverageMatch = output.match(/coverage: ([\d.]+)% of statements/)
      
      if (!coverageMatch) {
        return `⚠️  Could not parse coverage output`
      }

      const coverage = parseFloat(coverageMatch[1])
      
      if (coverage < args.minCoverage) {
        return `❌ Coverage too low: ${coverage}% (minimum: ${args.minCoverage}%)\n\nAdd more tests to increase coverage.`
      }

      return `✅ Coverage: ${coverage}% (exceeds minimum: ${args.minCoverage}%)`
    } catch (error) {
      return `Error checking coverage: ${error.message}`
    }
  }
})

export const enforceTestFirst = tool({
  description: "Enforce test-first development by checking git diff. Ensures test file was modified/created before or with the implementation file.",
  args: {
    implFile: tool.schema.string().describe("Implementation file path")
  },
  async execute(args) {
    try {
      const implFile = args.implFile
      const testFile = implFile.replace(/\.go$/, "_test.go")

      // Check git status
      const statusResult = await $`git status --porcelain`.text()
      const lines = statusResult.split("\n")

      const implModified = lines.some(line => line.includes(implFile))
      const testModified = lines.some(line => line.includes(testFile))

      if (implModified && !testModified) {
        return `❌ TDD VIOLATION: Implementation modified without test.\n\nFile: ${implFile}\nTest: ${testFile}\n\nYou must modify the test file when changing implementation.`
      }

      if (!existsSync(testFile)) {
        return `❌ TDD VIOLATION: Test file doesn't exist.\n\nCreate: ${testFile}`
      }

      // Check git log to ensure test was committed first or with impl
      const implLog = await $`git log -1 --format=%H -- ${implFile}`.nothrow().text()
      const testLog = await $`git log -1 --format=%H -- ${testFile}`.nothrow().text()

      if (implLog && !testLog) {
        return `⚠️  Implementation committed without test in history.\n\nEnsure test exists and is committed.`
      }

      return `✅ Test-first development validated.\n\nTest: ${testFile}\nImpl: ${implFile}`
    } catch (error) {
      return `Error enforcing test-first: ${error.message}`
    }
  }
})
