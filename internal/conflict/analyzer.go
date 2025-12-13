// Copyright (c) 2025 Open Swarm Contributors
//
// This software is released under the MIT License.
// See LICENSE file in the repository for details.

package conflict

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"open-swarm/internal/patternmatch"
)

// Reservation represents a file reservation from Agent Mail
type Reservation struct {
	ID        int
	AgentName string
	Pattern   string
	Exclusive bool
	ExpiresAt time.Time
}

// Conflict represents a detected conflict between reservations
type Conflict struct {
	Requestor        string
	RequestedPattern string
	Holders          []Holder
	ConflictType     Type
}

// Holder represents an agent holding a conflicting reservation
type Holder struct {
	AgentName     string
	Pattern       string
	Exclusive     bool
	ExpiresAt     time.Time
	ReservationID int
}

// Type categorizes the type of conflict
type Type string

const (
	// TypeExclusiveExclusive represents a conflict where both want exclusive access
	TypeExclusiveExclusive Type = "exclusive-exclusive"
	// TypeExclusiveShared represents a conflict between exclusive and shared requests
	TypeExclusiveShared Type = "exclusive-shared"
	// TypeExpiring represents a conflict but reservation expiring soon
	TypeExpiring Type = "expiring"
)

// Resolution suggests how to resolve a conflict
type Resolution string

const (
	// ResolutionWait indicates to wait for reservation to expire
	ResolutionWait Resolution = "wait"
	// ResolutionNegotiate indicates to contact holder to negotiate
	ResolutionNegotiate Resolution = "negotiate"
	// ResolutionForceRelease indicates to force release stale reservation
	ResolutionForceRelease Resolution = "force-release"
	// ResolutionChangePattern indicates to use different file pattern
	ResolutionChangePattern Resolution = "change-pattern"
	// reservationExpirationThreshold is the time window to consider reservations as expiring soon
	reservationExpirationThreshold = 5 * time.Minute
)

// Analyzer detects conflicts in file reservations
type Analyzer struct {
	projectKey string
}

// NewAnalyzer creates a new conflict analyzer
func NewAnalyzer(projectKey string) *Analyzer {
	return &Analyzer{
		projectKey: projectKey,
	}
}

// CheckConflict checks if a requested pattern conflicts with existing reservations
func (a *Analyzer) CheckConflict(ctx context.Context, agentName string, pattern string, exclusive bool, reservations []Reservation) (*Conflict, error) {
	slog.InfoContext(ctx, "Checking for conflicts",
		"agent", agentName,
		"pattern", pattern,
		"exclusive", exclusive,
		"total_reservations", len(reservations))

	var holders []Holder

	for _, res := range reservations {
		// Skip own reservations
		if res.AgentName == agentName {
			continue
		}

		// Check if patterns overlap
		if patternsOverlap(pattern, res.Pattern) {
			slog.DebugContext(ctx, "Pattern overlap detected",
				"requestor", agentName,
				"holder", res.AgentName,
				"requested_pattern", pattern,
				"held_pattern", res.Pattern)

			// Conflict if either is exclusive
			if exclusive || res.Exclusive {
				slog.WarnContext(ctx, "Exclusive conflict detected",
					"requestor", agentName,
					"holder", res.AgentName,
					"requestor_exclusive", exclusive,
					"holder_exclusive", res.Exclusive,
					"expires_at", res.ExpiresAt)

				holders = append(holders, Holder{
					AgentName:     res.AgentName,
					Pattern:       res.Pattern,
					Exclusive:     res.Exclusive,
					ExpiresAt:     res.ExpiresAt,
					ReservationID: res.ID,
				})
			}
		}
	}

	if len(holders) == 0 {
		slog.InfoContext(ctx, "No conflicts found",
			"agent", agentName,
			"pattern", pattern)
		//nolint:nilnil // Return nil conflict and nil error to indicate no conflict found
		return nil, nil
	}

	// Determine conflict type
	conflictType := a.determineConflictType(exclusive, holders)

	slog.ErrorContext(ctx, "CONFLICT DETECTED",
		"requestor", agentName,
		"requested_pattern", pattern,
		"conflict_type", conflictType,
		"num_conflicts", len(holders))

	return &Conflict{
		Requestor:        agentName,
		RequestedPattern: pattern,
		Holders:          holders,
		ConflictType:     conflictType,
	}, nil
}

