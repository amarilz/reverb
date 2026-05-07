package main

import (
	"errors"
	"flag"
	"fmt"
	"os"
	"strings"
)

func main() {
	// Logger must be initialised first; subsequent helpers rely on it.
	if err := initLogger(); err != nil {
		// Logger is not yet available — write directly to stderr.
		fmt.Fprintf(os.Stderr, "error: initialising logger: %v\n", err)
		os.Exit(1)
	}

	// ── Flags ─────────────────────────────────────────────────────────────────
	configPath := flag.String("config", "config.json", "path to the configuration file")
	doStop := flag.Bool("stop", false, "stop ongoing speech (where supported)")
	doListVoices := flag.Bool("list-voices", false, "print available voices and exit")
	doTest := flag.Bool("test", false, "speak the test_text from config and exit")
	doInitConfig := flag.Bool("init-config", false, "write a default config.json and exit")

	overrideVoice := flag.String("voice", "", "override voice for this run")
	overrideRate := flag.String("rate", "", "override speaking rate for this run")
	overridePitch := flag.String("pitch", "", "override pitch for this run")
	overrideText := flag.String("text", "", "override test_text when used with --test")

	flag.Parse()

	// ── init-config ───────────────────────────────────────────────────────────
	if *doInitConfig {
		if err := CreateDefaultConfig(*configPath); err != nil {
			exitWithError(err)
		}
		fmt.Println("Default configuration written to:", *configPath)
		return
	}

	// ── Load & patch config ───────────────────────────────────────────────────
	config, err := LoadConfig(*configPath)
	if err != nil {
		exitWithError(fmt.Errorf("loading configuration: %w", err))
	}

	if *overrideVoice != "" {
		config.Voice = *overrideVoice
	}
	if *overrideRate != "" {
		config.Rate = *overrideRate
	}
	if *overridePitch != "" {
		config.Pitch = *overridePitch
	}
	if *overrideText != "" {
		config.TestText = *overrideText
	}

	// ── Subcommands ───────────────────────────────────────────────────────────
	if *doListVoices {
		logInfo("listing voices")
		if err := listVoices(config); err != nil {
			exitWithError(err)
		}
		return
	}

	if *doStop {
		logInfo("stopping speaker")
		if err := stopSpeaking(); err != nil {
			exitWithError(err)
		}
		return
	}

	if *doTest {
		logInfo("running TTS test")
		if err := speak(config.TestText, config); err != nil {
			exitWithError(err)
		}
		return
	}

	// ── Main path: read clipboard and speak ───────────────────────────────────
	mainPath(err, config)
}

func mainPath(err error, config AppConfig) {
	text, err := readClipboard()
	if err != nil {
		exitWithError(err)
	}

	text = normalizeText(text)
	if text == "" {
		exitWithError(errors.New("clipboard is empty or contains unreadable text"))
	}

	wordCount := len(strings.Fields(text))
	logInfo("speaking (chars: %d, words: %d)", len(text), wordCount)

	if err := speak(text, config); err != nil {
		exitWithError(err)
	}
}

// exitWithError logs err with full detail and terminates with exit code 1.
func exitWithError(err error) {
	if cmdErr, ok := err.(*CommandError); ok {
		if cmdErr.Stderr != "" {
			logError("exit_code=%d stderr=%q command=%s", cmdErr.ExitCode, cmdErr.Stderr, cmdErr.Command)
		} else {
			logError("exit_code=%d err=%v command=%s", cmdErr.ExitCode, cmdErr.Err, cmdErr.Command)
		}
	} else {
		logError("%v", err)
	}

	fmt.Fprintf(os.Stderr, "error: %v\n", err)
	os.Exit(1)
}
