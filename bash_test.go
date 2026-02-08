package main

import (
	"strings"
	"testing"
	"time"
)

func TestExecuteBash(t *testing.T) {
	tests := []struct {
		name           string
		command        string
		timeout        time.Duration
		expectedStdout string
		expectedStderr string
		expectedExit   int
		expectError    bool
	}{
		{
			name:           "Simple Echo",
			command:        "echo 'Hello, World!'",
			expectedStdout: "Hello, World!\n",
			expectedExit:   0,
		},
		{
			name:           "Stderr Output",
			command:        "echo 'Error Message' >&2",
			expectedStderr: "Error Message\n",
			expectedExit:   0,
		},
		{
			name:         "Exit Code 1",
			command:      "exit 1",
			expectedExit: 1,
		},
		{
			name:        "Timeout",
			command:     "sleep 2",
			timeout:     100 * time.Millisecond,
			expectError: false, // The function handles timeout and returns a result with error message in Stderr
			expectedStderr: "Error: Command timed out",
			expectedExit:   -1,
		},
		{
			name:           "Complex Script",
			command:        "var=test; echo $var",
			expectedStdout: "test\n",
			expectedExit:   0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := ExecuteBash(tt.command, ".", tt.timeout)

			if tt.expectError {
				if err == nil {
					t.Errorf("expected error but got none")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if result.ExitCode != tt.expectedExit {
				t.Errorf("expected exit code %d, got %d", tt.expectedExit, result.ExitCode)
			}

			if tt.expectedStdout != "" && result.Stdout != tt.expectedStdout {
				t.Errorf("expected stdout %q, got %q", tt.expectedStdout, result.Stdout)
			}

			if tt.expectedStderr != "" && !strings.Contains(result.Stderr, tt.expectedStderr) {
				t.Errorf("expected stderr to contain %q, got %q", tt.expectedStderr, result.Stderr)
			}
		})
	}
}