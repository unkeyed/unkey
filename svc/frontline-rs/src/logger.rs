//! Hand-rolled structured logger: JSON lines on stdout, leveled, with base
//! attributes (instanceID, region, version) mirroring pkg/logger usage.

use std::io::Write;
use std::sync::atomic::{AtomicU8, Ordering};
use std::sync::{Mutex, OnceLock};
use std::time::{SystemTime, UNIX_EPOCH};

#[derive(Debug, Clone, Copy, PartialEq, Eq, PartialOrd, Ord)]
pub enum Level {
    Debug = 0,
    Info = 1,
    Warn = 2,
    Error = 3,
}

impl Level {
    fn as_str(self) -> &'static str {
        match self {
            Level::Debug => "DEBUG",
            Level::Info => "INFO",
            Level::Warn => "WARN",
            Level::Error => "ERROR",
        }
    }
}

static MIN_LEVEL: AtomicU8 = AtomicU8::new(1);
static BASE_ATTRS: OnceLock<Mutex<Vec<(String, String)>>> = OnceLock::new();

fn base_attrs() -> &'static Mutex<Vec<(String, String)>> {
    BASE_ATTRS.get_or_init(|| Mutex::new(Vec::new()))
}

pub fn set_min_level(level: Level) {
    MIN_LEVEL.store(level as u8, Ordering::Relaxed);
}

/// Adds an attribute included on every subsequent log line, like
/// logger.AddBaseAttrs in Go.
pub fn add_base_attr(key: &str, value: &str) {
    base_attrs()
        .lock()
        .unwrap()
        .push((key.to_string(), value.to_string()));
}

pub fn unix_millis() -> i64 {
    SystemTime::now()
        .duration_since(UNIX_EPOCH)
        .map(|d| d.as_millis() as i64)
        .unwrap_or(0)
}

pub fn log(level: Level, msg: &str, fields: &[(&str, &str)]) {
    if (level as u8) < MIN_LEVEL.load(Ordering::Relaxed) {
        return;
    }
    let mut line = String::with_capacity(128);
    line.push('{');
    push_json_kv(&mut line, "time", &unix_millis().to_string());
    line.push(',');
    push_json_str_kv(&mut line, "level", level.as_str());
    line.push(',');
    push_json_str_kv(&mut line, "msg", msg);
    for (k, v) in base_attrs().lock().unwrap().iter() {
        line.push(',');
        push_json_str_kv(&mut line, k, v);
    }
    for (k, v) in fields {
        line.push(',');
        push_json_str_kv(&mut line, k, v);
    }
    line.push('}');
    line.push('\n');

    let stdout = std::io::stdout();
    let _ = stdout.lock().write_all(line.as_bytes());
}

fn push_json_kv(out: &mut String, key: &str, raw_value: &str) {
    out.push('"');
    out.push_str(key);
    out.push_str("\":");
    out.push_str(raw_value);
}

fn push_json_str_kv(out: &mut String, key: &str, value: &str) {
    out.push_str(&serde_json::to_string(key).unwrap_or_else(|_| "\"?\"".into()));
    out.push(':');
    out.push_str(&serde_json::to_string(value).unwrap_or_else(|_| "\"?\"".into()));
}

#[macro_export]
macro_rules! log_debug {
    ($msg:expr $(, $k:expr => $v:expr)* $(,)?) => {
        $crate::logger::log($crate::logger::Level::Debug, $msg, &[$(($k, &*($v).to_string())),*])
    };
}

#[macro_export]
macro_rules! log_info {
    ($msg:expr $(, $k:expr => $v:expr)* $(,)?) => {
        $crate::logger::log($crate::logger::Level::Info, $msg, &[$(($k, &*($v).to_string())),*])
    };
}

#[macro_export]
macro_rules! log_warn {
    ($msg:expr $(, $k:expr => $v:expr)* $(,)?) => {
        $crate::logger::log($crate::logger::Level::Warn, $msg, &[$(($k, &*($v).to_string())),*])
    };
}

#[macro_export]
macro_rules! log_error {
    ($msg:expr $(, $k:expr => $v:expr)* $(,)?) => {
        $crate::logger::log($crate::logger::Level::Error, $msg, &[$(($k, &*($v).to_string())),*])
    };
}
