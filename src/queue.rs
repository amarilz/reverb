use crate::{config::Config, platform, text};
use anyhow::{Context, Result};
use serde::{Deserialize, Serialize};
use std::fs::{self, File, OpenOptions, TryLockError};
use std::io::{ErrorKind, Write};
use std::path::{Path, PathBuf};
use tempfile::NamedTempFile;

pub struct Queue {
    root: PathBuf,
    jobs: PathBuf,
}

#[derive(Deserialize, Serialize)]
struct Job<T, C> {
    text: T,
    config: C,
}

pub struct Generation {
    path: PathBuf,
    value: Vec<u8>,
}

impl Generation {
    pub fn cancelled(&self) -> Result<bool> {
        Ok(read_generation(&self.path)? != self.value)
    }
}

fn read_generation(path: &Path) -> Result<Vec<u8>> {
    match fs::read(path) {
        Ok(value) => Ok(value),
        Err(error) if error.kind() == ErrorKind::NotFound => Ok(Vec::new()),
        Err(error) => Err(error).context("reading playback generation"),
    }
}

impl Queue {
    pub fn for_user() -> Result<Self> {
        let state_dir = dirs::state_dir()
            .or_else(dirs::data_local_dir)
            .context("resolving the user state directory")?;
        Self::new(&state_dir.join("reverb"))
    }

    pub fn new(root: &Path) -> Result<Self> {
        let jobs = root.join("jobs");
        fs::create_dir_all(&jobs).with_context(|| format!("creating queue at {jobs:?}"))?;
        Ok(Self {
            root: root.to_owned(),
            jobs,
        })
    }

    fn lock_file(&self, name: &str) -> Result<File> {
        OpenOptions::new()
            .create(true)
            .truncate(false)
            .read(true)
            .write(true)
            .open(self.root.join(name))
            .with_context(|| format!("opening queue lock {name}"))
    }

    fn state_lock(&self) -> Result<File> {
        let file = self.lock_file("state.lock")?;
        file.lock().context("locking queue state")?;
        Ok(file)
    }

    pub fn generation(&self) -> Result<Generation> {
        let path = self.root.join("generation");
        let value = read_generation(&path)?;
        Ok(Generation { path, value })
    }

    pub fn stop(&self) -> Result<()> {
        let _state = self.state_lock()?;
        let mut marker = NamedTempFile::new_in(&self.root)?;
        let token = marker
            .path()
            .file_name()
            .context("missing stop token")?
            .to_string_lossy()
            .into_owned();
        marker.write_all(token.as_bytes())?;
        marker
            .persist(self.root.join("generation"))
            .context("publishing playback stop")?;
        for job in self.pending()? {
            fs::remove_file(&job).with_context(|| format!("clearing queued job {job:?}"))?;
        }
        Ok(())
    }

    pub fn enqueue_and_drain(&self, text: &str, config: &Config) -> Result<()> {
        let state = self.state_lock()?;
        self.enqueue(text, config)?;
        let worker = self.lock_file("drain.lock")?;
        match worker.try_lock() {
            Ok(()) => {}
            Err(TryLockError::WouldBlock) => return Ok(()),
            Err(error) => return Err(error).context("locking queue worker"),
        }
        drop(state);

        let mut first_error = None;
        loop {
            let state = self.state_lock()?;
            let Some(job) = self.pending()?.into_iter().next() else {
                // Release the worker while producers are still excluded by the state lock.
                drop(worker);
                return first_error.map_or(Ok(()), Err);
            };
            let generation = self.generation()?;
            let contents = fs::read(&job).with_context(|| format!("reading queued job {job:?}"))?;
            fs::remove_file(&job).with_context(|| format!("claiming queued job {job:?}"))?;
            drop(state);

            let playback = if job.extension().is_some_and(|ext| ext == "txt") {
                let text = text::normalize(&contents);
                if text.is_empty() {
                    continue;
                }
                platform::speak(&text, config, &generation)
            } else {
                serde_json::from_slice::<Job<String, Config>>(&contents)
                    .with_context(|| format!("decoding queued job {job:?}"))
                    .and_then(|job| platform::speak(&job.text, &job.config, &generation))
            };
            if let Err(error) = playback {
                first_error.get_or_insert(error);
            }
        }
    }

    fn enqueue(&self, text: &str, config: &Config) -> Result<()> {
        // New names sort after timestamp-based jobs from the Go implementation.
        let sequence = self
            .pending()?
            .last()
            .and_then(|path| {
                path.file_stem()?
                    .to_str()?
                    .strip_prefix("job-")?
                    .parse::<u64>()
                    .ok()
            })
            .unwrap_or(0)
            .checked_add(1)
            .context("queue sequence exhausted")?;
        let mut job = NamedTempFile::new_in(&self.jobs)?;
        serde_json::to_writer(&mut job, &Job { text, config })?;
        job.persist_noclobber(self.jobs.join(format!("job-{sequence:020}.json")))
            .context("publishing queued speech")?;
        Ok(())
    }

    fn pending(&self) -> Result<Vec<PathBuf>> {
        let mut jobs = Vec::new();
        for entry in fs::read_dir(&self.jobs).context("reading speech queue")? {
            let entry = entry?;
            if entry.file_type()?.is_file()
                && entry
                    .path()
                    .extension()
                    .is_some_and(|ext| ext == "txt" || ext == "json")
            {
                jobs.push(entry.path());
            }
        }
        jobs.sort_unstable();
        Ok(jobs)
    }
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn clears_legacy_and_new_jobs_and_cancels_previous_generation() -> Result<()> {
        let dir = tempfile::tempdir()?;
        let queue = Queue::new(dir.path())?;
        let previous = queue.generation()?;
        fs::write(queue.jobs.join("20260101000000.000000000-42.txt"), "legacy")?;
        let italian = Config {
            voice: "Alice".into(),
            rate: "620".into(),
            ..Config::default()
        };
        let english = Config {
            voice: "Samantha".into(),
            rate: "450".into(),
            ..Config::default()
        };
        queue.enqueue("first", &italian)?;
        queue.enqueue("second", &english)?;
        let jobs = queue.pending()?;
        assert_eq!(fs::read_to_string(&jobs[0])?, "legacy");
        let first: Job<String, Config> = serde_json::from_slice(&fs::read(&jobs[1])?)?;
        let second: Job<String, Config> = serde_json::from_slice(&fs::read(&jobs[2])?)?;
        assert_eq!(
            (
                first.text.as_str(),
                first.config.voice.as_str(),
                first.config.rate.as_str()
            ),
            ("first", "Alice", "620")
        );
        assert_eq!(
            (
                second.text.as_str(),
                second.config.voice.as_str(),
                second.config.rate.as_str()
            ),
            ("second", "Samantha", "450")
        );
        queue.stop()?;
        assert!(queue.pending()?.is_empty());
        assert!(previous.cancelled()?);
        assert!(!queue.generation()?.cancelled()?);
        Ok(())
    }
}
