package gates

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestTestImmutabilityGate_LockFile verifies test file is locked to read-only.
func TestTestImmutabilityGate_LockFile(t *testing.T) {
	// Create temporary test file
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "test.go")
	if err := os.WriteFile(testFile, []byte("func TestExample(t *testing.T) { }"), 0o600); err != nil {
		t.Fatalf("failed to create test file: %v", err)
	}

	// Create temporary test binary
	binaryFile := filepath.Join(tmpDir, "test.bin")
	if err := os.WriteFile(binaryFile, []byte("#!/bin/bash\necho test"), 0o700); err != nil { //nolint:gosec
		t.Fatalf("failed to create test binary: %v", err)
	}

	gate := NewTestImmutabilityGate("task-1", testFile)
	gate.SetTestBinary(binaryFile)

	// Check initial file permissions
	info, _ := os.Stat(testFile)
	if info.Mode()&0o200 == 0 {
		t.Fatal("test file should be writable before gate execution")
	}

	// Execute gate
	if err := gate.Check(context.Background()); err != nil {
		t.Fatalf("gate should not fail with valid test file: %v", err)
	}

	// Verify file is now read-only
	info, _ = os.Stat(testFile)
	if info.Mode()&0o200 != 0 {
		t.Fatal("test file should be read-only after gate execution")
	}

	// Cleanup
	if err := gate.UnlockTestFile(); err != nil {
		t.Logf("cleanup error: %v", err)
	}
}

// TestTestImmutabilityGate_DetectModification verifies file tampering is detected.
func TestTestImmutabilityGate_DetectModification(t *testing.T) {
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "test.go")
	originalContent := "func TestExample(t *testing.T) { }"
	if err := os.WriteFile(testFile, []byte(originalContent), 0o600); err != nil {
		t.Fatalf("failed to create test file: %v", err)
	}

	gate := NewTestImmutabilityGate("task-1", testFile)
	if err := gate.Check(context.Background()); err != nil {
		t.Logf("gate check (expected to fail for modify test): %v", err)
	}

	// Save original hash
	originalHash := gate.originalHash

	// Attempt to modify (should fail due to permissions)
	err := os.WriteFile(testFile, []byte("modified content"), 0o644) //nolint:gosec
	if err == nil {
		// If we could modify (e.g., running as root), manually check hash mismatch
		if err := gate.verifyTestFileIntegrity(); err == nil {
			t.Fatal("gate should detect file modification")
		}
	} else {
		// Good - file modification was prevented
		if err := gate.verifyTestFileIntegrity(); err != nil {
			t.Fatalf("gate should not error on unmodified file: %v", err)
		}
	}

	// Verify original hash is still recorded
	if gate.originalHash != originalHash {
		t.Fatal("original hash should remain unchanged")
	}

	if err := gate.UnlockTestFile(); err != nil {
		t.Logf("cleanup error: %v", err)
	}
}

// TestTestImmutabilityGate_MissingBinary fails if test binary not set.
func TestTestImmutabilityGate_MissingBinary(t *testing.T) {
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "test.go")
	if err := os.WriteFile(testFile, []byte("func TestExample(t *testing.T) { }"), 0o600); err != nil {
		t.Fatalf("failed to create test file: %v", err)
	}

	gate := NewTestImmutabilityGate("task-1", testFile)
	// Don't set test binary

	err := gate.Check(context.Background())
	if err == nil {
		t.Fatal("gate should fail when test binary not set")
	}

	if gateErr, ok := err.(*GateError); !ok || gateErr.Gate != GateTestImmutability { //nolint:errorlint
		t.Fatal("error should be from test immutability gate")
	}
}

// TestEmpiricalHonestyGate_FailOnFalseSuccess verifies honest test reporting.
func TestEmpiricalHonestyGate_FailOnFalseSuccess(t *testing.T) {
	gate := NewEmpiricalHonestyGate("task-1")

	// Agent claims success but tests are failing
	gate.SetAgentClaim("Implementation complete and all tests pass!")
	gate.SetTestResult(&TestResult{
		Total:    5,
		Passed:   3,
		Failed:   2,
		Output:   "FAILED: test_foo, test_bar",
		ExitCode: 1,
	})

	err := gate.Check(context.Background())
	if err == nil {
		t.Fatal("gate should reject false success claim")
	}

	if gateErr, ok := err.(*GateError); !ok || gateErr.Gate != GateEmpiricalHonesty { //nolint:errorlint
		t.Fatal("error should be from empirical honesty gate")
	}
}

