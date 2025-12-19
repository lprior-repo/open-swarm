// Copyright (c) 2025 Open Swarm Contributors
//
// This software is released under the MIT License.
// See LICENSE file in the repository for details.

package opencode

import (
	"context"
	"testing"
	"time"
)

func TestNewConfigAwareness(t *testing.T) {
	ca := NewConfigAwareness(1 * time.Hour)

	if ca == nil {
		t.Fatal("NewConfigAwareness returned nil")
	}

	if ca.cacheExpiry != 1*time.Hour {
		t.Errorf("expected cacheExpiry 1h, got %v", ca.cacheExpiry)
	}
}

func TestNewConfigAwareness_DefaultCacheExpiry(t *testing.T) {
	ca := NewConfigAwareness(0)

	if ca.cacheExpiry != 1*time.Hour {
		t.Errorf("expected default cacheExpiry 1h, got %v", ca.cacheExpiry)
	}
}

func TestNewConfigAwareness_NegativeCacheExpiry(t *testing.T) {
	ca := NewConfigAwareness(-1 * time.Hour)

	if ca.cacheExpiry != 1*time.Hour {
		t.Errorf("expected default cacheExpiry 1h for negative input, got %v", ca.cacheExpiry)
	}
}

func TestRefreshConfig(t *testing.T) {
	ctx := context.Background()
	ca := NewConfigAwareness(1 * time.Hour)

	err := ca.RefreshConfig(ctx)
	if err != nil {
		t.Fatalf("RefreshConfig failed: %v", err)
	}

	if !ca.IsCacheFresh() {
		t.Errorf("expected cache to be fresh after RefreshConfig")
	}
}

func TestRefreshConfig_InitializesDefaults(t *testing.T) {
	ctx := context.Background()
	ca := NewConfigAwareness(1 * time.Hour)

	ca.RefreshConfig(ctx)

	models, _ := ca.GetAvailableModels(ctx)
	if len(models) == 0 {
		t.Errorf("expected models to be initialized, got 0")
	}

	providers, _ := ca.GetAvailableProviders(ctx)
	if len(providers) == 0 {
		t.Errorf("expected providers to be initialized, got 0")
	}
}

func TestIsCacheFresh_NotCached(t *testing.T) {
	ca := NewConfigAwareness(1 * time.Hour)

	if ca.IsCacheFresh() {
		t.Errorf("expected cache not fresh when not yet cached")
	}
}

func TestIsCacheFresh_AfterRefresh(t *testing.T) {
	ctx := context.Background()
	ca := NewConfigAwareness(1 * time.Hour)

	ca.RefreshConfig(ctx)

	if !ca.IsCacheFresh() {
		t.Errorf("expected cache to be fresh after refresh")
	}
}

func TestIsCacheFresh_AfterExpiry(t *testing.T) {
	ctx := context.Background()
	ca := NewConfigAwareness(100 * time.Millisecond)

	ca.RefreshConfig(ctx)
	if !ca.IsCacheFresh() {
		t.Errorf("expected cache to be fresh immediately after refresh")
	}

	time.Sleep(150 * time.Millisecond)

	if ca.IsCacheFresh() {
		t.Errorf("expected cache to be stale after expiry")
	}
}

func TestGetAvailableModels(t *testing.T) {
	ctx := context.Background()
	ca := NewConfigAwareness(1 * time.Hour)

	ca.RefreshConfig(ctx)

	models, err := ca.GetAvailableModels(ctx)
	if err != nil {
		t.Fatalf("GetAvailableModels failed: %v", err)
	}

	if len(models) == 0 {
		t.Errorf("expected available models, got 0")
	}

	// All returned models should be available
	for _, model := range models {
		if !model.IsAvailable {
			t.Errorf("GetAvailableModels returned unavailable model: %s", model.ID)
		}
	}
}

func TestGetAvailableModels_FiltersUnavailable(t *testing.T) {
	ctx := context.Background()
	ca := NewConfigAwareness(1 * time.Hour)

	ca.RefreshConfig(ctx)

	models, err := ca.GetAvailableModels(ctx)
	if err != nil {
		t.Fatalf("GetAvailableModels failed: %v", err)
	}

	// Mark one as unavailable
	ca.mu.Lock()
	for _, m := range ca.models {
		m.IsAvailable = false
		break
	}
	ca.mu.Unlock()

	// Get available models again
	models2, _ := ca.GetAvailableModels(ctx)

	// Should have fewer models now
	if len(models2) >= len(models) {
		t.Errorf("expected fewer models after marking one unavailable")
	}
}

func TestGetAvailableProviders(t *testing.T) {
	ctx := context.Background()
	ca := NewConfigAwareness(1 * time.Hour)

	ca.RefreshConfig(ctx)

	providers, err := ca.GetAvailableProviders(ctx)
	if err != nil {
		t.Fatalf("GetAvailableProviders failed: %v", err)
	}

	if len(providers) == 0 {
		t.Errorf("expected available providers, got 0")
	}

	// All returned providers should be available
	for _, provider := range providers {
		if !provider.IsAvailable {
			t.Errorf("GetAvailableProviders returned unavailable provider: %s", provider.ID)
		}
	}
}

