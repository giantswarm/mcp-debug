package agent

import (
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestCallbackHandlerPages(t *testing.T) {
	tests := []struct {
		name       string
		method     string
		query      string
		wantStatus int
		want       []string
		notWant    []string
	}{
		{
			name:       "signed in",
			method:     http.MethodGet,
			query:      "code=c&state=s",
			wantStatus: http.StatusOK,
			want:       []string{"<h1>Signed in</h1>", "mcp-debug is now signed in"},
		},
		{
			name:       "provider refuses",
			method:     http.MethodGet,
			query:      "error=access_denied&error_description=%3Cb%3Eno%3C%2Fb%3E",
			wantStatus: http.StatusBadRequest,
			want:       []string{"<h1>Sign-in failed</h1>", "The sign-in was denied or failed", "<code>access_denied</code>"},
			notWant:    []string{"<b>no</b>"},
		},
		{
			name:       "provider error that is not a registered code",
			method:     http.MethodGet,
			query:      "error=your_account_is_locked_call_555",
			wantStatus: http.StatusBadRequest,
			want:       []string{"The sign-in was denied or failed"},
			notWant:    []string{"your_account_is_locked"},
		},
		{
			name:       "no code",
			method:     http.MethodGet,
			query:      "state=s",
			wantStatus: http.StatusBadRequest,
			want:       []string{"<h1>Sign-in failed</h1>", "The sign-in response is incomplete"},
		},
		{
			name:       "not a GET",
			method:     http.MethodPost,
			query:      "code=c",
			wantStatus: http.StatusMethodNotAllowed,
			want:       []string{"The sign-in callback only accepts GET requests"},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			logger := NewLogger(false, false, false)
			logger.SetWriter(io.Discard)
			rr := httptest.NewRecorder()

			createCallbackHandler(logger, make(chan callbackResult, 1))(rr, httptest.NewRequest(tc.method, "/callback?"+tc.query, nil))

			if rr.Code != tc.wantStatus {
				t.Errorf("status = %d, want %d", rr.Code, tc.wantStatus)
			}
			if !strings.Contains(rr.Header().Get("Content-Security-Policy"), "default-src 'none'") {
				t.Errorf("expected a strict Content-Security-Policy, got %q", rr.Header().Get("Content-Security-Policy"))
			}
			body := rr.Body.String()
			for _, want := range tc.want {
				if !strings.Contains(body, want) {
					t.Errorf("body does not contain %q: %s", want, body)
				}
			}
			for _, notWant := range tc.notWant {
				if strings.Contains(body, notWant) {
					t.Errorf("body contains %q", notWant)
				}
			}
		})
	}
}
