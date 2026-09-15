package agent

import (
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/chzyer/readline"
)

func TestClassifyReadline(t *testing.T) {
	readErr := errors.New("terminal closed")

	tests := []struct {
		name       string
		err        error
		wantAction replAction
		wantFatal  error
	}{
		{
			name:       "no error executes the line",
			err:        nil,
			wantAction: replExecute,
		},
		{
			name:       "interrupt discards the line",
			err:        readline.ErrInterrupt,
			wantAction: replSkip,
		},
		{
			name:       "wrapped interrupt discards the line",
			err:        fmt.Errorf("read: %w", readline.ErrInterrupt),
			wantAction: replSkip,
		},
		{
			name:       "EOF quits",
			err:        io.EOF,
			wantAction: replQuit,
		},
		{
			name:       "any other error is fatal",
			err:        readErr,
			wantAction: replSkip,
			wantFatal:  readErr,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			action, fatal := classifyReadline(tt.err)

			if action != tt.wantAction {
				t.Errorf("action = %v, want %v", action, tt.wantAction)
			}
			if tt.wantFatal == nil {
				if fatal != nil {
					t.Errorf("fatal = %v, want nil", fatal)
				}
				return
			}
			if !errors.Is(fatal, tt.wantFatal) {
				t.Errorf("fatal = %v, want it to wrap %v", fatal, tt.wantFatal)
			}
		})
	}
}

func TestHistoryFilePath(t *testing.T) {
	t.Setenv("XDG_CACHE_HOME", t.TempDir())

	cache, err := os.UserCacheDir()
	if err != nil {
		t.Skipf("no user cache directory on this host: %v", err)
	}

	got := historyFilePath()

	want := filepath.Join(cache, "mcp-debug", historyFileName)
	if got != want {
		t.Errorf("historyFilePath() = %q, want %q", got, want)
	}
	info, err := os.Stat(filepath.Dir(got))
	if err != nil {
		t.Fatalf("history directory was not created: %v", err)
	}
	if runtime.GOOS != "windows" && info.Mode().Perm() != 0o700 {
		t.Errorf("history directory mode = %v, want 0700", info.Mode().Perm())
	}
}
