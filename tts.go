package main

import (
	"bytes"
	"errors"
	"fmt"
	"os/exec"
	"runtime"
	"strings"
)

func speak(text string, config AppConfig) error {
	switch runtime.GOOS {
	case "darwin":
		return speakDarwin(text, config)
	case "linux":
		return speakLinux(text, config)
	case "windows":
		return speakWindows(text, config)
	default:
		return fmt.Errorf("TTS non supportato su: %s", runtime.GOOS)
	}
}

func speakDarwin(text string, config AppConfig) error {
	args := make([]string, 0)

	if config.Voice != "" {
		args = append(args, "-v", config.Voice)
	}

	if config.Rate != "" {
		args = append(args, "-r", config.Rate)
	}

	return runCommandWithInput(text, "say", args...)
}

func runCommandWithInput(input string, name string, args ...string) error {
	cmd := exec.Command(name, args...)

	var stdout bytes.Buffer
	var stderr bytes.Buffer

	cmd.Stdin = strings.NewReader(input)
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()
	if err == nil {
		return nil
	}

	if name == "say" && strings.Contains(err.Error(), "signal: terminated") {
		return nil
	}

	exitCode := -1
	if exitErr, ok := err.(*exec.ExitError); ok {
		exitCode = exitErr.ExitCode()
	}

	return &CommandError{
		Command:  name + " " + strings.Join(args, " ") + " <stdin>",
		ExitCode: exitCode,
		StdErr:   strings.TrimSpace(stderr.String()),
		Err:      err,
	}
}

func speakLinux(text string, config AppConfig) error {
	if config.PreferLinuxEngine == "espeak" && commandExists("espeak") {
		return speakWithEspeak(text, config)
	}

	if commandExists("spd-say") {
		return speakWithSpdSay(text, config)
	}

	if commandExists("espeak") {
		return speakWithEspeak(text, config)
	}

	return errors.New("nessun TTS locale trovato su Linux; installa speech-dispatcher/spd-say oppure espeak")
}

func speakWithSpdSay(text string, config AppConfig) error {
	args := make([]string, 0)

	if config.Rate != "" {
		args = append(args, "--rate", config.Rate)
	}

	if config.Pitch != "" {
		args = append(args, "--pitch", config.Pitch)
	}

	args = append(args, text)
	return runCommand("spd-say", args...)
}

func speakWithEspeak(text string, config AppConfig) error {
	args := make([]string, 0)

	if config.Voice != "" {
		args = append(args, "-v", config.Voice)
	}

	if config.Rate != "" {
		args = append(args, "-s", config.Rate)
	}

	if config.Pitch != "" {
		args = append(args, "-p", config.Pitch)
	}

	args = append(args, text)
	return runCommand("espeak", args...)
}

func speakWindows(text string, config AppConfig) error {
	script := `
Add-Type -AssemblyName System.Speech;
$synth = New-Object System.Speech.Synthesis.SpeechSynthesizer;
$synth.SetOutputToDefaultAudioDevice();
`

	if config.Voice != "" {
		script += fmt.Sprintf("$synth.SelectVoice(%s);\n", powershellQuote(config.Voice))
	}

	if config.Rate != "" {
		script += fmt.Sprintf("$synth.Rate = [int](%s);\n", powershellQuote(config.Rate))
	}

	script += fmt.Sprintf("$synth.Speak(%s);\n", powershellQuote(text))

	return runCommand("powershell", "-NoProfile", "-Command", script)
}

func stopSpeaking() error {
	switch runtime.GOOS {
	case "darwin":
		return stopDarwinSpeaking()
	case "linux":
		if commandExists("spd-say") {
			return runCommand("spd-say", "--cancel")
		}
		if commandExists("pkill") {
			_ = runCommand("pkill", "espeak")
			return nil
		}
		return errors.New("stop non supportato: installa spd-say o pkill")
	case "windows":
		return errors.New("stop non implementato su Windows per System.Speech")
	default:
		return fmt.Errorf("sistema operativo non supportato: %s", runtime.GOOS)
	}
}

func stopDarwinSpeaking() error {
	err := runCommand("killall", "say")
	if err == nil {
		logInfo("speaker stopped")
		return nil
	}

	if commandErr, ok := err.(*CommandError); ok {
		if commandErr.ExitCode == 1 &&
			strings.Contains(commandErr.StdErr, "No matching processes") {
			logInfo("speaker already stopped")
			return nil
		}
	}

	return err
}

func listVoicesForCurrentOS(config AppConfig) error {
	switch runtime.GOOS {
	case "darwin":
		return listDarwinVoices()
	case "linux":
		return listLinuxVoices(config)
	case "windows":
		return listWindowsVoices()
	default:
		return fmt.Errorf("sistema operativo non supportato: %s", runtime.GOOS)
	}
}

func listDarwinVoices() error {
	return runCommand("say", "-v", "?")
}

func listLinuxVoices(config AppConfig) error {
	if config.PreferLinuxEngine == "espeak" && commandExists("espeak") {
		return runCommand("espeak", "--voices")
	}

	if commandExists("spd-say") {
		fmt.Println("speech-dispatcher/spd-say non espone sempre una lista voci semplice.")
		fmt.Println("Provo a mostrare i moduli disponibili:")
		_ = runCommand("spd-say", "--help")

		if commandExists("espeak") {
			fmt.Println()
			fmt.Println("Voci disponibili tramite espeak:")
			return runCommand("espeak", "--voices")
		}

		return nil
	}

	if commandExists("espeak") {
		return runCommand("espeak", "--voices")
	}

	return errors.New("nessun motore TTS trovato: installa speech-dispatcher/spd-say oppure espeak")
}

func listWindowsVoices() error {
	script := `
Add-Type -AssemblyName System.Speech;
$synth = New-Object System.Speech.Synthesis.SpeechSynthesizer;
$synth.GetInstalledVoices() | ForEach-Object {
    $info = $_.VoiceInfo;
    Write-Output ($info.Name + " | " + $info.Culture + " | " + $info.Gender + " | " + $info.Age);
}
`

	return runCommand("powershell", "-NoProfile", "-Command", script)
}
