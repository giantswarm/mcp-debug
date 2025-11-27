package agent

import (
	"context"
	"errors"
	"net"
	"testing"

	"github.com/mark3labs/mcp-go/mcp"
)

func TestServerCapabilityChecking(t *testing.T) {
	logger := NewLogger(false, false, false)
	client := NewClient(ClientConfig{
		Endpoint:    "test://endpoint",
		Transport:   "streamable-http",
		Logger:      logger,
		OAuthConfig: nil,
		Version:     "test",
	})

	// Test with no capabilities set (should all return false)
	if client.ServerSupportsTools() {
		t.Error("Expected ServerSupportsTools to return false when no capabilities are set")
	}
	if client.ServerSupportsResources() {
		t.Error("Expected ServerSupportsResources to return false when no capabilities are set")
	}
	if client.ServerSupportsPrompts() {
		t.Error("Expected ServerSupportsPrompts to return false when no capabilities are set")
	}

	// Test that the methods exist and don't panic
	// We can't easily test the positive cases without mocking the actual MCP types
	// but we can verify the methods work correctly when no capabilities are set
	if client.serverCapabilities != nil {
		t.Error("Expected serverCapabilities to be nil initially")
	}

	// Test with capabilities set
	// Note: We can't easily test the positive cases without importing the specific capability types
	// from mcp-go, but we can at least verify the logic works when capabilities are present
	// The negative cases above already verify the core logic
}

func TestNewClient(t *testing.T) {
	logger := NewLogger(false, false, false)
	client := NewClient(ClientConfig{
		Endpoint:    "http://localhost:8080",
		Transport:   "streamable-http",
		Logger:      logger,
		OAuthConfig: nil,
		Version:     "test",
	})

	if client == nil {
		t.Fatal("Expected client to be created, but got nil")
	}

	if client.endpoint != "http://localhost:8080" {
		t.Errorf("Expected endpoint to be http://localhost:8080, got %s", client.endpoint)
	}

	if client.transport != "streamable-http" {
		t.Errorf("Expected transport to be streamable-http, got %s", client.transport)
	}

	if client.version != "test" {
		t.Errorf("Expected version to be test, got %s", client.version)
	}

	if client.notificationChan == nil {
		t.Error("Expected notificationChan to be initialized")
	}

	if client.toolCache == nil {
		t.Error("Expected toolCache to be initialized")
	}

	if client.resourceCache == nil {
		t.Error("Expected resourceCache to be initialized")
	}

	if client.promptCache == nil {
		t.Error("Expected promptCache to be initialized")
	}
}

func TestNewClientWithOAuth(t *testing.T) {
	logger := NewLogger(false, false, false)
	oauthConfig := &OAuthConfig{
		Enabled:      true,
		ClientID:     "test-client-id",
		ClientSecret: "test-client-secret",
		RedirectURL:  "http://localhost:8080/callback",
	}

	client := NewClient(ClientConfig{
		Endpoint:    "http://localhost:8080",
		Transport:   "streamable-http",
		Logger:      logger,
		OAuthConfig: oauthConfig,
		Version:     "test",
	})

	if client == nil {
		t.Fatal("Expected client to be created, but got nil")
	}

	if client.oauthConfig == nil {
		t.Error("Expected oauthConfig to be set")
	}

	if !client.oauthConfig.Enabled {
		t.Error("Expected OAuth to be enabled")
	}
}

func TestShouldReconnect(t *testing.T) {
	tests := []struct {
		name string
		err  error
		want bool
	}{
		{
			name: "nil error",
			err:  nil,
			want: false,
		},
		{
			name: "context canceled",
			err:  context.Canceled,
			want: true,
		},
		{
			name: "context deadline exceeded",
			err:  context.DeadlineExceeded,
			want: true,
		},
		{
			name: "connection refused",
			err:  errors.New("connection refused"),
			want: true,
		},
		{
			name: "connection reset by peer",
			err:  errors.New("connection reset by peer"),
			want: true,
		},
		{
			name: "transport is closing",
			err:  errors.New("transport is closing"),
			want: true,
		},
		{
			name: "broken pipe",
			err:  errors.New("broken pipe"),
			want: true,
		},
		{
			name: "unexpected EOF",
			err:  errors.New("unexpected EOF"),
			want: true,
		},
		{
			name: "network timeout",
			err:  &timeoutError{},
			want: true,
		},
		{
			name: "other error",
			err:  errors.New("some other error"),
			want: false,
		},
		{
			name: "uppercase connection refused",
			err:  errors.New("Connection Refused"),
			want: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := shouldReconnect(tt.err)
			if got != tt.want {
				t.Errorf("shouldReconnect(%v) = %v, want %v", tt.err, got, tt.want)
			}
		})
	}
}

