package main

import (
	"fmt"
	"runtime"
)

// readClipboard returns the current clipboard contents as a UTF-8 string.
func readClipboard() (string, error) {
	switch runtime.GOOS {
	case "darwin":
		return commandOutputWithEnv(
			[]string{
				"LANG=en_US.UTF-8",
				"LC_ALL=en_US.UTF-8",
				"LC_CTYPE=UTF-8",
			},
			"pbpaste",
			"-Prefer", "txt",
		)
	case "linux":
		return readLinuxClipboard()
	case "windows":
		return commandOutput("powershell", "-NoProfile", "-Command", "Get-Clipboard")
	default:
		return "", fmt.Errorf("unsupported operating system: %s", runtime.GOOS)
	}
}

// readLinuxClipboard tries wl-paste, xclip, and xsel in order.
func readLinuxClipboard() (string, error) {
	tools := [][]string{
		{"wl-paste", "--no-newline"},
		{"xclip", "-selection", "clipboard", "-o"},
		{"xsel", "--clipboard", "--output"},
	}

	var lastErr error
	for _, t := range tools {
		if !commandExists(t[0]) {
			continue
		}
		text, err := commandOutput(t[0], t[1:]...)
		if err == nil {
			return text, nil
		}
		lastErr = err
	}

	return "", fmt.Errorf(
		"cannot read clipboard on Linux; install wl-clipboard, xclip, or xsel: %w",
		lastErr,
	)
}
