package main

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

const queueWorkerLockStaleAfter = 6 * time.Hour

func enqueueAndDrain(text string, config AppConfig) error {
	if err := enqueueSpeechJob(text); err != nil {
		return err
	}

	release, acquired, err := tryAcquireQueueWorkerLock()
	if err != nil {
		return err
	}

	if !acquired {
		logInfo("speech job queued; another worker is already running")
		return nil
	}

	defer release()

	return drainSpeechQueue(config)
}

func queueBaseDir() (string, error) {
	execDir, err := executableDir()
	if err != nil {
		return "", err
	}

	return filepath.Join(execDir, "reverb-queue"), nil
}

func queueJobsDir() (string, error) {
	baseDir, err := queueBaseDir()
	if err != nil {
		return "", err
	}

	return filepath.Join(baseDir, "jobs"), nil
}

func queueWorkerLockDir() (string, error) {
	baseDir, err := queueBaseDir()
	if err != nil {
		return "", err
	}

	return filepath.Join(baseDir, "worker.lock"), nil
}

func enqueueSpeechJob(text string) error {
	jobsDir, err := queueJobsDir()
	if err != nil {
		return err
	}

	if err := os.MkdirAll(jobsDir, 0o755); err != nil {
		return fmt.Errorf("creating queue directory: %w", err)
	}

	jobName := fmt.Sprintf(
		"%s-%d.txt",
		time.Now().UTC().Format("20060102150405.000000000"),
		os.Getpid(),
	)

	jobPath := filepath.Join(jobsDir, jobName)

	if err := os.WriteFile(jobPath, []byte(text), 0o644); err != nil {
		return fmt.Errorf("writing speech job: %w", err)
	}

	logInfo("queued speech job: %s", jobName)
	return nil
}

func tryAcquireQueueWorkerLock() (func(), bool, error) {
	baseDir, err := queueBaseDir()
	if err != nil {
		return nil, false, err
	}

	lockDir, err := queueWorkerLockDir()
	if err != nil {
		return nil, false, err
	}

	if err := os.MkdirAll(baseDir, 0o755); err != nil {
		return nil, false, fmt.Errorf("creating queue base directory: %w", err)
	}

	for {
		err := os.Mkdir(lockDir, 0o755)
		if err == nil {
			logInfo("acquired queue worker lock")

			return func() {
				if err := os.Remove(lockDir); err != nil {
					logError("removing queue worker lock: %v", err)
				} else {
					logInfo("released queue worker lock")
				}
			}, true, nil
		}

		if !os.IsExist(err) {
			return nil, false, fmt.Errorf("creating queue worker lock: %w", err)
		}

		if isStaleQueueWorkerLock(lockDir) {
			logError("removing stale queue worker lock")
			_ = os.Remove(lockDir)
			continue
		}

		return nil, false, nil
	}
}

func isStaleQueueWorkerLock(lockDir string) bool {
	info, err := os.Stat(lockDir)
	if err != nil {
		return false
	}

	return time.Since(info.ModTime()) > queueWorkerLockStaleAfter
}

func drainSpeechQueue(config AppConfig) error {
	for {
		nextSpeechJobPath, ok, err := nextSpeechJob()
		if err != nil {
			return err
		}

		if !ok {
			logInfo("speech queue is empty")
			return nil
		}

		text, err := os.ReadFile(nextSpeechJobPath)
		if err != nil {
			return fmt.Errorf("reading speech job %q: %w", nextSpeechJobPath, err)
		}

		cleanText := normalizeText(string(text))
		if cleanText == "" {
			_ = os.Remove(nextSpeechJobPath)
			continue
		}

		wordCount := len(strings.Fields(cleanText))
		logInfo("speaking queued job (chars: %d, words: %d)", len(cleanText), wordCount)

		if err := speak(cleanText, config); err != nil {
			_ = os.Remove(nextSpeechJobPath)
			return err
		}

		if err := os.Remove(nextSpeechJobPath); err != nil {
			return fmt.Errorf("removing speech job %q: %w", nextSpeechJobPath, err)
		}
	}
}

func nextSpeechJob() (string, bool, error) {
	jobsDir, err := queueJobsDir()
	if err != nil {
		return "", false, err
	}

	entries, err := os.ReadDir(jobsDir)
	if err != nil {
		if os.IsNotExist(err) {
			return "", false, nil
		}
		return "", false, fmt.Errorf("reading queue directory: %w", err)
	}

	var jobs []string

	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}

		name := entry.Name()
		if strings.HasSuffix(name, ".txt") {
			jobs = append(jobs, filepath.Join(jobsDir, name))
		}
	}

	if len(jobs) == 0 {
		return "", false, nil
	}

	sort.Strings(jobs)
	return jobs[0], true, nil
}
