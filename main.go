package main

import (
	"bytes"
	"errors"
	"flag"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"unicode/utf8"

	"golang.org/x/text/encoding/charmap"
)

func main() {
	if err := initLogger(); err != nil {
		exitWithError(err)
	}

	configPath := flag.String("config", "config.json", "percorso del file di configurazione")
	stop := flag.Bool("stop", false, "interrompe la lettura in corso, dove supportato")
	listVoices := flag.Bool("list-voices", false, "stampa le voci disponibili")
	test := flag.Bool("test", false, "esegue un test TTS usando il testo di configurazione")
	initConfig := flag.Bool("init-config", false, "crea un file config.json di esempio")

	overrideVoice := flag.String("voice", "", "override temporaneo della voce")
	overrideRate := flag.String("rate", "", "override temporaneo della velocità")
	overridePitch := flag.String("pitch", "", "override temporaneo del pitch")
	overrideText := flag.String("text", "", "testo da usare con --test")

	flag.Parse()

	if *initConfig {
		if err := CreateDefaultConfig(*configPath); err != nil {
			exitWithError(err)
		}
		fmt.Println("Configurazione creata:", *configPath)
		return
	}

	config, err := LoadConfig(*configPath)
	if err != nil {
		exitWithError(fmt.Errorf("error on loading the configuration: %w", err))
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

	if *listVoices {
		logInfo("list-voices")
		if err := listVoicesForCurrentOS(config); err != nil {
			exitWithError(err)
		}
		return
	}

	if *stop {
		if err := stopSpeaking(); err != nil {
			exitWithError(err)
		}
		return
	}

	if *test {
		logInfo("testing speaker")
		if err := speak(config.TestText, config); err != nil {
			exitWithError(err)
		}
		return
	}

	text, err := readClipboard()
	if err != nil {
		exitWithError(err)
	}
	text = normalizeText(text)
	if text == "" {
		exitWithError(errors.New("clipboard empty or with unreadable text"))
	}
	wordCount := len(strings.Fields(text))
	logInfo("starting speaker (characters: %d, words: %d)", len(text), wordCount)
	if err := speak(text, config); err != nil {
		exitWithError(err)
	}
}

func normalizeText(text string) string {
	text = strings.ReplaceAll(text, "\r\n", "\n")
	text = strings.ReplaceAll(text, "\r", "\n")
	text = strings.TrimSpace(text)
	text = ensureUTF8(text)
	return text
}

func ensureUTF8(text string) string {
	text = strings.TrimSpace(text)

	if utf8.ValidString(text) {
		return text
	}
	if decoded, err := charmap.Windows1252.NewDecoder().String(text); err == nil && utf8.ValidString(decoded) {
		return strings.TrimSpace(decoded)
	}
	if decoded, err := charmap.Macintosh.NewDecoder().String(text); err == nil && utf8.ValidString(decoded) {
		return strings.TrimSpace(decoded)
	}

	logError("clipboard contains invalid UTF-8; replacing invalid bytes")
	return strings.ToValidUTF8(text, "")
}

func commandOutputWithEnv(env []string, name string, args ...string) (string, error) {
	cmd := exec.Command(name, args...)
	cmd.Env = append(os.Environ(), env...)

	output, err := cmd.Output()
	if err != nil {
		return "", err
	}

	return string(output), nil
}

func commandOutput(name string, args ...string) (string, error) {
	cmd := exec.Command(name, args...)
	output, err := cmd.Output()
	if err != nil {
		logError("error executing command: %s %s", name, strings.Join(args, " "))
		return "", err
	}
	return string(output), nil
}

func runCommand(name string, args ...string) error {
	cmd := exec.Command(name, args...)

	var stdout bytes.Buffer
	var stderr bytes.Buffer

	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()
	if err == nil {
		return nil
	}

	exitCode := -1
	if exitErr, ok := err.(*exec.ExitError); ok {
		exitCode = exitErr.ExitCode()
	}

	return &CommandError{
		Command:  name + " " + strings.Join(args, " "),
		ExitCode: exitCode,
		StdErr:   strings.TrimSpace(stderr.String()),
		Err:      err,
	}
}

func commandExists(name string) bool {
	_, err := exec.LookPath(name)
	return err == nil
}

func powershellQuote(value string) string {
	return "'" + strings.ReplaceAll(value, "'", "''") + "'"
}

func exitWithError(err error) {

	if logger != nil {

		if commandErr, ok := err.(*CommandError); ok {
			if commandErr.StdErr != "" {
				logError(
					"exitCode=%d stderr=%q command=%s",
					commandErr.ExitCode,
					commandErr.StdErr,
					commandErr.Command,
				)
			} else {
				logError(
					"exitCode=%d err=%v command=%s",
					commandErr.ExitCode,
					commandErr.Err,
					commandErr.Command,
				)
			}
		} else {
			logError("%v", err)
		}
	}

	_, _ = fmt.Fprintln(os.Stderr, "Errore:", err)

	os.Exit(1)
}
