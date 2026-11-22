mod config;
mod platform;
mod queue;
mod text;

use anyhow::{Context, Result, bail};
use chrono::Local;
use clap::Parser;
use config::Config;
use queue::Queue;
use std::fs::OpenOptions;
use std::io::Write;
use std::path::{Path, PathBuf};
use std::process::ExitCode;

#[derive(Parser)]
#[command(about = "Read clipboard text aloud", disable_version_flag = true)]
struct Args {
    /// Path to the JSON configuration
    #[arg(long, default_value = "config.json")]
    config: PathBuf,
    /// Stop Reverb playback and clear the queue
    #[arg(long)]
    stop: bool,
    /// Print available voices
    #[arg(long)]
    list_voices: bool,
    /// Speak the configured test text
    #[arg(long)]
    test: bool,
    /// Write the default configuration
    #[arg(long)]
    init_config: bool,
    /// Skip fenced Markdown code blocks
    #[arg(long)]
    skip_code: bool,
    /// Queue clipboard speech in FIFO order
    #[arg(long)]
    queue: bool,
    /// Print the version
    #[arg(long)]
    version: bool,
    /// Override the configured voice
    #[arg(long)]
    voice: Option<String>,
    /// Override the configured rate
    #[arg(long, allow_hyphen_values = true)]
    rate: Option<String>,
    /// Override the configured pitch
    #[arg(long, allow_hyphen_values = true)]
    pitch: Option<String>,
    /// Override the text used by --test
    #[arg(long)]
    text: Option<String>,
}

fn main() -> ExitCode {
    let mut args = Args::parse();
    if args.config.as_os_str().is_empty() {
        args.config = "config.json".into();
    }
    let result = (|| -> Result<()> {
        let executable = std::env::current_exe().context("resolving executable path")?;
        let dir = executable
            .parent()
            .context("resolving executable directory")?;
        if let Err(error) = run(args, dir) {
            log(dir, "ERROR", &format!("{error:#}"));
            return Err(error);
        }
        Ok(())
    })();
    if let Err(error) = result {
        eprintln!("error: {error:#}");
        ExitCode::FAILURE
    } else {
        ExitCode::SUCCESS
    }
}

fn run(args: Args, dir: &Path) -> Result<()> {
    if args.init_config {
        Config::write_default(&args.config)?;
        println!(
            "Default configuration written to: {}",
            args.config.display()
        );
        return Ok(());
    }

    let mut config = Config::load(&args.config).context("loading configuration")?;
    for (value, field) in [
        (args.voice, &mut config.voice),
        (args.rate, &mut config.rate),
        (args.pitch, &mut config.pitch),
        (args.text, &mut config.test_text),
    ] {
        if let Some(value) = value.filter(|value| !value.is_empty()) {
            *field = value;
        }
    }

    if args.list_voices {
        log(dir, "INFO", "listing voices");
        return platform::list_voices(&config);
    }
    if args.stop {
        Queue::new(dir)?.stop()?;
        log(dir, "INFO", "requested playback stop and cleared the queue");
        return Ok(());
    }
    if args.version {
        println!("{}", include_str!("../VERSION").trim());
        return Ok(());
    }
    if args.test {
        log(dir, "INFO", "running TTS test");
        let generation = Queue::new(dir)?.generation()?;
        return platform::speak(&config.test_text, &config, &generation);
    }

    let mut text = text::normalize(&platform::clipboard()?);
    if args.skip_code {
        text = text::normalize(text::strip_code_blocks(&text).as_bytes());
    }
    if text.is_empty() {
        bail!("clipboard is empty or contains unreadable text");
    }

    let queue = Queue::new(dir)?;
    if args.queue {
        queue.enqueue_and_drain(&text, &config, dir)
    } else {
        let generation = queue.generation()?;
        log_speech(dir, &text);
        platform::speak(&text, &config, &generation)
    }
}

fn log_speech(dir: &Path, text: &str) {
    log(
        dir,
        "INFO",
        &format!(
            "speaking (chars: {}, words: {})",
            text.len(),
            text.split_whitespace().count()
        ),
    );
}

fn log(dir: &Path, level: &str, message: &str) {
    let line = format!(
        "[reverb] {} [{level}] {message}\n",
        Local::now().format("%Y/%m/%d %H:%M:%S")
    );
    let _ = std::io::stdout().write_all(line.as_bytes());
    if let Ok(mut file) = OpenOptions::new()
        .create(true)
        .append(true)
        .open(dir.join("reverb.log"))
    {
        let _ = file.write_all(line.as_bytes());
    }
}
