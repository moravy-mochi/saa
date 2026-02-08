package main

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"syscall"
	"time"
)

type BashResult struct {
	Stdout   string
	Stderr   string
	ExitCode int
}

func ExecuteBash(command string, workDir string, timeout time.Duration) (BashResult, error) {
	absWorkDir, err := filepath.Abs(workDir)
	if err != nil {
		return BashResult{}, err
	}

	tmpFile, err := os.CreateTemp("", "saa_script_*.sh")
	if err != nil {
		return BashResult{}, err
	}
	defer os.Remove(tmpFile.Name())

	_, err = tmpFile.WriteString("#!/bin/bash\nset -euo pipefail\n" + command)
	if err != nil {
		return BashResult{}, err
	}
	tmpFile.Close()

	if err := os.Chmod(tmpFile.Name(), 0700); err != nil {
		return BashResult{}, err
	}

	ctx := context.Background()
	if timeout > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, timeout)
		defer cancel()
	}

	cmd := exec.CommandContext(ctx, "/bin/bash", tmpFile.Name())
	cmd.Dir = absWorkDir
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
	cmd.Cancel = func() error {
		return syscall.Kill(-cmd.Process.Pid, syscall.SIGKILL)
	}

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err = cmd.Run()
	exitCode := 0
	if err != nil {
		if ctx.Err() == context.DeadlineExceeded {
			return BashResult{
				Stderr:   fmt.Sprintf("Error: Command timed out after %v.", timeout),
				ExitCode: -1,
			}, nil
		}

		if exitError, ok := err.(*exec.ExitError); ok {
			exitCode = exitError.ExitCode()
		} else {
			return BashResult{}, err
		}
	}

	stdoutStr := stdout.String()
	stderrStr := stderr.String()

	return BashResult{
		Stdout:   stdoutStr,
		Stderr:   stderrStr,
		ExitCode: exitCode,
	}, nil
}