use crate::{config::Config, queue::Generation};
use anyhow::{Context, Result, bail};
use std::io::{Seek, Write};
use std::process::{Command, Stdio};
use std::thread;
use std::time::Duration;

#[cfg(any(target_os = "linux", all(unix, test)))]
mod dispatcher;

pub fn clipboard() -> Result<Vec<u8>> {
    match std::env::consts::OS {
        "macos" => output(
            Command::new("pbpaste")
                .env("LANG", "en_US.UTF-8")
                .env("LC_ALL", "en_US.UTF-8")
                .env("LC_CTYPE", "UTF-8")
                .args(["-Prefer", "txt"]),
        ),
        "windows" => output(&mut powershell(
            "[Console]::OutputEncoding = New-Object System.Text.UTF8Encoding; Get-Clipboard -Raw",
        )),
        "linux" => {
            let mut last_error = None;
            for (name, args) in [
                ("wl-paste", &["--no-newline"][..]),
                ("xclip", &["-selection", "clipboard", "-o"][..]),
                ("xsel", &["--clipboard", "--output"][..]),
            ] {
                match output(Command::new(name).args(args)) {
                    Ok(bytes) => return Ok(bytes),
                    Err(error) => last_error = Some(error),
                }
            }
            Err(last_error.unwrap())
                .context("cannot read clipboard on Linux; install wl-clipboard, xclip, or xsel")
        }
        os => bail!("clipboard unsupported on {os}"),
    }
}

fn output(command: &mut Command) -> Result<Vec<u8>> {
    let result = command
        .output()
        .with_context(|| format!("running {:?}", command.get_program()))?;
    if !result.status.success() {
        bail!(
            "{:?} failed ({}): {}",
            command.get_program(),
            result.status,
            String::from_utf8_lossy(&result.stderr).trim()
        );
    }
    Ok(result.stdout)
}

pub fn list_voices(config: &Config) -> Result<()> {
    let bytes = match std::env::consts::OS {
        "macos" => output(Command::new("say").args(["-v", "?"]))?,
        "windows" => output(&mut powershell(
            r#"
$ErrorActionPreference = 'Stop'
[Console]::OutputEncoding = New-Object System.Text.UTF8Encoding
Add-Type -AssemblyName System.Speech
$synth = New-Object System.Speech.Synthesis.SpeechSynthesizer
$synth.GetInstalledVoices() | ForEach-Object {
    $v = $_.VoiceInfo
    "{0,-30} | {1,-10} | {2,-6} | {3}" -f $v.Name, $v.Culture, $v.Gender, $v.Age
}
"#,
        ))?,
        "linux" => {
            let engine = linux_engine(config)?;
            let flag = if engine == "spd-say" {
                "--list-output-modules"
            } else {
                "--voices"
            };
            output(Command::new(engine).arg(flag))?
        }
        os => bail!("voice listing unsupported on {os}"),
    };
    std::io::stdout().write_all(&bytes)?;
    Ok(())
}

pub fn speak(text: &str, config: &Config, generation: &Generation) -> Result<()> {
    if generation.cancelled()? {
        return Ok(());
    }
    let mut command = match std::env::consts::OS {
        "macos" => {
            let mut command = Command::new("say");
            setting(&mut command, "-v", &config.voice);
            setting(&mut command, "-r", &config.rate);
            command
        }
        "windows" => {
            let mut command = powershell(
                r#"
$ErrorActionPreference = 'Stop'
[Console]::InputEncoding = New-Object System.Text.UTF8Encoding
Add-Type -AssemblyName System.Speech
$synth = New-Object System.Speech.Synthesis.SpeechSynthesizer
$synth.SetOutputToDefaultAudioDevice()
if ($env:REVERB_VOICE) { $synth.SelectVoice($env:REVERB_VOICE) }
if ($env:REVERB_RATE) { $synth.Rate = [int]$env:REVERB_RATE }
$synth.Speak([Console]::In.ReadToEnd())
"#,
            );
            command
                .env("REVERB_VOICE", &config.voice)
                .env("REVERB_RATE", &config.rate);
            command
        }
        "linux" => {
            let engine = linux_engine(config)?;
            #[cfg(any(target_os = "linux", all(unix, test)))]
            if engine == "spd-say" {
                return dispatcher::speak(text, config, generation);
            }
            let mut command = Command::new(engine);
            setting(&mut command, "-v", &config.voice);
            setting(&mut command, "-s", &config.rate);
            setting(&mut command, "-p", &config.pitch);
            command.arg("--stdin");
            command
        }
        os => bail!("speech unsupported on {os}"),
    };
    play(&mut command, text, config, generation)
}

