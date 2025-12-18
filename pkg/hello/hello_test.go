package hello

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestHelloWorld(t *testing.T) {
	tests := []struct {
		name string
		want string
	}{
		{
			name: "returns hello world message",
			want: "Hello, World!",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := HelloWorld()
			assert.Equal(t, tt.want, got)
		})
	}
}
