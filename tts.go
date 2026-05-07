package main

import (
	"errors"
	"fmt"
	"runtime"
	"strings"
)

// speak synthesises text using the TTS engine appropriate for the current OS.
func speak(text string, config AppConfig) error {
	switch runtime.GOOS {
	case "darwin":
		return speakDarwin(text, config)
	case "linux":
		return speakLinux(text, config)
	case "windows":
		return speakWindows(text, config)
	default:
		return fmt.Errorf("TTS not supported on: %s", runtime.GOOS)
	}
}

// ── macOS ─────────────────────────────────────────────────────────────────────

func speakDarwin(text string, config AppConfig) error {
	args := make([]string, 0, 4)
	if config.Voice != "" {
		args = append(args, "-v", config.Voice)
	}
	if config.Rate != "" {
		args = append(args, "-r", config.Rate)
	}
	return runCommandWithInput(text, "say", args...)
}

// ── Linux ─────────────────────────────────────────────────────────────────────

func speakLinux(text string, config AppConfig) error {
	prefer := config.PreferLinuxEngine

	if prefer == "espeak-ng" && commandExists("espeak-ng") {
		return speakWithEspeakNG(text, config)
	}
	if prefer == "espeak" && commandExists("espeak") {
		return speakWithEspeak(text, config)
	}
	if prefer == "spd-say" && commandExists("spd-say") {
		return speakWithSpdSay(text, config)
	}

	if commandExists("espeak-ng") {
		return speakWithEspeakNG(text, config)
	}
	if commandExists("espeak") {
		return speakWithEspeak(text, config)
	}
	if commandExists("spd-say") {
		return speakWithSpdSay(text, config)
	}

	return errors.New("no TTS engine found on Linux; install espeak-ng, espeak or speech-dispatcher")
}

func speakWithEspeakNG(text string, config AppConfig) error {
	args := make([]string, 0, 8)

	if config.Voice != "" {
		args = append(args, "-v", config.Voice)
	}
	if config.Rate != "" {
		args = append(args, "-s", config.Rate)
	}
	if config.Pitch != "" {
		args = append(args, "-p", config.Pitch)
	}

	args = append(args, "--stdin")

	err := runCommandWithInput(text, "espeak-ng", args...)
	if err == nil {
		return nil
	}

	if cmdErr, ok := err.(*CommandError); ok {
		if strings.Contains(cmdErr.Stderr, "Please install necessary MBROLA voice") ||
			strings.Contains(cmdErr.Stderr, "Could not load the specified mbrola voice file") {
			return fmt.Errorf(
				"MBROLA voice %q is not installed or cannot be loaded; install the corresponding package, for example: sudo apt install mbrola-%s",
				config.Voice,
				strings.TrimPrefix(config.Voice, "mb-"),
			)
		}
	}

	return err
}

func speakWithEspeak(text string, config AppConfig) error {
	args := make([]string, 0, 8)
	if config.Voice != "" {
		args = append(args, "-v", config.Voice)
	}
	if config.Rate != "" {
		args = append(args, "-s", config.Rate)
	}
	if config.Pitch != "" {
		args = append(args, "-p", config.Pitch)
	}
	// Pass text via stdin using the "pipe" flag to avoid shell-quoting issues.
	args = append(args, "--stdin")
	return runCommandWithInput(text, "espeak", args...)
}

// speakWithSpdSay uses stdin so that long or special-character texts are safe.
func speakWithSpdSay(text string, config AppConfig) error {
	args := make([]string, 0, 6)
	if config.Rate != "" {
		args = append(args, "--rate", config.Rate)
	}
	if config.Pitch != "" {
		args = append(args, "--pitch", config.Pitch)
	}
	args = append(args, "-e")
	return runCommandWithInput(text, "spd-say", args...)
}

// ── Windows ───────────────────────────────────────────────────────────────────

