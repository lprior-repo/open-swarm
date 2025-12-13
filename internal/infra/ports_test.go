// Copyright (c) 2025 Open Swarm Contributors
//
// This software is released under the MIT License.
// See LICENSE file in the repository for details.

package infra

import (
	"testing"
)

func TestPortManager_Allocate(t *testing.T) {
	pm := NewPortManager(8000, 8010)

	// Test basic allocation
	port1, err := pm.Allocate()
	if err != nil {
		t.Fatalf("Failed to allocate port: %v", err)
	}
	if port1 < 8000 || port1 > 8010 {
		t.Errorf("Allocated port %d outside range 8000-8010", port1)
	}

	// Test second allocation is different
	port2, err := pm.Allocate()
	if err != nil {
		t.Fatalf("Failed to allocate second port: %v", err)
	}
	if port1 == port2 {
		t.Errorf("Allocated same port twice: %d", port1)
	}

	// Test allocation count
	if count := pm.AllocatedCount(); count != 2 {
		t.Errorf("Expected 2 allocated ports, got %d", count)
	}
}

func TestPortManager_Release(t *testing.T) {
	pm := NewPortManager(8000, 8010)

	port, err := pm.Allocate()
	if err != nil {
		t.Fatalf("Failed to allocate port: %v", err)
	}

	// Test release
	err = pm.Release(port)
	if err != nil {
		t.Errorf("Failed to release port %d: %v", port, err)
	}

	// Test double release fails
	err = pm.Release(port)
	if err == nil {
		t.Error("Expected error on double release, got nil")
	}

	// Verify port can be reallocated
	port2, err := pm.Allocate()
	if err != nil {
		t.Fatalf("Failed to reallocate port: %v", err)
	}
	if port2 != port {
		t.Logf("Note: Reallocated port %d instead of released port %d (acceptable)", port2, port)
	}
}

func TestPortManager_Exhaustion(t *testing.T) {
	pm := NewPortManager(8000, 8002) // Only 3 ports

	// Allocate all ports
	ports := make([]int, 0, 3)
	for i := 0; i < 3; i++ {
		port, err := pm.Allocate()
		if err != nil {
			t.Fatalf("Failed to allocate port %d: %v", i, err)
		}
		ports = append(ports, port)
	}

	// Next allocation should fail
	_, err := pm.Allocate()
	if err == nil {
		t.Error("Expected error when all ports exhausted, got nil")
	}

	// Release one and verify we can allocate again
	if err := pm.Release(ports[0]); err != nil {
		t.Fatalf("Failed to release port: %v", err)
	}

	_, err = pm.Allocate()
	if err != nil {
		t.Errorf("Expected successful allocation after release, got error: %v", err)
	}
}

func TestPortManager_IsAllocated(t *testing.T) {
	pm := NewPortManager(8000, 8010)

	port, _ := pm.Allocate()

	if !pm.IsAllocated(port) {
		t.Errorf("Port %d should be allocated", port)
	}

	if pm.IsAllocated(port + 1) {
		t.Errorf("Port %d should not be allocated", port+1)
	}

	_ = pm.Release(port)

	if pm.IsAllocated(port) {
		t.Errorf("Port %d should not be allocated after release", port)
	}
}

func TestPortManager_AvailableCount(t *testing.T) {
	pm := NewPortManager(8000, 8010) // 11 ports total

	if available := pm.AvailableCount(); available != 11 {
		t.Errorf("Expected 11 available ports initially, got %d", available)
	}

	port, _ := pm.Allocate()

	if available := pm.AvailableCount(); available != 10 {
		t.Errorf("Expected 10 available ports after allocation, got %d", available)
	}

	_ = pm.Release(port)

	if available := pm.AvailableCount(); available != 11 {
		t.Errorf("Expected 11 available ports after release, got %d", available)
	}
}