fn setting(command: &mut Command, flag: &str, value: &str) {
    if !value.is_empty() {
        command.args([flag, value]);
    }
}

fn powershell(script: &str) -> Command {
    let mut command = Command::new("powershell");
    command.args(["-NoProfile", "-NonInteractive", "-Command", script]);
    command
}

fn linux_engine(config: &Config) -> Result<&'static str> {
    let engines = ["espeak-ng", "espeak", "spd-say"];
    let preferred = engines
        .iter()
        .copied()
        .find(|&name| name == config.prefer_linux_engine);
    preferred
        .into_iter()
        .chain(engines)
        .find(|name| {
            #[cfg(unix)]
            {
                use std::os::unix::fs::PermissionsExt;
                std::env::var_os("PATH").is_some_and(|path| {
                    std::env::split_paths(&path).any(|dir| {
                        dir.join(name).metadata().is_ok_and(|meta| {
                            meta.is_file() && meta.permissions().mode() & 0o111 != 0
                        })
                    })
                })
            }
            #[cfg(not(unix))]
            {
                let _ = name;
                false
            }
        })
        .context("no TTS engine found on Linux; install espeak-ng, espeak or speech-dispatcher")
}

fn play(command: &mut Command, text: &str, config: &Config, generation: &Generation) -> Result<()> {
    // File-backed I/O lets the owner observe stop requests even if the engine quits reading.
    let mut input = tempfile::tempfile()?;
    input.write_all(text.as_bytes())?;
    input.rewind()?;
    let errors = tempfile::tempfile()?;
    let mut child = command
        .stdin(input)
        .stdout(Stdio::null())
        .stderr(errors.try_clone()?)
        .spawn()
        .with_context(|| format!("starting {:?}", command.get_program()))?;
    let result = loop {
        match generation.cancelled() {
            Ok(true) => break Ok(None),
            Ok(false) => {}
            Err(error) => break Err(error),
        }
        match child.try_wait() {
            Ok(Some(status)) => break Ok(Some(status)),
            Ok(None) => thread::sleep(Duration::from_millis(25)),
            Err(error) => break Err(error.into()),
        }
    };
    let status = match result {
        Ok(Some(status)) => status,
        other => {
            // Use only the owned child handle, without searching for processes by name.
            if child.try_wait()?.is_none() {
                child.kill()?;
                child.wait()?;
            }
            return other.map(|_| ());
        }
    };
    if status.success() {
        return Ok(());
    }
    let mut errors = errors;
    errors.rewind()?;
    let mut stderr = Vec::new();
    std::io::Read::read_to_end(&mut errors, &mut stderr)?;
    let stderr = String::from_utf8_lossy(&stderr);
    if command.get_program() == "espeak-ng"
        && (stderr.contains("Please install necessary MBROLA voice")
            || stderr.contains("Could not load the specified mbrola voice file"))
    {
        let package = config.voice.trim_start_matches("mb-");
        bail!(
            "MBROLA voice {:?} cannot be loaded; install the corresponding package (e.g. mbrola-{package})",
            config.voice
        );
    }
    bail!(
        "{:?} failed ({status}): {}",
        command.get_program(),
        stderr.trim()
    )
}
