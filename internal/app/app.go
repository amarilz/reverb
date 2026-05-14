package app

import (
	"errors"
	"flag"
	"fmt"
	"os"
	"strings"
)

var Version = "dev"

func Run() error {
	// logger must be initialised first; subsequent helpers rely on it.
	if err := initLogger(); err != nil {
		// logger is not yet available — write directly to stderr.
		fmt.Fprintf(os.Stderr, "error: initialising logger: %v\n", err)
		os.Exit(1)
	}

	configPath := flag.String("config", "config.json", "path to the configuration file")
	doStop := flag.Bool("stop", false, "stop ongoing speech (where supported)")
	doListVoices := flag.Bool("list-voices", false, "print available voices and exit")
	doTest := flag.Bool("test", false, "speak the test_text from config and exit")
	doInitConfig := flag.Bool("init-config", false, "write a default config.json and exit")
	skipCode := flag.Bool("skip-code", false, "skip fenced Markdown code blocks before speaking")
	doQueue := flag.Bool("queue", false, "queue speech if another reverb instance is already speaking")
	doVersion := flag.Bool("version", false, "print version and exit")

	overrideVoice := flag.String("voice", "", "override voice for this run")
	overrideRate := flag.String("rate", "", "override speaking rate for this run")
	overridePitch := flag.String("pitch", "", "override pitch for this run")
	overrideText := flag.String("text", "", "override test_text when used with --test")

	flag.Parse()

	if *doInitConfig {
		if err := CreateDefaultConfig(*configPath); err != nil {
			return logAndReturn(err)
		}
		fmt.Println("Default configuration written to:", *configPath)
		return nil
	}

	config, err := LoadConfig(*configPath)
	if err != nil {
		return logAndReturn(fmt.Errorf("loading configuration: %w", err))
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

	if *doListVoices {
		logInfo("listing voices")
		if err := listVoices(config); err != nil {
			return logAndReturn(err)
		}
		return nil
	}

	if *doStop {
		logInfo("stopping speaker and clearing queue")

		if err := clearSpeechQueue(); err != nil {
			return logAndReturn(err)
		}

		if err := stopSpeaking(); err != nil {
			return logAndReturn(err)
		}

		return nil
	}

	if *doVersion {
		fmt.Println(Version)
		return nil
	}

	if *doTest {
		logInfo("running TTS test")
		if err := speak(config.TestText, config); err != nil {
			return logAndReturn(err)
		}
		return nil
	}

	return mainPath(config, *skipCode, *doQueue)
}

func mainPath(config AppConfig, skipCode bool, queue bool) error {
	text, err := readClipboard()
	if err != nil {
		return logAndReturn(err)
	}

	text = normalizeText(text)

	if skipCode {
		text = stripMarkdownCodeBlocks(text)
		text = normalizeText(text)
	}

	if text == "" {
		return logAndReturn(errors.New("clipboard is empty or contains unreadable text"))
	}

	if queue {
		if err := enqueueAndDrain(text, config); err != nil {
			return logAndReturn(err)
		}
		return nil
	}

	wordCount := len(strings.Fields(text))
	logInfo("speaking (chars: %d, words: %d)", len(text), wordCount)

	if err := speak(text, config); err != nil {
		return logAndReturn(err)
	}

	return nil
}

func logAndReturn(err error) error {
	if cmdErr, ok := err.(*CommandError); ok {
		if cmdErr.Stderr != "" {
			logError("exit_code=%d stderr=%q command=%s", cmdErr.ExitCode, cmdErr.Stderr, cmdErr.Command)
		} else {
			logError("exit_code=%d err=%v command=%s", cmdErr.ExitCode, cmdErr.Err, cmdErr.Command)
		}
	} else {
		logError("%v", err)
	}

	return err
}
