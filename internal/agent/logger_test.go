package agent

import (
	"bytes"
	"strings"
	"testing"
)

func TestInfoVerbose(t *testing.T) {
	tests := []struct {
		name           string
		verbose        bool
		format         string
		args           []interface{}
		expectOutput   bool
		expectedSubstr string
	}{
		{
			name:           "verbose enabled - should output",
			verbose:        true,
			format:         "test message: %s",
			args:           []interface{}{"hello"},
			expectOutput:   true,
			expectedSubstr: "test message: hello",
		},
		{
			name:         "verbose disabled - should not output",
			verbose:      false,
			format:       "test message: %s",
			args:         []interface{}{"hello"},
			expectOutput: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			buf := &bytes.Buffer{}
			logger := NewLoggerWithWriter(tt.verbose, false, false, buf)

			logger.InfoVerbose(tt.format, tt.args...)

			output := buf.String()
			if tt.expectOutput {
				if !strings.Contains(output, tt.expectedSubstr) {
					t.Errorf("expected output to contain %q, got %q", tt.expectedSubstr, output)
				}
			} else {
				if output != "" {
					t.Errorf("expected no output, got %q", output)
				}
			}
		})
	}
}

func TestWarningVerbose(t *testing.T) {
	tests := []struct {
		name           string
		verbose        bool
		format         string
		args           []interface{}
		expectOutput   bool
		expectedSubstr string
	}{
		{
			name:           "verbose enabled - should output",
			verbose:        true,
			format:         "warning: %s",
			args:           []interface{}{"test warning"},
			expectOutput:   true,
			expectedSubstr: "warning: test warning",
		},
		{
			name:         "verbose disabled - should not output",
			verbose:      false,
			format:       "warning: %s",
			args:         []interface{}{"test warning"},
			expectOutput: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			buf := &bytes.Buffer{}
			logger := NewLoggerWithWriter(tt.verbose, false, false, buf)

			logger.WarningVerbose(tt.format, tt.args...)

			output := buf.String()
			if tt.expectOutput {
				if !strings.Contains(output, tt.expectedSubstr) {
					t.Errorf("expected output to contain %q, got %q", tt.expectedSubstr, output)
				}
			} else {
				if output != "" {
					t.Errorf("expected no output, got %q", output)
				}
			}
		})
	}
}

func TestInfoVerboseNilLogger(t *testing.T) {
	// Should not panic with nil logger
	var logger *Logger
	logger.InfoVerbose("test message")
	// If we reach here, test passes (no panic)
}

func TestWarningVerboseNilLogger(t *testing.T) {
	// Should not panic with nil logger
	var logger *Logger
	logger.WarningVerbose("test warning")
	// If we reach here, test passes (no panic)
}

func TestLoggerBasicFunctions(t *testing.T) {
	buf := &bytes.Buffer{}
	logger := NewLoggerWithWriter(false, false, false, buf)

	t.Run("Info", func(t *testing.T) {
		buf.Reset()
		logger.Info("info message")
		if !strings.Contains(buf.String(), "info message") {
			t.Errorf("expected Info to log message, got %q", buf.String())
		}
	})

	t.Run("Error", func(t *testing.T) {
		buf.Reset()
		logger.Error("error message")
		if !strings.Contains(buf.String(), "error message") {
			t.Errorf("expected Error to log message, got %q", buf.String())
		}
	})

	t.Run("Success", func(t *testing.T) {
		buf.Reset()
		logger.Success("success message")
		if !strings.Contains(buf.String(), "success message") {
			t.Errorf("expected Success to log message, got %q", buf.String())
		}
	})

	t.Run("Warning", func(t *testing.T) {
		buf.Reset()
		logger.Warning("warning message")
		if !strings.Contains(buf.String(), "warning message") {
			t.Errorf("expected Warning to log message, got %q", buf.String())
		}
	})

	t.Run("Debug verbose enabled", func(t *testing.T) {
		buf.Reset()
		logger.SetVerbose(true)
		logger.Debug("debug message")
		if !strings.Contains(buf.String(), "debug message") {
			t.Errorf("expected Debug to log message in verbose mode, got %q", buf.String())
		}
	})

	t.Run("Debug verbose disabled", func(t *testing.T) {
		buf.Reset()
		logger.SetVerbose(false)
		logger.Debug("debug message")
		if buf.String() != "" {
			t.Errorf("expected Debug to not log message when verbose is disabled, got %q", buf.String())
		}
	})
}

func TestLoggerConstructors(t *testing.T) {
	t.Run("NewLogger", func(t *testing.T) {
		logger := NewLogger(true, true, true)
		if logger == nil {
			t.Error("expected NewLogger to return non-nil logger")
		}
		if !logger.verbose {
			t.Error("expected verbose to be true")
		}
		if !logger.useColor {
			t.Error("expected useColor to be true")
		}
		if !logger.jsonRPCMode {
			t.Error("expected jsonRPCMode to be true")
		}
	})

	t.Run("NewLoggerWithWriter", func(t *testing.T) {
		buf := &bytes.Buffer{}
		logger := NewLoggerWithWriter(false, false, false, buf)
		if logger == nil {
			t.Error("expected NewLoggerWithWriter to return non-nil logger")
		}
		if logger.writer != buf {
			t.Error("expected writer to be set to provided buffer")
		}
	})
}

func TestSetWriter(t *testing.T) {
	buf1 := &bytes.Buffer{}
	buf2 := &bytes.Buffer{}

	logger := NewLoggerWithWriter(false, false, false, buf1)
	logger.Info("message1")

	if !strings.Contains(buf1.String(), "message1") {
		t.Error("expected message to be written to buf1")
	}

	buf1.Reset()
	logger.SetWriter(buf2)
	logger.Info("message2")

	if buf1.String() != "" {
		t.Error("expected buf1 to be empty after changing writer")
	}

	if !strings.Contains(buf2.String(), "message2") {
		t.Error("expected message to be written to buf2")
	}
}
