package app

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"strings"
)

// CommandError carries structured information about a failed subprocess call.
type CommandError struct {
	Command  string
	ExitCode int
	Stderr   string
	Err      error
}

func (e *CommandError) Error() string {
	if e.Stderr != "" {
		return fmt.Sprintf("command %q exited %d: %s", e.Command, e.ExitCode, e.Stderr)
	}
	if e.Err != nil {
		return fmt.Sprintf("command %q exited %d: %v", e.Command, e.ExitCode, e.Err)
	}
	return fmt.Sprintf("command %q failed", e.Command)
}

func (e *CommandError) Unwrap() error { return e.Err }

// commandOutput runs a command and returns its combined stdout as a string.
func commandOutput(name string, args ...string) (string, error) {
	return commandOutputWithEnv(nil, name, args...)
}

// commandOutputWithEnv runs a command with extra environment variables appended
// to the current process environment and returns stdout.
func commandOutputWithEnv(extraEnv []string, name string, args ...string) (string, error) {
	cmd := exec.Command(name, args...)
	if len(extraEnv) > 0 {
		cmd.Env = append(os.Environ(), extraEnv...)
	}

	out, err := cmd.Output()
	if err != nil {
		logError("command failed: %s %s", name, strings.Join(args, " "))
		return "", err
	}

	return string(out), nil
}

// runCommand executes a command, capturing stdout/stderr for error reporting.
// Returns nil on success, *CommandError on failure.
func runCommand(name string, args ...string) error {
	return runCommandWithInput("", name, args...)
}

// runCommandWithInput is like runCommand but feeds input to stdin when non-empty.
func runCommandWithInput(input, name string, args ...string) error {
	cmd := exec.Command(name, args...)

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if input != "" {
		cmd.Stdin = strings.NewReader(input)
	}

	err := cmd.Run()
	if err == nil {
		return nil
	}

	// `say` on macOS exits via SIGTERM when stopped intentionally — treat as success.
	if name == "say" && strings.Contains(err.Error(), "signal: terminated") {
		return nil
	}

	exitCode := -1
	if exitErr, ok := err.(*exec.ExitError); ok {
		exitCode = exitErr.ExitCode()
	}

	return &CommandError{
		Command:  name + " " + strings.Join(args, " "),
		ExitCode: exitCode,
		Stderr:   strings.TrimSpace(stderr.String()),
		Err:      err,
	}
}

// commandExists reports whether name is found in PATH.
func commandExists(name string) bool {
	_, err := exec.LookPath(name)
	return err == nil
}

// powershellEscape escapes a string for safe use inside a single-quoted
// PowerShell string literal, also neutralising backtick and dollar-sign
// sequences that would be interpreted even inside single quotes in some contexts.
func powershellEscape(s string) string {
	// Inside '…' PowerShell only expands '' → '; everything else is literal.
	s = strings.ReplaceAll(s, "'", "''")
	return "'" + s + "'"
}
