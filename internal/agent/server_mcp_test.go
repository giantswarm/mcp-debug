package agent

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// A body beyond the bound is refused, and the endpoint keeps answering.
func TestMCPBodyIsBounded(t *testing.T) {
	ms, err := NewMCPServer(nil, "streamable-http", nil, false)
	if err != nil {
		t.Fatal(err)
	}
	ts := httptest.NewServer(ms.httpHandler())
	defer ts.Close()
	post := func(body string) int {
		req, err := http.NewRequest(http.MethodPost, ts.URL+"/mcp", strings.NewReader(body))
		if err != nil {
			t.Fatal(err)
		}
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Accept", "application/json, text/event-stream")
		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			t.Fatal(err)
		}
		_ = resp.Body.Close()
		return resp.StatusCode
	}

	big := `{"jsonrpc":"2.0","id":1,"method":"ping","params":{"pad":"` + strings.Repeat("x", maxBodyBytes) + `"}}`
	if got := post(big); got < 400 {
		t.Errorf("oversized body: status %d, want an error", got)
	}
	if got := post(`{"jsonrpc":"2.0","id":2,"method":"initialize","params":{"protocolVersion":"2025-06-18","capabilities":{},"clientInfo":{"name":"t","version":"0"}}}`); got != http.StatusOK {
		t.Errorf("initialize: status %d, want 200", got)
	}
}
