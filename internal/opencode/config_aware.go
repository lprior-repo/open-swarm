// Copyright (c) 2025 Open Swarm Contributors
//
// This software is released under the MIT License.
// See LICENSE file in the repository for details.

package opencode

import (
	"context"
	"fmt"
	"sync"
	"time"
)

// ModelInfo represents information about an available AI model.
type ModelInfo struct {
	ID           string        // Model identifier (e.g., "claude-3-opus", "gpt-4")
	Provider     string        // Provider name (e.g., "anthropic", "openai")
	DisplayName  string        // Human-readable name
	ContextSize  int           // Max context window in tokens
	CostPer1K    float64       // Cost per 1000 tokens
	IsAvailable  bool          // Whether model is currently available
	Capabilities []string      // Supported features (e.g., "vision", "function_calling")
}

// ProviderInfo represents information about an AI provider.
type ProviderInfo struct {
	ID          string   // Provider identifier (e.g., "anthropic", "openai")
	Name        string   // Display name
	IsAvailable bool     // Whether provider is currently accessible
	Models      []string // List of available model IDs for this provider
	Config      map[string]interface{} // Provider-specific configuration
}

// ConfigAwareness provides intelligent provider and model selection based on cached configuration.
// It discovers available models and providers from the OpenCode SDK and caches the results
// to improve performance and enable cost-aware task execution.
type ConfigAwareness struct {
	mu sync.RWMutex

	// Cached configuration
	models         map[string]*ModelInfo
	providers      map[string]*ProviderInfo
	defaultModel   string
	defaultProvider string
	cachedAt       time.Time
	cacheExpiry    time.Duration

	// Integration points (placeholder for OpenCode SDK client)
	// In a real implementation, this would hold: client *opencode.Client
}

// NewConfigAwareness creates a new ConfigAwareness instance with default cache expiry.
// cacheExpiry: how long to keep cached configuration (default: 1 hour)
func NewConfigAwareness(cacheExpiry time.Duration) *ConfigAwareness {
	if cacheExpiry <= 0 {
		cacheExpiry = 1 * time.Hour
	}

	return &ConfigAwareness{
		models:      make(map[string]*ModelInfo),
		providers:   make(map[string]*ProviderInfo),
		cacheExpiry: cacheExpiry,
		cachedAt:    time.Time{}, // Not cached yet
	}
}

// RefreshConfig queries the OpenCode Config service to discover available models and providers.
// This should be called on startup and periodically to stay up-to-date with provider changes.
func (ca *ConfigAwareness) RefreshConfig(ctx context.Context) error {
	ca.mu.Lock()
	defer ca.mu.Unlock()

	// TODO: Integrate with OpenCode SDK Config service
	// Example integration (when SDK available):
	/*
	config, err := ca.client.Config.Get(ctx, opencode.ConfigGetParams{})
	if err != nil {
		return fmt.Errorf("failed to fetch config: %w", err)
	}

	// Parse models from config
	ca.models = parseModels(config.AvailableModels)
	ca.providers = parseProviders(config.AvailableProviders)
	ca.defaultModel = config.DefaultModel
	ca.defaultProvider = config.DefaultProvider
	ca.cachedAt = time.Now()

	return nil
	*/

	// Placeholder: Initialize with sensible defaults
	ca.initializeDefaults()
	ca.cachedAt = time.Now()
	return nil
}

// GetAvailableModels returns the list of all available models.
// Returns cached results if available; returns error if cache is stale and refresh fails.
func (ca *ConfigAwareness) GetAvailableModels(ctx context.Context) ([]*ModelInfo, error) {
	if err := ca.ensureFreshConfig(ctx); err != nil {
		return nil, err
	}

	ca.mu.RLock()
	defer ca.mu.RUnlock()

	models := make([]*ModelInfo, 0, len(ca.models))
	for _, m := range ca.models {
		if m.IsAvailable {
			models = append(models, m)
		}
	}
	return models, nil
}