// SuggestResolution suggests the best resolution strategy
func (a *Analyzer) SuggestResolution(conflict *Conflict) Resolution {
	if conflict == nil {
		return ""
	}

	slog.Info("Analyzing conflict resolution options",
		"requestor", conflict.Requestor,
		"pattern", conflict.RequestedPattern,
		"num_holders", len(conflict.Holders))

	// Check if any reservations expire soon (within reservationExpirationThreshold)
	threshold := time.Now().Add(reservationExpirationThreshold)
	allExpiringSoon := true
	for _, holder := range conflict.Holders {
		if holder.ExpiresAt.After(threshold) {
			allExpiringSoon = false
			break
		}
	}

	if allExpiringSoon {
		slog.Info("Resolution: WAIT - all reservations expire soon",
			"requestor", conflict.Requestor,
			"strategy", "wait",
			"reason", "all reservations expire within 5 minutes")
		return ResolutionWait
	}

	// Check if reservation is stale (expired but not released)
	now := time.Now()
	hasStale := false
	staleAgents := []string{}
	for _, holder := range conflict.Holders {
		if holder.ExpiresAt.Before(now) {
			hasStale = true
			staleAgents = append(staleAgents, holder.AgentName)
		}
	}

	if hasStale {
		slog.Warn("Resolution: FORCE RELEASE - stale reservations detected",
			"requestor", conflict.Requestor,
			"strategy", "force-release",
			"stale_agents", staleAgents,
			"reason", "reservations have expired but not released")
		return ResolutionForceRelease
	}

	// Default: negotiate with holders
	holderNames := make([]string, len(conflict.Holders))
	for i, holder := range conflict.Holders {
		holderNames[i] = holder.AgentName
	}
	slog.Info("Resolution: NEGOTIATE - contact holders via Agent Mail",
		"requestor", conflict.Requestor,
		"strategy", "negotiate",
		"holders", holderNames,
		"reason", "active reservations require coordination")
	return ResolutionNegotiate
}

// FormatConflictReport generates a human-readable conflict report
func (a *Analyzer) FormatConflictReport(conflict *Conflict, resolution Resolution) string {
	if conflict == nil {
		return "No conflicts detected"
	}

	report := "❌ CONFLICT DETECTED\n\n"
	report += fmt.Sprintf("Requestor: %s\n", conflict.Requestor)
	report += fmt.Sprintf("Requested pattern: %s\n", conflict.RequestedPattern)
	report += fmt.Sprintf("Conflict type: %s\n", conflict.ConflictType)
	report += fmt.Sprintf("\nConflicting reservations (%d):\n", len(conflict.Holders))

	for i, holder := range conflict.Holders {
		expiresIn := time.Until(holder.ExpiresAt)
		report += fmt.Sprintf("\n  %d. Agent: %s\n", i+1, holder.AgentName)
		report += fmt.Sprintf("     Pattern: %s\n", holder.Pattern)
		report += fmt.Sprintf("     Exclusive: %t\n", holder.Exclusive)
		report += fmt.Sprintf("     Expires: %v (in %v)\n", holder.ExpiresAt.Format(time.RFC3339), expiresIn)
	}

	report += fmt.Sprintf("\n💡 Suggested resolution: %s\n", resolution)

	switch resolution {
	case ResolutionWait:
		report += "   Wait for reservations to expire (all expire within 5 minutes)\n"
	case ResolutionNegotiate:
		report += "   Contact holders via Agent Mail to coordinate access\n"
	case ResolutionForceRelease:
		report += "   Use force_release_file_reservation for stale reservations\n"
	case ResolutionChangePattern:
		report += "   Modify your file pattern to avoid overlap\n"
	}

	return report
}

// determineConflictType categorizes the conflict
func (a *Analyzer) determineConflictType(requestExclusive bool, holders []Holder) Type {
	// Check if all expire soon
	threshold := time.Now().Add(reservationExpirationThreshold)
	allExpiringSoon := true
	for _, holder := range holders {
		if holder.ExpiresAt.After(threshold) {
			allExpiringSoon = false
			break
		}
	}

	if allExpiringSoon {
		return TypeExpiring
	}

	// Check holder types
	hasExclusive := false
	for _, holder := range holders {
		if holder.Exclusive {
			hasExclusive = true
			break
		}
	}

	if requestExclusive && hasExclusive {
		return TypeExclusiveExclusive
	}

	return TypeExclusiveShared
}

// patternsOverlap checks if two file patterns overlap using glob matching
// This implements symmetric fnmatchcase: either pattern can match the other
func patternsOverlap(pattern1, pattern2 string) bool {
	return patternmatch.Overlap(pattern1, pattern2)
}
