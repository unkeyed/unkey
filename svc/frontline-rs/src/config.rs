//! Port of config.go: TOML configuration for the frontline server.

use serde::{Deserialize, Deserializer};
use std::time::Duration;

/// Parses Go-style duration strings ("15m", "1h30m", "500ms", "10s") so
/// config files stay compatible with the Go service. Plain integers are
/// treated as seconds.
pub fn parse_duration(s: &str) -> Result<Duration, String> {
    let s = s.trim();
    if s.is_empty() {
        return Err("empty duration".into());
    }
    if let Ok(secs) = s.parse::<u64>() {
        return Ok(Duration::from_secs(secs));
    }
    let mut total = Duration::ZERO;
    let mut num = String::new();
    let mut chars = s.chars().peekable();
    while let Some(c) = chars.next() {
        if c.is_ascii_digit() || c == '.' {
            num.push(c);
            continue;
        }
        let mut unit = String::from(c);
        // Two-letter units: ms, us, ns.
        if let Some(&next) = chars.peek() {
            if next == 's' && (c == 'm' || c == 'u' || c == 'n') {
                unit.push(chars.next().unwrap());
            }
        }
        let value: f64 = num
            .parse()
            .map_err(|_| format!("bad duration number in {s:?}"))?;
        num.clear();
        let mult = match unit.as_str() {
            "ns" => 1e-9,
            "us" => 1e-6,
            "ms" => 1e-3,
            "s" => 1.0,
            "m" => 60.0,
            "h" => 3600.0,
            "d" => 86400.0,
            _ => return Err(format!("unknown duration unit {unit:?} in {s:?}")),
        };
        total += Duration::from_secs_f64(value * mult);
    }
    if !num.is_empty() {
        return Err(format!("duration {s:?} ends without a unit"));
    }
    Ok(total)
}

fn de_duration<'de, D: Deserializer<'de>>(d: D) -> Result<Duration, D::Error> {
    let s = String::deserialize(d)?;
    parse_duration(&s).map_err(serde::de::Error::custom)
}

fn de_opt_duration<'de, D: Deserializer<'de>>(d: D) -> Result<Option<Duration>, D::Error> {
    let s = Option::<String>::deserialize(d)?;
    s.map(|s| parse_duration(&s).map_err(serde::de::Error::custom))
        .transpose()
}

fn default_http_port() -> u16 {
    7070
}
fn default_https_port() -> u16 {
    7443
}
fn default_apex_domain() -> String {
    "unkey.cloud".to_string()
}
fn default_max_hops() -> u32 {
    10
}
fn default_ctrl_addr() -> String {
    "localhost:8080".to_string()
}
fn default_request_timeout() -> Duration {
    Duration::from_secs(15 * 60)
}
fn default_batch_size() -> usize {
    5000
}
fn default_buffer_size() -> usize {
    10_000
}
fn default_consumers() -> usize {
    1
}

#[derive(Debug, Clone, Deserialize, Default)]
#[serde(deny_unknown_fields)]
pub struct ClickHouseConfig {
    /// ClickHouse HTTP endpoint, e.g. "http://default:password@localhost:8123".
    /// When empty, analytics are dropped (no-op buffers). Note: the Go service
    /// uses the native protocol; this port uses the HTTP interface.
    #[serde(default)]
    pub url: String,
    #[serde(default = "default_batch_size")]
    pub batch_size: usize,
    #[serde(default = "default_buffer_size")]
    pub buffer_size: usize,
    #[serde(default = "default_consumers")]
    pub consumers: usize,
}

#[derive(Debug, Clone, Deserialize, Default)]
#[serde(deny_unknown_fields)]
pub struct TlsConfig {
    #[serde(default)]
    pub cert_file: String,
    #[serde(default)]
    pub key_file: String,
    #[serde(default)]
    pub disabled: bool,
}

#[derive(Debug, Clone, Deserialize, Default)]
#[serde(deny_unknown_fields)]
pub struct DatabaseConfig {
    /// Primary MySQL DSN, e.g. "mysql://user:pass@host:3306/unkey".
    pub primary: String,
    /// Read-only replica DSN. Falls back to primary when empty.
    #[serde(default)]
    pub readonly_replica: String,
}