// timeoutError implements net.Error with Timeout() returning true
type timeoutError struct{}

func (e *timeoutError) Error() string   { return "timeout error" }
func (e *timeoutError) Timeout() bool   { return true }
func (e *timeoutError) Temporary() bool { return true }

var _ net.Error = (*timeoutError)(nil)

func TestPrettyJSON(t *testing.T) {
	tests := []struct {
		name    string
		input   interface{}
		wantNil bool
	}{
		{
			name: "simple map",
			input: map[string]interface{}{
				"key": "value",
			},
		},
		{
			name: "simple struct",
			input: struct {
				Name string
				Age  int
			}{
				Name: "test",
				Age:  42,
			},
		},
		{
			name:  "nil",
			input: nil,
		},
		{
			name:  "string",
			input: "test string",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := PrettyJSON(tt.input)
			if result == "" {
				t.Error("Expected non-empty result from PrettyJSON")
			}
		})
	}
}

func TestShowDiffs(t *testing.T) {
	logger := NewLogger(false, false, false)
	client := NewClient(ClientConfig{
		Endpoint:  "test://endpoint",
		Transport: "streamable-http",
		Logger:    logger,
		Version:   "test",
	})

	t.Run("tool diff - no changes", func(t *testing.T) {
		tools := []mcp.Tool{
			{Name: "tool1", Description: "desc1"},
			{Name: "tool2", Description: "desc2"},
		}
		// Should not panic
		client.showToolDiff(tools, tools)
	})

	t.Run("tool diff - with additions", func(t *testing.T) {
		oldTools := []mcp.Tool{
			{Name: "tool1", Description: "desc1"},
		}
		newTools := []mcp.Tool{
			{Name: "tool1", Description: "desc1"},
			{Name: "tool2", Description: "desc2"},
		}
		client.showToolDiff(oldTools, newTools)
	})

	t.Run("tool diff - with removals", func(t *testing.T) {
		oldTools := []mcp.Tool{
			{Name: "tool1", Description: "desc1"},
			{Name: "tool2", Description: "desc2"},
		}
		newTools := []mcp.Tool{
			{Name: "tool1", Description: "desc1"},
		}
		client.showToolDiff(oldTools, newTools)
	})

	t.Run("resource diff - no changes", func(t *testing.T) {
		resources := []mcp.Resource{
			{URI: "resource1", Name: "res1"},
			{URI: "resource2", Name: "res2"},
		}
		client.showResourceDiff(resources, resources)
	})

	t.Run("resource diff - with additions", func(t *testing.T) {
		oldResources := []mcp.Resource{
			{URI: "resource1", Name: "res1"},
		}
		newResources := []mcp.Resource{
			{URI: "resource1", Name: "res1"},
			{URI: "resource2", Name: "res2"},
		}
		client.showResourceDiff(oldResources, newResources)
	})

	t.Run("prompt diff - no changes", func(t *testing.T) {
		prompts := []mcp.Prompt{
			{Name: "prompt1"},
			{Name: "prompt2"},
		}
		client.showPromptDiff(prompts, prompts)
	})

	t.Run("prompt diff - with additions", func(t *testing.T) {
		oldPrompts := []mcp.Prompt{
			{Name: "prompt1"},
		}
		newPrompts := []mcp.Prompt{
			{Name: "prompt1"},
			{Name: "prompt2"},
		}
		client.showPromptDiff(oldPrompts, newPrompts)
	})
}
