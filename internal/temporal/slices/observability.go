// Copyright (c) 2025 Open Swarm Contributors
//
// This software is released under the MIT License.
// See LICENSE file in the repository for details.

package slices

import (
	"sync"
	"time"

	"go.temporal.io/sdk/workflow"
)

// ============================================================================
// METRICS COLLECTION
// ============================================================================

// MetricsCollector tracks workflow metrics and provides query handlers
type MetricsCollector struct {
	mu sync.RWMutex

	// Lock metrics
	lockAcquisitionTimes map[string]time.Duration // Lock name -> acquisition duration
	lockConflicts        map[string]int           // Lock name -> conflict count

	// Gate metrics
	gateDurations   map[string][]time.Duration // Gate name -> durations
	gateSuccessRate map[string]float64         // Gate name -> success rate (0.0-1.0)
	gateAttempts    map[string]int             // Gate name -> total attempts

	// Workflow metrics
	workflowStartTime time.Time
}

// NewMetricsCollector creates a new metrics collector
func NewMetricsCollector() *MetricsCollector {
	return &MetricsCollector{
		lockAcquisitionTimes: make(map[string]time.Duration),
		lockConflicts:        make(map[string]int),
		gateDurations:        make(map[string][]time.Duration),
		gateSuccessRate:      make(map[string]float64),
		gateAttempts:         make(map[string]int),
		workflowStartTime:    time.Now(),
	}
}

// RecordLockMetrics tracks lock acquisition time and conflict count
func RecordLockMetrics(collector *MetricsCollector, lockName string, acquisitionTime time.Duration, conflictOccurred bool) {
	if collector == nil {
		return
	}

	collector.mu.Lock()
	defer collector.mu.Unlock()

	// Record acquisition time
	collector.lockAcquisitionTimes[lockName] = acquisitionTime

	// Increment conflict counter if conflict occurred
	if conflictOccurred {
		collector.lockConflicts[lockName]++
	}
}

// RecordGateMetrics tracks gate duration and success rate
func RecordGateMetrics(collector *MetricsCollector, gateName string, duration time.Duration, success bool) {
	if collector == nil {
		return
	}

	collector.mu.Lock()
	defer collector.mu.Unlock()

	// Record duration
	collector.gateDurations[gateName] = append(collector.gateDurations[gateName], duration)

	// Update success rate
	collector.gateAttempts[gateName]++

	// In practice, you'd track successful attempts separately
	// For now, we calculate based on the success parameter of current call
	if success {
		if rate, exists := collector.gateSuccessRate[gateName]; exists {
			// Update rate with new attempt
			newRate := (rate*float64(collector.gateAttempts[gateName]-1) + 1.0) / float64(collector.gateAttempts[gateName])
			collector.gateSuccessRate[gateName] = newRate
		} else {
			collector.gateSuccessRate[gateName] = 1.0
		}
	} else {
		if rate, exists := collector.gateSuccessRate[gateName]; exists {
			newRate := (rate * float64(collector.gateAttempts[gateName]-1)) / float64(collector.gateAttempts[gateName])
			collector.gateSuccessRate[gateName] = newRate
		} else {
			collector.gateSuccessRate[gateName] = 0.0
		}
	}
}

// GetMetricsSnapshot returns current metrics for external systems
// Formatted for Temporal metrics export
func (m *MetricsCollector) GetMetricsSnapshot() map[string]interface{} {
	m.mu.RLock()
	defer m.mu.RUnlock()

	// Calculate average lock acquisition time
	avgLockAcquisition := make(map[string]float64)
	for lockName, duration := range m.lockAcquisitionTimes {
		avgLockAcquisition[lockName] = float64(duration.Milliseconds())
	}

	// Calculate average gate duration
	avgGateDuration := make(map[string]float64)
	for gateName, durations := range m.gateDurations {
		if len(durations) > 0 {
			total := time.Duration(0)
			for _, d := range durations {
				total += d
			}
			avgGateDuration[gateName] = float64(total.Milliseconds()) / float64(len(durations))
		}
	}

	// Calculate workflow duration
	workflowDuration := time.Since(m.workflowStartTime)

	return map[string]interface{}{
		"lock_acquisition_duration_ms": avgLockAcquisition,
		"lock_conflicts_total":         m.lockConflicts,
		"gate_duration_ms":             avgGateDuration,
		"gate_success_rate":            m.gateSuccessRate,
		"workflow_duration_ms":         float64(workflowDuration.Milliseconds()),
		"snapshot_timestamp":           time.Now(),
	}
}

// ============================================================================
// WORKFLOW QUERIES
// ============================================================================

// QueryWorkflowState returns current workflow state and progress
// This is registered as a query handler for external visibility
func QueryWorkflowState(ctx workflow.Context, state WorkflowState) (interface{}, error) {
	logger := workflow.GetLogger(ctx)
	logger.Debug("Query: WorkflowState requested")

	return map[string]interface{}{
		"current_state": state,
		"timestamp":     time.Now(),
	}, nil
}

// GetWorkflowProgress returns gate completion status and overall progress
// Used to determine how far along the workflow is
func GetWorkflowProgress(gateResults map[string]GateResult) map[string]interface{} {
	completedGates := 0
	totalGates := len(gateResults)

	successfulGates := 0
	for _, result := range gateResults {
		if result.Passed {
			successfulGates++
		}
		completedGates++
	}

	progressPercent := 0.0
	if totalGates > 0 {
		progressPercent = (float64(completedGates) / float64(totalGates)) * 100.0
	}

	return map[string]interface{}{
		"total_gates":      totalGates,
		"completed_gates":  completedGates,
		"successful_gates": successfulGates,
		"progress_percent": progressPercent,
	}
}
