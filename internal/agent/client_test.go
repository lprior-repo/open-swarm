// Copyright (c) 2025 Open Swarm Contributors
//
// This software is released under the MIT License.
// See LICENSE file in the repository for details.

package agent

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNewClient(t *testing.T) {
	baseURL := "http://localhost:8080"
	port := 8080

	client := NewClient(baseURL, port)

	assert.NotNil(t, client)
	assert.Equal(t, baseURL, client.GetBaseURL())
	assert.Equal(t, port, client.GetPort())
	assert.NotNil(t, client.GetSDK())
}

func TestClient_GetBaseURL(t *testing.T) {
	tests := []struct {
		name    string
		baseURL string
		port    int
	}{
		{"localhost with standard port", "http://localhost:8080", 8080},
		{"localhost with custom port", "http://localhost:9000", 9000},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client := NewClient(tt.baseURL, tt.port)
			assert.Equal(t, tt.baseURL, client.GetBaseURL())
		})
	}
}

func TestClient_GetPort(t *testing.T) {
	client := NewClient("http://localhost:8080", 8080)
	assert.Equal(t, 8080, client.GetPort())
}

func TestClient_ImplementsInterface(_ *testing.T) {
	client := NewClient("http://localhost:8080", 8080)
	var _ ClientInterface = client
}
