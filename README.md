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
| macOS    | Primary development platform                                      |
| Linux    | Speech Dispatcher, eSpeak NG and eSpeak |
| Windows  | Windows PowerShell with System.Speech                          |

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
| Linux    | `spd-say`, `espeak-ng` or `espeak`                            |
| Windows  | `System.Speech.SpeechSynthesizer` via PowerShell |

---

### Installation

Download a prebuilt archive from the [GitHub Releases](https://github.com/amarilz/reverb/releases)
page. Releases include macOS and Linux for x86-64 and ARM64, plus Windows x86-64.

##### From source

Install Rust 1.89 or newer and Cargo.

```bash
git clone https://github.com/amarilz/reverb
cd reverb
cargo install --locked --path . --force # installs into ~/.cargo/bin by default
```

On Linux, install a clipboard reader (`wl-clipboard`, `xclip` or `xsel`) and at
least one speech engine. For Speech Dispatcher, install `spd-say` and make the
dispatcher available through its normal user socket or `SPEECHD_ADDRESS`.
On Windows, the app uses Windows PowerShell and the installed System.Speech voices.

Build a local executable with `cargo build --release --locked`. The binary is in
`target/release/`. Run `make release` to copy a build for the current platform
into `dist/`. To build for another platform, use `make release-<rust-target>`;
`make release-all` builds all five targets when their Rust targets, linkers and
SDKs are configured.

The root `VERSION` file sets the version printed by `reverb --version` and used
for GitHub Releases. To publish a release, update `VERSION` (for example, to
`1.1.1`). Push the version bump to `main` or run the
Release workflow manually from `main`. GitHub Actions tests and packages each
target, creates the corresponding `v<version>` tag, and publishes the archives
with release notes generated from Conventional Commits. Published versions are
skipped.

---

##### Configuration

The default configuration path is `config.json` in the current working directory. Existing JSON configurations work with the Rust release. Generate a default config:

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
| `--version`       | Print the version                       |
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

Queued requests are played one at a time in FIFO order, using the configuration
of the invocation that drains the queue. Queue files and `reverb.log` live next
to the executable. Leave that directory writable to use the queue and playback
controls.

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

This stops speech started by Reverb, including `--test` playback, and clears the remaining FIFO queue.