#[derive(Debug, Clone, Deserialize, Default)]
#[serde(deny_unknown_fields)]
pub struct VaultConfig {
    #[serde(default)]
    pub url: String,
    #[serde(default)]
    pub token: String,
}

/// Accepted for config-file parity with the Go service. The policy engine
/// is out of scope in this port, so Redis is never used; a configured URL
/// is ignored with a startup warning.
#[derive(Debug, Clone, Deserialize, Default)]
#[serde(deny_unknown_fields)]
pub struct RedisConfig {
    #[serde(default)]
    pub url: String,
}

/// Accepted for config-file parity with the Go service. pprof is a Go
/// runtime facility; this port ignores it with a startup warning.
#[derive(Debug, Clone, Deserialize, Default)]
#[serde(deny_unknown_fields)]
#[allow(dead_code)]
pub struct PprofConfig {
    #[serde(default)]
    pub username: String,
    #[serde(default)]
    pub password: String,
    #[serde(default)]
    pub port: u16,
}

/// Observability settings are accepted for config-file compatibility with
/// the Go service; this port logs everything at the configured level and
/// does not implement tail sampling or OTLP tracing.
#[derive(Debug, Clone, Deserialize, Default)]
#[serde(deny_unknown_fields)]
#[allow(dead_code)]
pub struct LoggingConfig {
    #[serde(default, deserialize_with = "de_opt_duration")]
    pub slow_threshold: Option<Duration>,
    #[serde(default)]
    pub sample_rate: Option<f64>,
}

#[derive(Debug, Clone, Deserialize, Default)]
#[serde(deny_unknown_fields)]
#[allow(dead_code)]
pub struct TracingConfig {
    #[serde(default)]
    pub sample_rate: Option<f64>,
}

#[derive(Debug, Clone, Deserialize, Default)]
#[serde(deny_unknown_fields)]
#[allow(dead_code)]
pub struct Observability {
    #[serde(default)]
    pub logging: Option<LoggingConfig>,
    #[serde(default)]
    pub tracing: Option<TracingConfig>,
}

/// Complete configuration for the frontline server, loaded from TOML.
#[derive(Debug, Clone, Deserialize)]
#[serde(deny_unknown_fields)]
pub struct Config {
    #[serde(default)]
    pub instance_id: String,

    /// Plain-HTTP listener: ACME HTTP-01 challenges + 308 redirects to https.
    #[serde(default = "default_http_port")]
    pub http_port: u16,

    /// HTTPS frontline listener: TLS termination, policy engine, forwarding.
    #[serde(default = "default_https_port")]
    pub https_port: u16,

    /// Cloud provider: aws, gcp, local.
    pub platform: String,

    /// Geographic region of this node.
    pub region: String,

    /// Cross-region requests are forwarded to frontline.{region}.{apex_domain}.
    #[serde(default = "default_apex_domain")]
    pub apex_domain: String,

    /// Maximum frontline hops before rejecting a request (loop prevention).
    #[serde(default = "default_max_hops")]
    pub max_hops: u32,

    /// Control plane address for ACME challenge verification.
    #[serde(default = "default_ctrl_addr")]
    pub ctrl_addr: String,

    /// Prometheus /metrics port. 0 disables the metrics listener.
    #[serde(default)]
    pub prometheus_port: u16,

    #[serde(default)]
    pub tls: Option<TlsConfig>,

    pub database: DatabaseConfig,

    #[serde(default)]
    pub clickhouse: ClickHouseConfig,

    /// Maximum duration for proxied requests before a 504 is returned.
    #[serde(default = "default_request_timeout", deserialize_with = "de_duration")]
    pub request_timeout: Duration,

    #[allow(dead_code)]
    #[serde(default)]
    pub observability: Observability,

    #[serde(default)]
    pub vault: VaultConfig,

    /// Parsed but unused; see [RedisConfig].
    #[serde(default)]
    pub redis: RedisConfig,

    /// Parsed but unused; see [PprofConfig].
    #[serde(default)]
    pub pprof: Option<PprofConfig>,
}

