// Copyright (c) 2025 Open Swarm Contributors
//
// This software is released under the MIT License.
// See LICENSE file in the repository for details.

package conflict

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewAnalyzer(t *testing.T) {
	analyzer := NewAnalyzer("/test/project")
	assert.NotNil(t, analyzer)
	assert.Equal(t, "/test/project", analyzer.projectKey)
}

func TestPatternsOverlap(t *testing.T) {
	tests := []struct {
		name     string
		pattern1 string
		pattern2 string
		want     bool
	}{
		{"exact match", "foo.go", "foo.go", true},
		{"glob matches file", "*.go", "foo.go", true},
		{"file matches glob (symmetric)", "foo.go", "*.go", true},
		{"different files", "foo.go", "bar.go", false},
		{"nested glob", "pkg/**/*.go", "pkg/foo/bar.go", true}, // ** expands to match // ** not supported by filepath.Match
		{"partial match", "pkg/*.go", "pkg/foo.go", true},
		{"no overlap", "pkg/*.go", "cmd/*.go", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := patternsOverlap(tt.pattern1, tt.pattern2)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestCheckConflict_NoConflict(t *testing.T) {
	analyzer := NewAnalyzer("/test/project")

	reservations := []Reservation{
		{ID: 1, AgentName: "AgentA", Pattern: "pkg/*.go", Exclusive: true, ExpiresAt: time.Now().Add(1 * time.Hour)},
	}

	conflict, err := analyzer.CheckConflict(context.Background(), "AgentB", "cmd/*.go", true, reservations)
	require.NoError(t, err)
	assert.Nil(t, conflict)
}

func TestCheckConflict_ExclusiveConflict(t *testing.T) {
	analyzer := NewAnalyzer("/test/project")

	now := time.Now()
	reservations := []Reservation{
		{ID: 1, AgentName: "AgentA", Pattern: "*.go", Exclusive: true, ExpiresAt: now.Add(1 * time.Hour)},
	}

	conflict, err := analyzer.CheckConflict(context.Background(), "AgentB", "foo.go", true, reservations)
	require.NoError(t, err)
	require.NotNil(t, conflict)

	assert.Equal(t, "AgentB", conflict.Requestor)
	assert.Equal(t, "foo.go", conflict.RequestedPattern)
	assert.Len(t, conflict.Holders, 1)
	assert.Equal(t, "AgentA", conflict.Holders[0].AgentName)
	assert.Equal(t, TypeExclusiveExclusive, conflict.ConflictType)
}

func TestCheckConflict_SkipsOwnReservations(t *testing.T) {
	analyzer := NewAnalyzer("/test/project")

	reservations := []Reservation{
		{ID: 1, AgentName: "AgentA", Pattern: "*.go", Exclusive: true, ExpiresAt: time.Now().Add(1 * time.Hour)},
	}

	// Same agent requesting
	conflict, err := analyzer.CheckConflict(context.Background(), "AgentA", "foo.go", true, reservations)
	require.NoError(t, err)
	assert.Nil(t, conflict, "Should not conflict with own reservations")
}

func TestCheckConflict_SharedDoesNotConflict(t *testing.T) {
	analyzer := NewAnalyzer("/test/project")

	reservations := []Reservation{
		{ID: 1, AgentName: "AgentA", Pattern: "*.go", Exclusive: false, ExpiresAt: time.Now().Add(1 * time.Hour)},
	}

	// Shared request on shared reservation
	conflict, err := analyzer.CheckConflict(context.Background(), "AgentB", "foo.go", false, reservations)
	require.NoError(t, err)
	assert.Nil(t, conflict, "Shared reservations should not conflict")
}

func TestCheckConflict_ExclusiveVsShared(t *testing.T) {
	analyzer := NewAnalyzer("/test/project")

	reservations := []Reservation{
		{ID: 1, AgentName: "AgentA", Pattern: "*.go", Exclusive: false, ExpiresAt: time.Now().Add(1 * time.Hour)},
	}

	// Exclusive request conflicts with shared
	conflict, err := analyzer.CheckConflict(context.Background(), "AgentB", "foo.go", true, reservations)
	require.NoError(t, err)
	require.NotNil(t, conflict)
	assert.Equal(t, TypeExclusiveShared, conflict.ConflictType)
}

func TestSuggestResolution_Wait(t *testing.T) {
	analyzer := NewAnalyzer("/test/project")

	// All expire within 5 minutes
	conflict := &Conflict{
		Holders: []Holder{
			{ExpiresAt: time.Now().Add(2 * time.Minute)},
			{ExpiresAt: time.Now().Add(4 * time.Minute)},
		},
	}

	resolution := analyzer.SuggestResolution(conflict)
	assert.Equal(t, ResolutionWait, resolution)
}

func TestSuggestResolution_ForceRelease(t *testing.T) {
	analyzer := NewAnalyzer("/test/project")

	// Has stale reservation
	conflict := &Conflict{
		Holders: []Holder{
			{ExpiresAt: time.Now().Add(-5 * time.Minute)}, // Already expired
			{ExpiresAt: time.Now().Add(1 * time.Hour)},
		},
	}

	resolution := analyzer.SuggestResolution(conflict)
	assert.Equal(t, ResolutionForceRelease, resolution)
}

func TestSuggestResolution_Negotiate(t *testing.T) {
	analyzer := NewAnalyzer("/test/project")

	// Normal conflict - not expiring soon, not stale
	conflict := &Conflict{
		Holders: []Holder{
			{ExpiresAt: time.Now().Add(1 * time.Hour)},
		},
	}

	resolution := analyzer.SuggestResolution(conflict)
	assert.Equal(t, ResolutionNegotiate, resolution)
}

func TestSuggestResolution_NilConflict(t *testing.T) {
	analyzer := NewAnalyzer("/test/project")
	resolution := analyzer.SuggestResolution(nil)
	assert.Empty(t, resolution)
}

func TestFormatConflictReport_NoConflict(t *testing.T) {
	analyzer := NewAnalyzer("/test/project")
	report := analyzer.FormatConflictReport(nil, "")
	assert.Contains(t, report, "No conflicts detected")
}

func TestFormatConflictReport_WithConflict(t *testing.T) {
	analyzer := NewAnalyzer("/test/project")

	conflict := &Conflict{
		Requestor:        "AgentB",
		RequestedPattern: "foo.go",
		ConflictType:     TypeExclusiveExclusive,
		Holders: []Holder{
			{
				AgentName:     "AgentA",
				Pattern:       "*.go",
				Exclusive:     true,
				ExpiresAt:     time.Now().Add(1 * time.Hour),
				ReservationID: 123,
			},
		},
	}

	report := analyzer.FormatConflictReport(conflict, ResolutionNegotiate)

	assert.Contains(t, report, "CONFLICT DETECTED")
	assert.Contains(t, report, "AgentB")
	assert.Contains(t, report, "foo.go")
	assert.Contains(t, report, "AgentA")
	assert.Contains(t, report, "*.go")
	assert.Contains(t, report, "negotiate")
}

func TestConflictType_Expiring(t *testing.T) {
	analyzer := NewAnalyzer("/test/project")

	holders := []Holder{
		{ExpiresAt: time.Now().Add(2 * time.Minute)},
	}

	conflictType := analyzer.determineConflictType(true, holders)
	assert.Equal(t, TypeExpiring, conflictType)
}

func TestConflictType_ExclusiveExclusive(t *testing.T) {
	analyzer := NewAnalyzer("/test/project")

	holders := []Holder{
		{Exclusive: true, ExpiresAt: time.Now().Add(1 * time.Hour)},
	}

	conflictType := analyzer.determineConflictType(true, holders)
	assert.Equal(t, TypeExclusiveExclusive, conflictType)
}

func TestConflictType_ExclusiveShared(t *testing.T) {
	analyzer := NewAnalyzer("/test/project")

	holders := []Holder{
		{Exclusive: false, ExpiresAt: time.Now().Add(1 * time.Hour)},
	}

	conflictType := analyzer.determineConflictType(true, holders)
	assert.Equal(t, TypeExclusiveShared, conflictType)
}
