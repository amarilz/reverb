use super::output;
use crate::{config::Config, queue::Generation};
use anyhow::{Context, Result, bail};
use std::env;
use std::io::{BufRead, BufReader, ErrorKind, Read, Write};
use std::net::{TcpStream, ToSocketAddrs};
use std::os::unix::net::UnixStream;
use std::path::PathBuf;
use std::process::Command;
use std::time::{Duration, Instant};

const POLL: Duration = Duration::from_millis(50);
const TIMEOUT: Duration = Duration::from_secs(5);

trait Connection: Read + Write {}
impl Connection for UnixStream {}
impl Connection for TcpStream {}

fn connect() -> Result<Box<dyn Connection>> {
    let address = env::var("SPEECHD_ADDRESS").unwrap_or_default();
    if let Some(address) = address.strip_prefix("inet_socket") {
        let mut parts = address.trim_start_matches(':').split(':');
        let host = parts
            .next()
            .filter(|host| !host.is_empty())
            .unwrap_or("127.0.0.1");
        let port = parts.next().map(str::parse).transpose()?.unwrap_or(6560);
        let mut result = None;
        for addr in (host, port).to_socket_addrs()? {
            match TcpStream::connect_timeout(&addr, TIMEOUT) {
                Ok(stream) => {
                    stream.set_read_timeout(Some(POLL))?;
                    stream.set_write_timeout(Some(TIMEOUT))?;
                    return Ok(Box::new(stream));
                }
                Err(error) => result = Some(error),
            }
        }
        bail!("connecting to Speech Dispatcher at {host}:{port}: {result:?}");
    }

    let socket = match address.as_str() {
        "" | "unix_socket" | "unix_socket:" => {
            use std::os::unix::fs::MetadataExt;
            let runtime = env::var_os("XDG_RUNTIME_DIR")
                .filter(|dir| !dir.is_empty())
                .map(PathBuf::from)
                .or_else(|| {
                    std::fs::metadata("/proc/self")
                        .ok()
                        .map(|meta| PathBuf::from(format!("/run/user/{}", meta.uid())))
                })
                .context("resolving Speech Dispatcher runtime directory")?;
            runtime.join("speech-dispatcher/speechd.sock")
        }
        _ => PathBuf::from(
            address
                .strip_prefix("unix_socket:")
                .context("unsupported SPEECHD_ADDRESS")?,
        ),
    };
    let stream = UnixStream::connect(&socket)
        .with_context(|| format!("connecting to Speech Dispatcher at {socket:?}"))?;
    stream.set_read_timeout(Some(POLL))?;
    stream.set_write_timeout(Some(TIMEOUT))?;
    Ok(Box::new(stream))
}

struct Session {
    stream: BufReader<Box<dyn Connection>>,
    line: Vec<u8>,
}

impl Session {
    fn send(&mut self, message: &str) -> Result<()> {
        self.stream.get_mut().write_all(message.as_bytes())?;
        Ok(())
    }

    fn response(&mut self) -> Result<Option<u16>> {
        loop {
            match self.stream.read_until(b'\n', &mut self.line) {
                Ok(0) => bail!("Speech Dispatcher closed the connection"),
                Ok(_) => {}
                Err(error)
                    if matches!(error.kind(), ErrorKind::WouldBlock | ErrorKind::TimedOut) =>
                {
                    return Ok(None);
                }
                Err(error) => return Err(error.into()),
            }
            let line = std::mem::take(&mut self.line);
            let code =
                std::str::from_utf8(line.get(..3).context("short Speech Dispatcher reply")?)?
                    .parse()?;
            match line.get(3) {
                Some(b'-') => continue,
                Some(b' ') if !(400..700).contains(&code) => return Ok(Some(code)),
                Some(b' ') => bail!(
                    "Speech Dispatcher: {}",
                    String::from_utf8_lossy(&line).trim()
                ),
                _ => bail!("invalid Speech Dispatcher reply"),
            }
        }
    }

    fn command(&mut self, command: &str) -> Result<()> {
        self.send(&format!("{command}\r\n"))?;
        self.expect_reply()
    }