impl Config {
    /// Cross-field validation that struct-level defaults cannot express:
    /// TLS must be fully configured (both cert and key) or explicitly
    /// disabled. Port of Config.Validate in config.go.
    pub fn validate(&self) -> Result<(), String> {
        if self.platform.is_empty() {
            return Err("platform is required".into());
        }
        if self.region.is_empty() {
            return Err("region is required".into());
        }
        if let Some(tls) = &self.tls {
            if !tls.disabled && (tls.cert_file.is_empty() != tls.key_file.is_empty()) {
                return Err(
                    "both tls.cert_file and tls.key_file must be provided together when TLS is not disabled"
                        .into(),
                );
            }
        }
        Ok(())
    }

    pub fn load(path: &str) -> Result<Self, String> {
        let raw = std::fs::read_to_string(path)
            .map_err(|e| format!("unable to read config file {path}: {e}"))?;
        let cfg: Config =
            toml::from_str(&raw).map_err(|e| format!("unable to parse config file {path}: {e}"))?;
        cfg.validate()?;
        Ok(cfg)
    }
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn minimal_config_with_defaults() {
        let cfg: Config = toml::from_str(
            r#"
            platform = "local"
            region = "dev"
            [database]
            primary = "mysql://root@localhost:3306/unkey"
            "#,
        )
        .unwrap();
        assert_eq!(cfg.http_port, 7070);
        assert_eq!(cfg.https_port, 7443);
        assert_eq!(cfg.apex_domain, "unkey.cloud");
        assert_eq!(cfg.max_hops, 10);
        assert_eq!(cfg.request_timeout, Duration::from_secs(900));
        cfg.validate().unwrap();
    }

    #[test]
    fn duration_parsing() {
        assert_eq!(parse_duration("15m").unwrap(), Duration::from_secs(900));
        assert_eq!(parse_duration("1h30m").unwrap(), Duration::from_secs(5400));
        assert_eq!(parse_duration("500ms").unwrap(), Duration::from_millis(500));
        assert_eq!(parse_duration("10").unwrap(), Duration::from_secs(10));
        assert!(parse_duration("10x").is_err());
        assert!(parse_duration("").is_err());
    }

    #[test]
    fn parses_the_dev_k8s_configmap() {
        // Mirror of dev/k8s/manifests/frontline.yaml's ConfigMap. The Go
        // sections this port doesn't use ([redis], [pprof], observability)
        // must parse without error.
        let cfg: Config = toml::from_str(
            r#"
            platform = "dev"
            region = "local"
            http_port = 7070
            https_port = 7443
            apex_domain = "unkey.local"
            ctrl_addr = "http://ctrl-api.unkey.svc.cluster.local:7091"

            [database]
            primary = "unkey:password@tcp(mysql.unkey.svc.cluster.local:3306)/unkey?parseTime=true&interpolateParams=true"

            [redis]
            url = "redis://default:password@redis.unkey.svc.cluster.local:6379"

            [clickhouse]
            url = "clickhouse://default:password@clickhouse.unkey.svc.cluster.local:9000?secure=false&skip_verify=true"

            [tls]
            cert_file = "/certs/unkey.local.crt"
            key_file = "/certs/unkey.local.key"

            [vault]
            url = "http://vault.unkey.svc.cluster.local:8060"
            token = "vault-test-token-123"

            [pprof]
            username = "admin"
            password = "password"
            "#,
        )
        .unwrap();
        cfg.validate().unwrap();
        assert_eq!(cfg.apex_domain, "unkey.local");
        assert!(!cfg.redis.url.is_empty());
        assert!(cfg.pprof.is_some());
    }

    #[test]
    fn request_timeout_accepts_go_durations() {
        let cfg: Config = toml::from_str(
            r#"
            platform = "local"
            region = "dev"
            request_timeout = "2m30s"
            [database]
            primary = "mysql://root@localhost:3306/unkey"
            "#,
        )
        .unwrap();
        assert_eq!(cfg.request_timeout, Duration::from_secs(150));
    }

    #[test]
    fn partial_tls_is_rejected() {
        let cfg: Config = toml::from_str(
            r#"
            platform = "local"
            region = "dev"
            [database]
            primary = "mysql://root@localhost:3306/unkey"
            [tls]
            cert_file = "/tmp/cert.pem"
            "#,
        )
        .unwrap();
        assert!(cfg.validate().is_err());
    }
}