// GetAvailableProviders returns the list of all available providers.
func (ca *ConfigAwareness) GetAvailableProviders(ctx context.Context) ([]*ProviderInfo, error) {
	if err := ca.ensureFreshConfig(ctx); err != nil {
		return nil, err
	}

	ca.mu.RLock()
	defer ca.mu.RUnlock()

	providers := make([]*ProviderInfo, 0, len(ca.providers))
	for _, p := range ca.providers {
		if p.IsAvailable {
			providers = append(providers, p)
		}
	}
	return providers, nil
}

// SelectModelForTask intelligently selects a model based on task type and constraints.
// Uses heuristics to balance cost, capability, and availability.
//
// Task types:
//   - "analysis": Prefers models with large context windows
//   - "coding": Prefers models with strong code generation capability
//   - "summarization": Prefers cost-effective models
//   - "default": Uses default model or most capable available
func (ca *ConfigAwareness) SelectModelForTask(ctx context.Context, taskType string) (string, error) {
	models, err := ca.GetAvailableModels(ctx)
	if err != nil {
		return "", err
	}

	if len(models) == 0 {
		return "", fmt.Errorf("no available models")
	}

	ca.mu.RLock()
	defaultModel := ca.defaultModel
	ca.mu.RUnlock()

	// Task-specific selection heuristics
	switch taskType {
	case "analysis":
		// Prefer large context window models
		return selectByContextSize(models, true), nil

	case "coding":
		// Prefer models with coding capability
		return selectByCapability(models, "code_generation"), nil

	case "summarization":
		// Prefer cost-effective models
		return selectByCost(models, true), nil

	case "default", "":
		fallthrough
	default:
		// Return default if available, otherwise most capable
		for _, m := range models {
			if m.ID == defaultModel {
				return m.ID, nil
			}
		}
		// Fallback to most capable (largest context)
		return selectByContextSize(models, true), nil
	}
}

// GetModelInfo returns detailed information about a specific model.
func (ca *ConfigAwareness) GetModelInfo(modelID string) (*ModelInfo, error) {
	ca.mu.RLock()
	defer ca.mu.RUnlock()

	model, exists := ca.models[modelID]
	if !exists {
		return nil, fmt.Errorf("model %s not found in configuration", modelID)
	}

	if !model.IsAvailable {
		return nil, fmt.Errorf("model %s is not currently available", modelID)
	}

	return model, nil
}

// GetProviderInfo returns detailed information about a specific provider.
func (ca *ConfigAwareness) GetProviderInfo(providerID string) (*ProviderInfo, error) {
	ca.mu.RLock()
	defer ca.mu.RUnlock()

	provider, exists := ca.providers[providerID]
	if !exists {
		return nil, fmt.Errorf("provider %s not found in configuration", providerID)
	}

	if !provider.IsAvailable {
		return nil, fmt.Errorf("provider %s is not currently available", providerID)
	}

	return provider, nil
}

// IsCacheFresh returns whether the cached configuration is still valid.
func (ca *ConfigAwareness) IsCacheFresh() bool {
	ca.mu.RLock()
	defer ca.mu.RUnlock()

	if ca.cachedAt.IsZero() {
		return false
	}
	return time.Since(ca.cachedAt) < ca.cacheExpiry
}

// ClearCache clears the cached configuration, forcing a refresh on next access.
func (ca *ConfigAwareness) ClearCache() {
	ca.mu.Lock()
	defer ca.mu.Unlock()

	ca.models = make(map[string]*ModelInfo)
	ca.providers = make(map[string]*ProviderInfo)
	ca.cachedAt = time.Time{}
}

// Private helpers

// ensureFreshConfig refreshes config if cache is stale.
func (ca *ConfigAwareness) ensureFreshConfig(ctx context.Context) error {
	if !ca.IsCacheFresh() {
		return ca.RefreshConfig(ctx)
	}
	return nil
}