func TestSelectModelForTask_Default(t *testing.T) {
	ctx := context.Background()
	ca := NewConfigAwareness(1 * time.Hour)

	ca.RefreshConfig(ctx)

	modelID, err := ca.SelectModelForTask(ctx, "default")
	if err != nil {
		t.Fatalf("SelectModelForTask failed: %v", err)
	}

	if modelID == "" {
		t.Errorf("expected model selection, got empty string")
	}
}

func TestSelectModelForTask_Empty(t *testing.T) {
	ctx := context.Background()
	ca := NewConfigAwareness(1 * time.Hour)

	ca.RefreshConfig(ctx)

	modelID, err := ca.SelectModelForTask(ctx, "")
	if err != nil {
		t.Fatalf("SelectModelForTask with empty string failed: %v", err)
	}

	if modelID == "" {
		t.Errorf("expected model selection for empty string (default), got empty")
	}
}

func TestSelectModelForTask_Analysis(t *testing.T) {
	ctx := context.Background()
	ca := NewConfigAwareness(1 * time.Hour)

	ca.RefreshConfig(ctx)

	modelID, err := ca.SelectModelForTask(ctx, "analysis")
	if err != nil {
		t.Fatalf("SelectModelForTask(analysis) failed: %v", err)
	}

	if modelID == "" {
		t.Errorf("expected model selection for analysis task, got empty")
	}

	// Get the selected model and verify it has large context
	model, _ := ca.GetModelInfo(modelID)
	if model.ContextSize < 100000 {
		t.Errorf("expected large context size for analysis task, got %d", model.ContextSize)
	}
}

func TestSelectModelForTask_Coding(t *testing.T) {
	ctx := context.Background()
	ca := NewConfigAwareness(1 * time.Hour)

	ca.RefreshConfig(ctx)

	modelID, err := ca.SelectModelForTask(ctx, "coding")
	if err != nil {
		t.Fatalf("SelectModelForTask(coding) failed: %v", err)
	}

	if modelID == "" {
		t.Errorf("expected model selection for coding task, got empty")
	}

	// Selected model should have code_generation capability
	model, _ := ca.GetModelInfo(modelID)
	hasCodeGenCap := false
	for _, cap := range model.Capabilities {
		if cap == "code_generation" {
			hasCodeGenCap = true
			break
		}
	}
	if !hasCodeGenCap {
		t.Errorf("selected coding model does not have code_generation capability")
	}
}

func TestSelectModelForTask_Summarization(t *testing.T) {
	ctx := context.Background()
	ca := NewConfigAwareness(1 * time.Hour)

	ca.RefreshConfig(ctx)

	modelID, err := ca.SelectModelForTask(ctx, "summarization")
	if err != nil {
		t.Fatalf("SelectModelForTask(summarization) failed: %v", err)
	}

	if modelID == "" {
		t.Errorf("expected model selection for summarization task, got empty")
	}

	// Selected model should be one of the cheaper options
	model, _ := ca.GetModelInfo(modelID)
	if model.CostPer1K > 0.01 {
		t.Logf("warning: summarization model cost is high: %f per 1K tokens", model.CostPer1K)
	}
}

func TestSelectModelForTask_NoAvailableModels(t *testing.T) {
	ctx := context.Background()
	ca := NewConfigAwareness(1 * time.Hour)

	ca.RefreshConfig(ctx)

	// Clear all models
	ca.mu.Lock()
	ca.models = make(map[string]*ModelInfo)
	ca.mu.Unlock()

	_, err := ca.SelectModelForTask(ctx, "default")
	if err == nil {
		t.Errorf("expected error when no models available")
	}
}

func TestGetModelInfo(t *testing.T) {
	ctx := context.Background()
	ca := NewConfigAwareness(1 * time.Hour)

	ca.RefreshConfig(ctx)

	models, _ := ca.GetAvailableModels(ctx)
	if len(models) == 0 {
		t.Skip("no available models to test")
	}

	info, err := ca.GetModelInfo(models[0].ID)
	if err != nil {
		t.Fatalf("GetModelInfo failed: %v", err)
	}

	if info.ID != models[0].ID {
		t.Errorf("expected model %s, got %s", models[0].ID, info.ID)
	}
}

func TestGetModelInfo_Nonexistent(t *testing.T) {
	ctx := context.Background()
	ca := NewConfigAwareness(1 * time.Hour)

	ca.RefreshConfig(ctx)

	_, err := ca.GetModelInfo("nonexistent-model")
	if err == nil {
		t.Errorf("expected error for nonexistent model")
	}
}

func TestGetProviderInfo(t *testing.T) {
	ctx := context.Background()
	ca := NewConfigAwareness(1 * time.Hour)

	ca.RefreshConfig(ctx)

	providers, _ := ca.GetAvailableProviders(ctx)
	if len(providers) == 0 {
		t.Skip("no available providers to test")
	}

	info, err := ca.GetProviderInfo(providers[0].ID)
	if err != nil {
		t.Fatalf("GetProviderInfo failed: %v", err)
	}

	if info.ID != providers[0].ID {
		t.Errorf("expected provider %s, got %s", providers[0].ID, info.ID)
	}
}

