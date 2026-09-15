package agent

import (
	"errors"
	"fmt"
	"io"
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
