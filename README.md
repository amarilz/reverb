# reverb

A lightweight command-line clipboard-to-speech tool designed to accelerate information consumption through audio. In
many situations, listening is significantly faster and less cognitively demanding than reading. `reverb` allows you to
quickly send selected text to the operating system TTS engine and consume articles, documentation, notes, code reviews,
or AI responses hands-free. \
The workflow is intentionally minimal: **select text, trigger a shortcut, and continue working while the content is
spoken aloud**.

---

### Status

`reverb` is primarily developed and personally used on **macOS**.

Current platform status:

| Platform | Status                                                             |
|----------|--------------------------------------------------------------------|
| macOS    | Fully tested and daily-driven                                      |
| Linux    | Supported, but current native TTS engines are not yet satisfactory |
| Windows  | Implemented but not yet thoroughly tested                          |

---

### Features

* Reads text directly from the clipboard
* Native OS TTS integration
* FIFO speech queue
* Skip Markdown code blocks during speech
* Stop current speech and clear the queue
* Voice / rate / pitch configuration
* Cross-platform architecture

---

### Platform Engines

| Platform | Engine                                           |
|----------|--------------------------------------------------|
| macOS    | `say`                                            |
| Linux    | `spd-say` or `espeak`                            |
| Windows  | `System.Speech.SpeechSynthesizer` via PowerShell |

---

### Installation

##### From source

```bash
git clone https://github.com/amarilz/reverb
cd reverb
make install # installed into $(go env GOPATH)/bin
```

---

##### Configuration

Generate a default config:

```bash
reverb --init-config
```

Example configuration for Italian speech on macOS:

```json
{
  "voice": "Alice",
  "rate": "620"
}
```

---

### Configuration Options

| Key                   | Type   | Default                 | Description               |
|-----------------------|--------|-------------------------|---------------------------|
| `voice`               | string | `""`                    | TTS voice name            |
| `rate`                | string | `""`                    | Speaking rate             |
| `pitch`               | string | `""`                    | Pitch (Linux only)        |
| `prefer_linux_engine` | string | `"spd-say"`             | `"spd-say"` or `"espeak"` |
| `test_text`           | string | `"Hi, this is a test."` | Text used by `--test`     |

---

### Usage

```bash
reverb [flags]
```

| Flag              | Description                             |
|-------------------|-----------------------------------------|
| `--config <path>` | Path to config file                     |
| `--queue`         | Queue speech requests FIFO-style        |
| `--skip-code`     | Skip fenced Markdown code blocks        |
| `--stop`          | Stop current speech and clear the queue |
| `--list-voices`   | Print available voices                  |
| `--test`          | Speak the configured `test_text`        |
| `--init-config`   | Generate a default config file          |
| `--voice <name>`  | Override voice                          |
| `--rate <value>`  | Override rate                           |
| `--pitch <value>` | Override pitch                          |
| `--text <string>` | Override `test_text`                    |

---

### FIFO Queue

When `--queue` is enabled, multiple invocations are processed sequentially. This is particularly useful when repeatedly
triggering the tool through keyboard shortcuts while reading articles or documentation. \
Example:

```bash
reverb --queue --skip-code --config config.json
```

Speech requests are stored and consumed in FIFO order, ensuring that audio never overlaps.

---

### Markdown Code Skipping

When using:

```bash
--skip-code
```

Markdown fenced code blocks are automatically removed before speech synthesis.

Example:

````markdown
This will be spoken.

```go
fmt.Println("this will be skipped")
```
````

Inline code using single backticks is preserved.

---

### Personal macOS Workflow

`reverb` is mainly designed around a keyboard-driven workflow on macOS. I personally use two [Hammerspoon](https://github.com/hammerspoon/hammerspoon) shortcuts:

##### Start TTS

```bash
reverb --queue --skip-code --config configPath
```

This shortcut also automatically copies (via Hammerspoon Lua script) the currently selected text into the clipboard before launching `reverb`.

Typical workflow:

1. Select text
2. Trigger keyboard shortcut
3. Continue working while the text is spoken aloud

##### Stop TTS

```bash
reverb --stop
```

This immediately:

* stops the currently playing speech
* clears the remaining FIFO queue
