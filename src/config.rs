use anyhow::{Context, Result, bail};
use serde::{Deserialize, Serialize};
use std::{fs, io::ErrorKind, path::Path};

#[derive(Serialize, Deserialize)]
pub struct Config {
    pub voice: String,
    pub rate: String,
    pub pitch: String,
    pub prefer_linux_engine: String,
    pub test_text: String,
}

impl Default for Config {
    fn default() -> Self {
        Self {
            voice: String::new(),
            rate: String::new(),
            pitch: String::new(),
            prefer_linux_engine: "spd-say".into(),
            test_text: "Hi, this is a test.".into(),
        }
    }
}

impl Config {
    pub fn load(path: &Path) -> Result<Self> {
        let data = match fs::read(path) {
            Ok(data) => data,
            Err(error) if error.kind() == ErrorKind::NotFound => return Ok(Self::default()),
            Err(error) => return Err(error).with_context(|| format!("reading config {path:?}")),
        };
        let mut config = Self::default();
        let fields: Option<serde_json::Map<String, serde_json::Value>> =
            serde_json::from_slice(&data).with_context(|| format!("parsing config {path:?}"))?;
        for (key, value) in fields.into_iter().flatten() {
            let field = match key.to_ascii_lowercase().as_str() {
                "voice" => &mut config.voice,
                "rate" => &mut config.rate,
                "pitch" => &mut config.pitch,
                "prefer_linux_engine" => &mut config.prefer_linux_engine,
                "test_text" => &mut config.test_text,
                _ => continue,
            };
            if !value.is_null() {
                *field = value
                    .as_str()
                    .with_context(|| format!("config field {key:?} must be a string"))?
                    .into();
            }
        }
        for (name, value) in [("rate", &config.rate), ("pitch", &config.pitch)] {
            if !value.is_empty() && value.parse::<f64>().is_err() {
                bail!("invalid config: {name} {value:?} is not a valid number");
            }
        }
        Ok(config)
    }

    pub fn write_default(path: &Path) -> Result<()> {
        fs::write(path, serde_json::to_vec_pretty(&Self::default())?)
            .with_context(|| format!("writing config {path:?}"))
    }
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn loads_defaults_and_existing_config() -> Result<()> {
        let dir = tempfile::tempdir()?;
        let path = dir.path().join("config.json");
        assert_eq!(Config::load(&path)?.prefer_linux_engine, "spd-say");
        Config::write_default(&path)?;
        assert_eq!(Config::load(&path)?.test_text, Config::default().test_text);
        fs::write(&path, r#"{"voice":"Alice","rate":"500","test_text":null}"#)?;
        let config = Config::load(&path)?;
        assert_eq!(config.voice, "Alice");
        assert_eq!(config.rate, "500");
        assert_eq!(config.test_text, Config::default().test_text);
        fs::write(&path, r#"{"rate":"fast"}"#)?;
        assert!(Config::load(&path).is_err());
        Ok(())
    }
}