// TestEmpiricalHonestyGate_AllowHonestFailure allows honest failure reports.
func TestEmpiricalHonestyGate_AllowHonestFailure(t *testing.T) {
	gate := NewEmpiricalHonestyGate("task-1")

	// Agent honestly reports failure
	gate.SetAgentClaim("Tests are still failing. Need to debug implementation.")
	gate.SetTestResult(&TestResult{
		Total:    5,
		Passed:   2,
		Failed:   3,
		Output:   "FAILED: test_foo, test_bar, test_baz",
		Failures: []string{"test_foo: expected X, got Y"},
		ExitCode: 1,
	})

	err := gate.Check(context.Background())
	if err != nil {
		t.Fatalf("gate should allow honest failure report: %v", err)
	}
}

// TestEmpiricalHonestyGate_RequireRawOutput fails without raw test output.
func TestEmpiricalHonestyGate_RequireRawOutput(t *testing.T) {
	gate := NewEmpiricalHonestyGate("task-1")

	gate.SetAgentClaim("Tests failed")
	gate.SetTestResult(&TestResult{
		Total:  5,
		Passed: 2,
		Failed: 3,
		Output: "", // Empty output
		// No failure details
		ExitCode: 1,
	})

	err := gate.Check(context.Background())
	if err == nil {
		t.Fatal("gate should require raw test output")
	}
}

// TestHardWorkEnforcementGate_RejectStubCode rejects stubbed implementations.
func TestHardWorkEnforcementGate_RejectStubCode(t *testing.T) {
	gate := NewHardWorkEnforcementGate("task-1", "/impl/main.go")

	// Stub implementation with panic
	stubCode := `
func ProcessData(data []int) int {
	panic("not implemented")
}
`
	gate.SetImplementationCode(stubCode)
	gate.SetTestResult(&TestResult{
		Total:    3,
		Passed:   3,
		Failed:   0, // Stub passed tests (shouldn't happen)
		ExitCode: 0,
	})

	err := gate.Check(context.Background())
	if err == nil {
		t.Fatal("gate should reject stubbed code that passes tests")
	}

	if gateErr, ok := err.(*GateError); !ok || gateErr.Gate != GateHardWork { //nolint:errorlint
		t.Fatal("error should be from hard work enforcement gate")
	}
}

// TestHardWorkEnforcementGate_AllowFailingStubs allows failing stubs during development.
func TestHardWorkEnforcementGate_AllowFailingStubs(t *testing.T) {
	gate := NewHardWorkEnforcementGate("task-1", "/impl/main.go")

	// Stub implementation that correctly fails tests
	stubCode := `
func ProcessData(data []int) int {
	return 0 // Stub
}

func helper() {
	panic("not implemented")
}

func validate(x int) bool {
	return false
}

func transform(s string) string {
	return ""
}

func calculate(a, b int) int {
	return 0
}
`
	gate.SetImplementationCode(stubCode)
	gate.SetTestResult(&TestResult{
		Total:    5,
		Passed:   2,
		Failed:   3, // Failing stubs - correct behavior
		Output:   "FAILED: test_process, test_validate, test_calculate",
		ExitCode: 1,
	})

	err := gate.Check(context.Background())
	if err != nil {
		t.Fatalf("gate should allow failing stubs: %v", err)
	}
}

// TestHardWorkEnforcementGate_DetectCheating detects test framework bypass.
func TestHardWorkEnforcementGate_DetectCheating(t *testing.T) {
	gate := NewHardWorkEnforcementGate("task-1", "/impl/main.go")

	// Code attempting to bypass test framework
	cheatCode := `
func ProcessData(data []int) int {
	// Trying to suppress errors
	suppressExitCode()
	return 0
}

func suppressExitCode() {
	// Try to override test exit code
}
`
	gate.SetImplementationCode(cheatCode)
	gate.SetTestResult(&TestResult{
		Total:    3,
		Passed:   3,
		Failed:   0,
		ExitCode: 0,
	})

	err := gate.Check(context.Background())
	if err == nil {
		t.Fatal("gate should detect cheating patterns")
	}
}

