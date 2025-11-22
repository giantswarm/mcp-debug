package agent

import (
	"testing"
)

func TestServerCapabilityChecking(t *testing.T) {
	logger := NewLogger(false, false, false)
	client := NewClient("test://endpoint", "streamable-http", logger, nil)

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
}

func TestNewClient(t *testing.T) {
	logger := NewLogger(false, false, false)
	client := NewClient("http://localhost:8080", "streamable-http", logger, nil)

	if client == nil {
		t.Error("Expected client to be created, but got nil")
	}
}
