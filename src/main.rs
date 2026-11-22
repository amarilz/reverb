mod config;
mod platform;
mod queue;
mod text;

use anyhow::{Context, Result, bail};
use clap::Parser;
use config::Config;
use queue::Queue;
use std::path::{Path, PathBuf};
use std::process::ExitCode;

#[derive(Parser)]
#[command(about = "Read clipboard text aloud", disable_version_flag = true)]
struct Args {
    /// Path to the JSON configuration
    #[arg(long)]
    config: Option<PathBuf>,
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
    if let Err(error) = run(Args::parse()) {
        eprintln!("error: {error:#}");
        ExitCode::FAILURE
    } else {
        ExitCode::SUCCESS
    }
}

fn run(args: Args) -> Result<()> {
    if args.init_config {
        let path = args
            .config
            .as_deref()
            .unwrap_or_else(|| Path::new("config.json"));
        Config::write_default(path)?;
        println!("Default configuration written to: {}", path.display());
        return Ok(());
    }

    if args.stop {
        return Queue::for_user()?.stop();
    }
    if args.version {
        println!("{}", include_str!("../VERSION").trim());
        return Ok(());
    }

    let mut config = match args.config {
        Some(path) => Config::load(&path).context("loading configuration")?,
        None => Config::default(),
    };
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
        return platform::list_voices(&config);
    }
    if args.test {
        let generation = Queue::for_user()?.generation()?;
        return platform::speak(&config.test_text, &config, &generation);
    }

    let mut text = text::normalize(&platform::clipboard()?);
    if args.skip_code {
        text = text::normalize(text::strip_code_blocks(&text).as_bytes());
    }
    if text.is_empty() {
        bail!("clipboard is empty or contains unreadable text");
    }

    let queue = Queue::for_user()?;
    if args.queue {
        queue.enqueue_and_drain(&text, &config)
    } else {
        let generation = queue.generation()?;
        platform::speak(&text, &config, &generation)
    }
}