// TestRequirementDriftDetectionGate_DetectDrift verifies requirement alignment.
func TestRequirementDriftDetectionGate_DetectDrift(t *testing.T) {
	req := &Requirement{
		TaskID:      "task-1",
		Title:       "String Validation",
		Description: "Implement a function to validate email addresses and phone numbers",
		Acceptance:  "Function must validate email format and phone format",
		Scenarios:   []string{"validate email", "validate phone"},
		EdgeCases:   []string{"empty string", "special characters"},
	}

	gate := NewRequirementDriftDetectionGate("task-1", req)

	// Implementation that drifts from requirement (does something else)
	driftCode := `
func ProcessData(data []int) int {
	// Just sorting data, not validating emails/phones
	return len(data)
}
`
	gate.SetCurrentImplementation(driftCode)

	err := gate.Check(context.Background())
	if err == nil {
		t.Fatal("gate should detect requirement drift")
	}

	if gateErr, ok := err.(*GateError); !ok || gateErr.Gate != GateDriftDetection { //nolint:errorlint
		t.Fatal("error should be from drift detection gate")
	}
}

// TestRequirementDriftDetectionGate_AllowAlignedCode allows implementations that match requirement.
func TestRequirementDriftDetectionGate_AllowAlignedCode(t *testing.T) {
	req := &Requirement{
		TaskID:      "task-1",
		Title:       "String Validation Function",
		Description: "Implement email and phone validation functions",
		Acceptance:  "must validate email; must validate phone; must check format",
		Scenarios:   []string{"email", "phone"},
	}

	gate := NewRequirementDriftDetectionGate("task-1", req)

	// Implementation that matches requirement - must include acceptance criteria phrases
	alignedCode := `
func ValidateEmail(email string) bool {
	// must validate email - Email validation implementation
	if len(email) == 0 {
		return false
	}
	return contains(email, "@") && validateEmail(email)
}

func ValidatePhone(phone string) bool {
	// must validate phone - Phone validation implementation
	if len(phone) < 10 {
		return false
	}
	return validatePhone(phone)
}

func validateEmail(email string) bool {
	// must check format - Email format checking
	return contains(email, ".")
}

func validatePhone(phone string) bool {
	// must check format - Phone format validation
	return true
}

func contains(s, substr string) bool {
	return true
}
`
	gate.SetCurrentImplementation(alignedCode)

	err := gate.Check(context.Background())
	if err != nil {
		t.Fatalf("gate should allow aligned implementation: %v", err)
	}
}

// TestRequirementDriftDetectionGate_TokenBudgetTriggersCheck verifies periodic checking.
func TestRequirementDriftDetectionGate_TokenBudgetTriggersCheck(t *testing.T) {
	req := &Requirement{
		TaskID:      "task-1",
		Title:       "Simple task",
		Description: "Implement function",
	}

	gate := NewRequirementDriftDetectionGate("task-1", req)
	gate.SetTokenBudget(100)
	gate.SetCurrentImplementation("func Foo() { }")

	// Add tokens in small chunks
	for i := 0; i < 3; i++ {
		gate.AddTokens(40)
	}

	// After 120 tokens total, should have auto-checked
	if len(gate.checkpoints) == 0 {
		t.Fatal("gate should auto-check when token budget exceeded")
	}

	if gate.tokensSinceCheck > 0 && gate.tokensSinceCheck < 20 {
		t.Logf("Tokens reset after check: %d (expected < 20)", gate.tokensSinceCheck)
	}
}