// initializeDefaults sets up sensible default models and providers.
// This is used as placeholder until SDK integration is available.
func (ca *ConfigAwareness) initializeDefaults() {
	// Initialize providers
	ca.providers = map[string]*ProviderInfo{
		"anthropic": {
			ID:          "anthropic",
			Name:        "Anthropic",
			IsAvailable: true,
			Models:      []string{"claude-3-opus", "claude-3-sonnet", "claude-3-haiku"},
		},
		"openai": {
			ID:          "openai",
			Name:        "OpenAI",
			IsAvailable: true,
			Models:      []string{"gpt-4", "gpt-4-turbo", "gpt-3.5-turbo"},
		},
	}

	// Initialize models with realistic capabilities
	ca.models = map[string]*ModelInfo{
		"claude-3-opus": {
			ID:          "claude-3-opus",
			Provider:    "anthropic",
			DisplayName: "Claude 3 Opus",
			ContextSize: 200000,
			CostPer1K:   0.015,
			IsAvailable: true,
			Capabilities: []string{"vision", "code_generation", "analysis", "reasoning"},
		},
		"claude-3-sonnet": {
			ID:          "claude-3-sonnet",
			Provider:    "anthropic",
			DisplayName: "Claude 3 Sonnet",
			ContextSize: 200000,
			CostPer1K:   0.003,
			IsAvailable: true,
			Capabilities: []string{"vision", "code_generation"},
		},
		"claude-3-haiku": {
			ID:          "claude-3-haiku",
			Provider:    "anthropic",
			DisplayName: "Claude 3 Haiku",
			ContextSize: 200000,
			CostPer1K:   0.00025,
			IsAvailable: true,
			Capabilities: []string{"fast", "low_cost"},
		},
		"gpt-4": {
			ID:          "gpt-4",
			Provider:    "openai",
			DisplayName: "GPT-4",
			ContextSize: 128000,
			CostPer1K:   0.03,
			IsAvailable: true,
			Capabilities: []string{"vision", "code_generation", "function_calling"},
		},
		"gpt-4-turbo": {
			ID:          "gpt-4-turbo",
			Provider:    "openai",
			DisplayName: "GPT-4 Turbo",
			ContextSize: 128000,
			CostPer1K:   0.01,
			IsAvailable: true,
			Capabilities: []string{"vision", "code_generation"},
		},
	}

	// Set sensible defaults
	ca.defaultModel = "claude-3-opus"
	ca.defaultProvider = "anthropic"
}

// selectByContextSize returns the model with largest (if largest=true) or smallest context.
func selectByContextSize(models []*ModelInfo, largest bool) string {
	if len(models) == 0 {
		return ""
	}

	best := models[0]
	for _, m := range models[1:] {
		if largest && m.ContextSize > best.ContextSize {
			best = m
		} else if !largest && m.ContextSize < best.ContextSize {
			best = m
		}
	}
	return best.ID
}

// selectByCapability returns a model that supports the given capability.
func selectByCapability(models []*ModelInfo, capability string) string {
	for _, m := range models {
		for _, cap := range m.Capabilities {
			if cap == capability {
				return m.ID
			}
		}
	}
	// Fallback to first available
	if len(models) > 0 {
		return models[0].ID
	}
	return ""
}

// selectByCost returns the cheapest (if cheap=true) or most expensive model.
func selectByCost(models []*ModelInfo, cheap bool) string {
	if len(models) == 0 {
		return ""
	}

	best := models[0]
	for _, m := range models[1:] {
		if cheap && m.CostPer1K < best.CostPer1K {
			best = m
		} else if !cheap && m.CostPer1K > best.CostPer1K {
			best = m
		}
	}
	return best.ID
}

// Usage examples for agents:
//
// Initializing config awareness on startup:
//   ca := NewConfigAwareness(1 * time.Hour)
//   err := ca.RefreshConfig(ctx)
//   // Agents now have current provider/model info
//
// Selecting optimal model for task:
//   modelID, err := ca.SelectModelForTask(ctx, "coding")
//   // Agent uses selected model for code generation
//
// Cost-aware task execution:
//   models, _ := ca.GetAvailableModels(ctx)
//   for _, model := range models {
//     cost := estimateCost(model.CostPer1K, tokenCount)
//     if cost <= budget {
//       executeTask(model.ID)
//     }
//   }
//
// Provider fallback handling:
//   model1, _ := ca.GetModelInfo("claude-3-opus")
//   if !model1.IsAvailable {
//     // Try next available provider
//     model2, _ := ca.SelectModelForTask(ctx, "default")
//   }
