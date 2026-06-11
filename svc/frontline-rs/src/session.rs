//! Per-request tracking state (port of proxy.RequestTracking) plus host/
//! address helpers shared by the gateway.

/// Data collected during a local-instance request for ClickHouse logging.
/// Port of proxy.RequestTracking. Cross-region requests do not populate the
/// deployment fields — the peer frontline writes its own row.
#[derive(Debug, Default, Clone)]
pub struct RequestTracking {
    pub request_id: String,
    pub start_unix_ms: i64,
    pub deployment_id: String,
    pub workspace_id: String,
    pub environment_id: String,
    pub project_id: String,

    /// (Re)set before each instance attempt in the retry loop, so on success
    /// they reflect the instance that served the response.
    pub instance_id: String,
    pub address: String,

    pub request_body: Vec<u8>,
    pub instance_start_ms: i64,
    pub instance_end_ms: i64,
    pub response_body: Vec<u8>,

    /// Request metadata snapshotted at arrival so the deferred ClickHouse
    /// emission (after the response stream completes) doesn't need the
    /// original request.
    pub method: String,
    pub path: String,
    pub raw_query: String,
    pub request_headers: Vec<String>,
    pub user_agent: String,
    pub accept: String,
}

/// Removes the port from an address. Handles IPv4 (1.2.3.4:80), bracketed
/// IPv6 ([::1]:8080), and plain addresses. Port of zen.stripPort /
/// proxy.ExtractHostname.
pub fn strip_port(addr: &str) -> String {
    if let Some(rest) = addr.strip_prefix('[') {
        // Bracketed IPv6, possibly with a port.
        if let Some(end) = rest.find(']') {
            return rest[..end].to_string();
        }
        return addr.to_string();
    }
    // A single colon separates host:port; multiple colons mean a bare IPv6
    // address, which has no port to strip.
    let colons = addr.matches(':').count();
    if colons == 1 {
        if let Some((host, port)) = addr.rsplit_once(':') {
            if !port.is_empty() && port.chars().all(|c| c.is_ascii_digit()) {
                return host.to_string();
            }
        }
    }
    addr.to_string()
}

/// Extracts the hostname from a Host header value, stripping any port.
/// Port of proxy.ExtractHostname.
pub fn extract_hostname(host: &str) -> String {
    strip_port(host)
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn extract_hostname_strips_ports() {
        assert_eq!(extract_hostname("example.com:443"), "example.com");
        assert_eq!(extract_hostname("example.com"), "example.com");
        assert_eq!(extract_hostname("[::1]:8080"), "::1");
        assert_eq!(extract_hostname("192.168.1.1:80"), "192.168.1.1");
        assert_eq!(extract_hostname("::1"), "::1");
    }
}
