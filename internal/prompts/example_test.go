package prompts_test

import (
	"fmt"
	"open-swarm/internal/prompts"
)

// Example demonstrates basic usage of the prompt builders
func Example() {
	// Create a review request
	request := prompts.ReviewRequest{
		Type:            prompts.ReviewTypeArchitecture,
		TaskID:          "TASK-123",
		TaskDescription: "Implement user authentication service",
		CodeContext: prompts.CodeContext{
			FilePath:    "auth/service.go",
			FileContent: "package auth\n\ntype Service struct {}\n\nfunc (s *Service) Login() error { return nil }",
			Language:    "go",
			PackageName: "auth",
		},
		AcceptanceCriteria: []string{
			"Must follow SOLID principles",
			"Must be easily testable",
			"Must handle errors gracefully",
		},
		RequireVote: true,
	}

	// Build the prompt
	prompt, err := prompts.BuildArchitecturePrompt(request)
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		return
	}

	fmt.Printf("Generated prompt length: %d characters\n", len(prompt))
	// Output: Generated prompt length: 3099 characters
}

// Example_buildAllReviewPrompts shows how to generate all three review types
func Example_buildAllReviewPrompts() {
	request := prompts.ReviewRequest{
		TaskID:          "TASK-789",
		TaskDescription: "Refactor payment processing",
		CodeContext: prompts.CodeContext{
			FilePath:    "payment/processor.go",
			FileContent: "package payment\n\nfunc Process() error { return nil }",
			Language:    "go",
		},
		RequireVote: true,
	}

	// Generate all three types of review prompts
	allPrompts, err := prompts.BuildAllReviewPrompts(request)
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		return
	}

	fmt.Printf("Generated %d review prompts\n", len(allPrompts))
	fmt.Printf("Architecture review: %v\n", len(allPrompts[prompts.ReviewTypeArchitecture]) > 0)
	fmt.Printf("Functional review: %v\n", len(allPrompts[prompts.ReviewTypeFunctional]) > 0)
	fmt.Printf("Testing review: %v\n", len(allPrompts[prompts.ReviewTypeTesting]) > 0)
	// Output:
	// Generated 3 review prompts
	// Architecture review: true
	// Functional review: true
	// Testing review: true
}