func speakWindows(text string, config AppConfig) error {
	var sb strings.Builder
	sb.WriteString("Add-Type -AssemblyName System.Speech;\n")
	sb.WriteString("$synth = New-Object System.Speech.Synthesis.SpeechSynthesizer;\n")
	sb.WriteString("$synth.SetOutputToDefaultAudioDevice();\n")

	if config.Voice != "" {
		fmt.Fprintf(&sb, "$synth.SelectVoice(%s);\n", powershellEscape(config.Voice))
	}
	if config.Rate != "" {
		fmt.Fprintf(&sb, "$synth.Rate = [int](%s);\n", powershellEscape(config.Rate))
	}

	fmt.Fprintf(&sb, "$synth.Speak(%s);\n", powershellEscape(text))

	return runCommand("powershell", "-NoProfile", "-Command", sb.String())
}

// ── Stop ──────────────────────────────────────────────────────────────────────

// stopSpeaking interrupts any ongoing TTS playback.
func stopSpeaking() error {
	switch runtime.GOOS {
	case "darwin":
		return stopDarwin()
	case "linux":
		return stopLinux()
	case "windows":
		return stopWindows()
	default:
		return fmt.Errorf("unsupported operating system: %s", runtime.GOOS)
	}
}

func stopDarwin() error {
	err := runCommand("killall", "say")
	if err == nil {
		logInfo("speaker stopped")
		return nil
	}
	if cmdErr, ok := err.(*CommandError); ok {
		if cmdErr.ExitCode == 1 && strings.Contains(cmdErr.Stderr, "No matching processes") {
			logInfo("speaker was already stopped")
			return nil
		}
	}
	return err
}

func stopLinux() error {
	stopped := false

	if commandExists("spd-say") {
		_ = runCommand("spd-say", "--cancel")
		stopped = true
	}
	if commandExists("pkill") {
		_ = runCommand("pkill", "espeak-ng")
		_ = runCommand("pkill", "espeak")
		stopped = true
	}

	if stopped {
		logInfo("speaker stopped")
		return nil
	}
	return errors.New("stop not supported; install spd-say or ensure pkill is available")
}

func stopWindows() error {
	err := runCommand("taskkill", "/F", "/IM", "powershell.exe")
	if err == nil {
		logInfo("speaker stopped (powershell terminated)")
		return nil
	}
	if cmdErr, ok := err.(*CommandError); ok && cmdErr.ExitCode == 128 {
		logInfo("speaker was already stopped")
		return nil
	}
	return err
}

// ── List voices ───────────────────────────────────────────────────────────────

// listVoices prints the available TTS voices for the current OS.
func listVoices(config AppConfig) error {
	switch runtime.GOOS {
	case "darwin":
		return runCommand("say", "-v", "?")
	case "linux":
		return listLinuxVoices(config)
	case "windows":
		return listWindowsVoices()
	default:
		return fmt.Errorf("unsupported operating system: %s", runtime.GOOS)
	}
}

func listLinuxVoices(config AppConfig) error {
	prefer := config.PreferLinuxEngine

	if prefer == "espeak-ng" && commandExists("espeak-ng") {
		return runCommand("espeak-ng", "--voices")
	}
	if prefer == "espeak" && commandExists("espeak") {
		return runCommand("espeak", "--voices")
	}
	if prefer == "spd-say" && commandExists("spd-say") {
		return listSpeechDispatcherVoices()
	}

	if commandExists("espeak-ng") {
		return runCommand("espeak-ng", "--voices")
	}
	if commandExists("espeak") {
		return runCommand("espeak", "--voices")
	}
	if commandExists("spd-say") {
		return listSpeechDispatcherVoices()
	}

	return errors.New("no TTS engine found; install espeak-ng, espeak or speech-dispatcher")
}

func listSpeechDispatcherVoices() error {
	fmt.Println("speech-dispatcher does not expose a simple voice list.")
	fmt.Println("Showing available output modules instead:")

	return runCommand("spd-say", "--list-output-modules")
}

func listWindowsVoices() error {
	script := `
Add-Type -AssemblyName System.Speech;
$synth = New-Object System.Speech.Synthesis.SpeechSynthesizer;
$synth.GetInstalledVoices() | ForEach-Object {
    $v = $_.VoiceInfo;
    Write-Output ("{0,-30} | {1,-10} | {2,-6} | {3}" -f $v.Name, $v.Culture, $v.Gender, $v.Age);
}
`
	return runCommand("powershell", "-NoProfile", "-Command", script)
}
