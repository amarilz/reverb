package main

import (
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
)

var logger *log.Logger

// initLogger sets up a logger that writes to both stdout and a log file
// placed next to the running executable.
func initLogger() error {
	execDir, err := executableDir()
	if err != nil {
		return err
	}

	logPath := filepath.Join(execDir, "reverb.log")

	logFile, err := os.OpenFile(logPath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o644)
	if err != nil {
		return fmt.Errorf("opening log file %q: %w", logPath, err)
	}

	logger = log.New(
		io.MultiWriter(os.Stdout, logFile),
		"[reverb] ",
		log.Ldate|log.Ltime,
	)

	return nil
}

func logInfo(format string, v ...any) {
	if logger == nil {
		return
	}
	logger.Printf("[INFO] "+format, v...)
}

func logError(format string, v ...any) {
	if logger == nil {
		fmt.Fprintf(os.Stderr, "[ERROR] "+format+"\n", v...)
		return
	}
	logger.Printf("[ERROR] "+format, v...)
}