func TestGetProviderInfo_Nonexistent(t *testing.T) {
	ctx := context.Background()
	ca := NewConfigAwareness(1 * time.Hour)

	ca.RefreshConfig(ctx)

	_, err := ca.GetProviderInfo("nonexistent-provider")
	if err == nil {
		t.Errorf("expected error for nonexistent provider")
	}
}

func TestClearCache(t *testing.T) {
	ctx := context.Background()
	ca := NewConfigAwareness(1 * time.Hour)

	ca.RefreshConfig(ctx)
	if !ca.IsCacheFresh() {
		t.Errorf("expected fresh cache after refresh")
	}

	ca.ClearCache()

	if ca.IsCacheFresh() {
		t.Errorf("expected stale cache after ClearCache")
	}
}

func TestCacheExpiry_AutoRefresh(t *testing.T) {
	ctx := context.Background()
	ca := NewConfigAwareness(100 * time.Millisecond)

	ca.RefreshConfig(ctx)
	models1, _ := ca.GetAvailableModels(ctx)

	time.Sleep(150 * time.Millisecond)

	// This should trigger an auto-refresh since cache is stale
	models2, _ := ca.GetAvailableModels(ctx)

	if len(models1) == 0 || len(models2) == 0 {
		t.Errorf("expected models in both calls")
	}
}

func TestModelInfoStructure(t *testing.T) {
	ca := NewConfigAwareness(1 * time.Hour)
	ctx := context.Background()

	ca.RefreshConfig(ctx)

	models, _ := ca.GetAvailableModels(ctx)
	if len(models) == 0 {
		t.Skip("no models to test")
	}

	model := models[0]

	if model.ID == "" {
		t.Errorf("ModelInfo.ID should not be empty")
	}

	if model.Provider == "" {
		t.Errorf("ModelInfo.Provider should not be empty")
	}

	if model.DisplayName == "" {
		t.Errorf("ModelInfo.DisplayName should not be empty")
	}

	if model.ContextSize <= 0 {
		t.Errorf("ModelInfo.ContextSize should be positive")
	}

	if model.CostPer1K < 0 {
		t.Errorf("ModelInfo.CostPer1K should not be negative")
	}

	if !model.IsAvailable {
		t.Errorf("returned models should be available")
	}

	if len(model.Capabilities) == 0 {
		t.Logf("warning: model %s has no capabilities defined", model.ID)
	}
}

func TestProviderInfoStructure(t *testing.T) {
	ca := NewConfigAwareness(1 * time.Hour)
	ctx := context.Background()

	ca.RefreshConfig(ctx)

	providers, _ := ca.GetAvailableProviders(ctx)
	if len(providers) == 0 {
		t.Skip("no providers to test")
	}

	provider := providers[0]

	if provider.ID == "" {
		t.Errorf("ProviderInfo.ID should not be empty")
	}

	if provider.Name == "" {
		t.Errorf("ProviderInfo.Name should not be empty")
	}

	if !provider.IsAvailable {
		t.Errorf("returned providers should be available")
	}

	if len(provider.Models) == 0 {
		t.Errorf("ProviderInfo.Models should not be empty")
	}
}

func TestMultipleTaskTypeSelections(t *testing.T) {
	ctx := context.Background()
	ca := NewConfigAwareness(1 * time.Hour)

	ca.RefreshConfig(ctx)

	taskTypes := []string{"analysis", "coding", "summarization", "default"}

	selectedModels := make(map[string]string)

	for _, taskType := range taskTypes {
		modelID, err := ca.SelectModelForTask(ctx, taskType)
		if err != nil {
			t.Errorf("SelectModelForTask(%s) failed: %v", taskType, err)
		}

		if modelID != "" {
			selectedModels[taskType] = modelID
		}
	}

	if len(selectedModels) != len(taskTypes) {
		t.Errorf("expected selection for all task types")
	}
}

func TestConcurrentCacheAccess(t *testing.T) {
	ctx := context.Background()
	ca := NewConfigAwareness(1 * time.Hour)

	ca.RefreshConfig(ctx)

	// Concurrent reads
	done := make(chan bool, 10)

	for i := 0; i < 10; i++ {
		go func() {
			_, _ = ca.GetAvailableModels(ctx)
			_, _ = ca.GetAvailableProviders(ctx)
			done <- true
		}()
	}

	for i := 0; i < 10; i++ {
		<-done
	}

	if !ca.IsCacheFresh() {
		t.Errorf("expected cache to still be fresh after concurrent reads")
	}
}

func TestConcurrentCacheRefresh(t *testing.T) {
	ctx := context.Background()
	ca := NewConfigAwareness(1 * time.Hour)

	done := make(chan bool, 5)

	for i := 0; i < 5; i++ {
		go func() {
			_ = ca.RefreshConfig(ctx)
			done <- true
		}()
	}

	for i := 0; i < 5; i++ {
		<-done
	}

	if !ca.IsCacheFresh() {
		t.Errorf("expected cache to be fresh after concurrent refreshes")
	}
}