// TestGateChain_SequentialExecution runs gates in order.
func TestGateChain_SequentialExecution(t *testing.T) {
	req := &Requirement{
		TaskID:      "task-1",
		Title:       "Test task",
		Description: "Test description",
		Scenarios:   []string{"scenario1"},
	}

	gate1 := NewRequirementsVerificationGate("task-1", req)
	gate1.SetGeneratedTests([]string{"test_scenario1_success"})

	gate2 := NewEmpiricalHonestyGate("task-1")
	gate2.SetAgentClaim("Tests pass")
	gate2.SetTestResult(&TestResult{
		Total:    1,
		Passed:   1,
		Failed:   0,
		Output:   "PASSED: test_scenario1",
		ExitCode: 0,
	})

	chain := NewGateChain(gate1, gate2)
	err := chain.Execute(context.Background())

	if err != nil {
		t.Fatalf("gate chain should succeed with valid gates: %v", err)
	}
}

// TestGateChain_StopsOnFirstFailure stops on first gate failure.
func TestGateChain_StopsOnFirstFailure(t *testing.T) {
	req := &Requirement{
		TaskID:      "task-1",
		Title:       "Test task",
		Description: "Test",
		Scenarios:   []string{"scenario1"},
	}

	// First gate will fail (no tests generated)
	gate1 := NewRequirementsVerificationGate("task-1", req)
	// Don't set generated tests

	// Second gate would fail too
	gate2 := NewEmpiricalHonestyGate("task-1")
	gate2.SetAgentClaim("Success")
	gate2.SetTestResult(&TestResult{
		Total:  1,
		Passed: 0,
		Failed: 1,
	})

	chain := NewGateChain(gate1, gate2)
	err := chain.Execute(context.Background())

	if err == nil {
		t.Fatal("gate chain should fail on first gate failure")
	}

	// Verify it's gate1 that failed, not gate2
	if gateErr, ok := err.(*GateError); !ok || gateErr.Gate != GateRequirements { //nolint:errorlint
		t.Fatal("error should be from requirements gate (first gate)")
	}
}

// TestGateBuilder constructs gates fluently.
func TestGateBuilder(t *testing.T) {
	req := &Requirement{
		TaskID:      "task-1",
		Title:       "Task",
		Description: "Description",
		Scenarios:   []string{"scenario"},
	}

	chain := NewGateBuilder().
		Add(NewRequirementsVerificationGate("task-1", req)).
		Add(NewEmpiricalHonestyGate("task-1")).
		Build()

	if chain == nil {
		t.Fatal("builder should construct gate chain")
	}

	if len(chain.gates) != 2 {
		t.Fatalf("chain should have 2 gates, got %d", len(chain.gates))
	}
}

// TestGateChain_ParallelExecution runs gates in parallel.
func TestGateChain_ParallelExecution(t *testing.T) {
	req := &Requirement{
		TaskID:      "task-1",
		Title:       "Task",
		Description: "Description",
		Scenarios:   []string{"scenario"},
	}

	gate1 := NewRequirementsVerificationGate("task-1", req)
	gate1.SetGeneratedTests([]string{"test_scenario_success"})

	gate2 := NewEmpiricalHonestyGate("task-1")
	gate2.SetAgentClaim("Passing")
	gate2.SetTestResult(&TestResult{
		Total:    1,
		Passed:   1,
		Failed:   0,
		Output:   "OK",
		ExitCode: 0,
	})

	chain := NewGateChain(gate1, gate2)

	// Parallel execution should return any errors that occurred
	errs := chain.ExecuteParallel(context.Background())

	if len(errs) > 0 {
		t.Logf("Parallel execution returned %d errors", len(errs))
		for _, err := range errs {
			if err != nil {
				t.Fatalf("gate should pass: %v", err)
			}
		}
	}
}

// BenchmarkGateExecution benchmarks gate checking speed.
func BenchmarkGateExecution(b *testing.B) {
	req := &Requirement{
		TaskID:      "task-1",
		Title:       "Benchmark task",
		Description: "Long requirement " + strings.Repeat("x", 1000),
		Scenarios:   []string{"scenario1", "scenario2", "scenario3"},
	}

	gate := NewRequirementsVerificationGate("task-1", req)
	gate.SetGeneratedTests([]string{"test_s1", "test_s2", "test_s3"})

	ctx := context.Background()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = gate.Check(ctx)
	}
}