    fn expect_reply(&mut self) -> Result<()> {
        let deadline = Instant::now() + TIMEOUT;
        while Instant::now() < deadline {
            if let Some(200..=299) = self.response()? {
                return Ok(());
            }
        }
        bail!("Speech Dispatcher response timed out")
    }
}

pub(super) fn speak(text: &str, config: &Config, generation: &Generation) -> Result<()> {
    let stream = connect().or_else(|first_error| {
        // spd-say uses the platform's normal Speech Dispatcher autospawn mechanism.
        output(Command::new("spd-say").arg("--list-output-modules"))
            .with_context(|| format!("{first_error:#}"))?;
        connect()
    })?;
    speak_on(stream, text, config, generation)
}

fn speak_on(
    stream: Box<dyn Connection>,
    text: &str,
    config: &Config,
    generation: &Generation,
) -> Result<()> {
    let mut session = Session {
        stream: BufReader::new(stream),
        line: Vec::new(),
    };
    session.command(r#"SET SELF CLIENT_NAME "reverb:reverb:main""#)?;
    session.command("SET SELF PRIORITY MESSAGE")?;
    session.command("SET SELF NOTIFICATION END ON")?;
    session.command("SET SELF NOTIFICATION CANCEL ON")?;
    for (option, value) in [("RATE", &config.rate), ("PITCH", &config.pitch)] {
        if !value.is_empty() {
            let value = value
                .parse::<i32>()
                .with_context(|| format!("invalid Speech Dispatcher {option}: {value:?}"))?;
            session.command(&format!("SET SELF {option} {value}"))?;
        }
    }

    if text.is_empty() || generation.cancelled()? {
        return Ok(());
    }
    // SSIP uses a dot-only line to end the message. Escape every leading dot.
    let mut payload = String::new();
    for line in text.replace("\r\n", "\n").replace('\r', "\n").split('\n') {
        if line.starts_with('.') {
            payload.push('.');
        }
        payload.push_str(line);
        payload.push_str("\r\n");
    }
    payload.push_str(".\r\n");
    session.command("SPEAK")?;
    session.send(&payload)?;
    session.expect_reply()?;

    loop {
        if generation.cancelled()? {
            return session.command("CANCEL SELF");
        }
        if matches!(session.response()?, Some(702 | 703)) {
            return Ok(());
        }
    }
}

#[cfg(test)]
mod tests {
    use super::*;
    use crate::queue::Queue;
    use std::thread;

    #[test]
    fn cancels_only_its_own_dispatcher_connection() -> Result<()> {
        let dir = tempfile::tempdir()?;
        let queue = Queue::new(dir.path())?;
        let generation = queue.generation()?;
        let (client, server) = UnixStream::pair()?;
        client.set_read_timeout(Some(POLL))?;
        server.set_read_timeout(Some(TIMEOUT))?;

        let server = thread::spawn(move || -> Result<()> {
            let mut socket = BufReader::new(server);
            let mut line = String::new();
            for expected in [
                r#"SET SELF CLIENT_NAME "reverb:reverb:main""#,
                "SET SELF PRIORITY MESSAGE",
                "SET SELF NOTIFICATION END ON",
                "SET SELF NOTIFICATION CANCEL ON",
                "SPEAK",
            ] {
                line.clear();
                socket.read_line(&mut line)?;
                assert_eq!(line.trim_end(), expected);
                socket.get_mut().write_all(b"200 OK\r\n")?;
            }
            line.clear();
            socket.read_line(&mut line)?;
            socket.read_line(&mut line)?;
            socket.read_line(&mut line)?;
            assert_eq!(line, "Hi\r\n..\r\n.\r\n");
            socket.get_mut().write_all(b"225-1\r\n225 QUEUED\r\n")?;
            queue.stop()?;
            line.clear();
            socket.read_line(&mut line)?;
            assert_eq!(line, "CANCEL SELF\r\n");
            socket.get_mut().write_all(b"200 OK\r\n")?;
            Ok(())
        });

        let playback = speak_on(Box::new(client), "Hi\n.", &Config::default(), &generation);
        server.join().expect("mock dispatcher panicked")?;
        playback
    }
}
