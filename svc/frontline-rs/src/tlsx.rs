//! TLS helpers: outbound client connections (peer frontlines, vault, ctrl,
//! ClickHouse over https) and the inbound server config with dynamic
//! certificate resolution (port of tls.go + certmanager wiring).

use std::io;
use std::sync::Arc;
use std::time::Duration;

use rustls::pki_types::ServerName;
use rustls::{ClientConfig, ClientConnection, RootCertStore, StreamOwned};

use crate::httpx::{dial, Stream};

/// Client config trusting the standard web PKI roots.
pub fn client_config() -> Arc<ClientConfig> {
    static CONFIG: std::sync::OnceLock<Arc<ClientConfig>> = std::sync::OnceLock::new();
    CONFIG
        .get_or_init(|| {
            let mut roots = RootCertStore::empty();
            roots.extend(webpki_roots::TLS_SERVER_ROOTS.iter().cloned());
            Arc::new(
                ClientConfig::builder()
                    .with_root_certificates(roots)
                    .with_no_client_auth(),
            )
        })
        .clone()
}

/// Dials host:port and performs a TLS handshake with SNI = host.
/// The dial phase and the TLS phase are deliberately separate so dial
/// failures keep their retry-safety semantics.
pub fn connect_tls(host: &str, port: u16, timeout: Duration) -> io::Result<Box<dyn Stream>> {
    let tcp = dial(&format!("{host}:{port}"), timeout)?;
    tcp.set_read_timeout(Some(Duration::from_secs(30)))?;
    let server_name = ServerName::try_from(host.to_string())
        .map_err(|e| io::Error::new(io::ErrorKind::InvalidInput, format!("bad SNI name: {e}")))?;
    let conn = ClientConnection::new(client_config(), server_name)
        .map_err(|e| io::Error::new(io::ErrorKind::Other, format!("tls client: {e}")))?;
    Ok(Box::new(StreamOwned::new(conn, tcp)))
}

/// Minimal URL splitter for the handful of endpoints frontline dials
/// (vault, ctrl, clickhouse). Supports scheme://user:pass@host:port/path.
#[derive(Debug, Clone, PartialEq)]
pub struct SimpleUrl {
    pub scheme: String,
    pub username: String,
    pub password: String,
    pub host: String,
    pub port: u16,
    pub path: String,
}

impl SimpleUrl {
    pub fn parse(input: &str) -> Result<Self, String> {
        let (scheme, rest) = input.split_once("://").unwrap_or(("http", input));
        let scheme = scheme.to_ascii_lowercase();
        let (authority, path) = match rest.find('/') {
            Some(i) => (&rest[..i], rest[i..].to_string()),
            None => (rest, "/".to_string()),
        };
        let (userinfo, hostport) = match authority.rsplit_once('@') {
            Some((u, h)) => (u, h),
            None => ("", authority),
        };
        let (username, password) = match userinfo.split_once(':') {
            Some((u, p)) => (u.to_string(), p.to_string()),
            None => (userinfo.to_string(), String::new()),
        };
        let (host, port) = match hostport.rsplit_once(':') {
            Some((h, p)) if p.chars().all(|c| c.is_ascii_digit()) && !p.is_empty() => (
                h.to_string(),
                p.parse::<u16>().map_err(|e| format!("bad port: {e}"))?,
            ),
            _ => (
                hostport.to_string(),
                if scheme == "https" { 443 } else { 80 },
            ),
        };
        if host.is_empty() {
            return Err(format!("no host in url {input:?}"));
        }
        Ok(Self {
            scheme,
            username: crate::encoding::percent_decode(&username),
            password: crate::encoding::percent_decode(&password),
            host,
            port,
            path,
        })
    }

    pub fn is_tls(&self) -> bool {
        self.scheme == "https"
    }

    /// Opens a connection to this URL's host/port, TLS when https.
    pub fn connect(&self, timeout: Duration) -> io::Result<Box<dyn Stream>> {
        if self.is_tls() {
            connect_tls(&self.host, self.port, timeout)
        } else {
            let tcp = dial(&format!("{}:{}", self.host, self.port), timeout)?;
            tcp.set_read_timeout(Some(Duration::from_secs(30)))?;
            Ok(Box::new(tcp))
        }
    }

    pub fn host_header(&self) -> String {
        let default_port = if self.is_tls() { 443 } else { 80 };
        if self.port == default_port {
            self.host.clone()
        } else {
            format!("{}:{}", self.host, self.port)
        }
    }
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn parses_full_url() {
        let u = SimpleUrl::parse("https://user:p%40ss@ch.example.com:8443/db").unwrap();
        assert_eq!(u.scheme, "https");
        assert_eq!(u.username, "user");
        assert_eq!(u.password, "p@ss");
        assert_eq!(u.host, "ch.example.com");
        assert_eq!(u.port, 8443);
        assert_eq!(u.path, "/db");
        assert!(u.is_tls());
    }

    #[test]
    fn defaults_scheme_ports_and_path() {
        let u = SimpleUrl::parse("http://localhost").unwrap();
        assert_eq!(u.port, 80);
        assert_eq!(u.path, "/");
        let u = SimpleUrl::parse("https://vault.internal").unwrap();
        assert_eq!(u.port, 443);
        // Scheme-less host:port (e.g. ctrl_addr = "localhost:8080").
        let u = SimpleUrl::parse("localhost:8080").unwrap();
        assert_eq!(u.scheme, "http");
        assert_eq!(u.host, "localhost");
        assert_eq!(u.port, 8080);
    }

    #[test]
    fn host_header_omits_default_port() {
        assert_eq!(
            SimpleUrl::parse("https://a.example.com")
                .unwrap()
                .host_header(),
            "a.example.com"
        );
        assert_eq!(
            SimpleUrl::parse("http://a.example.com:8080")
                .unwrap()
                .host_header(),
            "a.example.com:8080"
        );
    }
}
