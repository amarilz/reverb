package main

import (
	"io"
	"log"
	"os"
	"path/filepath"
)

type CommandError struct {
	Command  string
	ExitCode int
	StdErr   string
	Err      error
}

func (e *CommandError) Error() string {
	if e.Err != nil {
		return e.Err.Error()
	}
	return "command failed"
}

var logger *log.Logger

func initLogger() error {

	executablePath, err := os.Executable()
	if err != nil {
		return err
	}

	executableDir := filepath.Dir(executablePath)

	logPath := filepath.Join(executableDir, "app.log")

	logFile, err := os.OpenFile(
		logPath,
		os.O_CREATE|os.O_WRONLY|os.O_APPEND,
		0644,
	)
	if err != nil {
		return err
	}

	multiWriter := io.MultiWriter(os.Stdout, logFile)

	logger = log.New(
		multiWriter,
		"[clipboard-tts] ",
		log.Ldate|log.Ltime,
	)

	return nil
}

func logInfo(format string, v ...any) {
	logger.Printf("[INFO] "+format, v...)
}

func logError(format string, v ...any) {
	logger.Printf("[ERROR] "+format, v...)
}
