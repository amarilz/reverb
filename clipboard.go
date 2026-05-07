package main

import (
	"fmt"
	"runtime"
)

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
			"-Prefer",
			"txt",
		)
	case "linux":
		return readLinuxClipboard()
	case "windows":
		return commandOutput("powershell", "-NoProfile", "-Command", "Get-Clipboard")
	default:
		return "", fmt.Errorf("sistema operativo non supportato: %s", runtime.GOOS)
	}
}

func readLinuxClipboard() (string, error) {
	candidates := [][]string{
		{"wl-paste", "--no-newline"},
		{"xclip", "-selection", "clipboard", "-o"},
		{"xsel", "--clipboard", "--output"},
	}

	var lastErr error
	for _, candidate := range candidates {
		text, err := commandOutput(candidate[0], candidate[1:]...)
		if err == nil {
			return text, nil
		}
		lastErr = err
	}

	return "", fmt.Errorf("impossibile leggere la clipboard su Linux; installa wl-clipboard, xclip o xsel: %w", lastErr)
}
