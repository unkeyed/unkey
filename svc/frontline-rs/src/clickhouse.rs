//! Port of pkg/clickhouse's buffered inserts, over ClickHouse's HTTP
//! interface (JSONEachRow) using the hand-rolled HTTP client.
//!
//! Rows are buffered on a bounded channel; when the buffer is full new rows
//! are silently dropped (Drop: true in the Go config — analytics must never
//! block or break the request path). Consumer threads flush on batch size
//! or every 5 seconds.

use std::sync::mpsc::{sync_channel, Receiver, RecvTimeoutError, SyncSender};
use std::sync::{Arc, Mutex};
use std::time::Duration;

use serde::Serialize;

use crate::encoding::base64_encode;
use crate::httpx::{copy_body, Conn, Headers};
use crate::session::RequestTracking;
use crate::tlsx::SimpleUrl;

const FLUSH_INTERVAL: Duration = Duration::from_secs(5);

#[derive(Clone)]
pub struct ClickHouseClient {
    url: SimpleUrl,
}

impl ClickHouseClient {
    pub fn new(url: &str) -> Result<Self, String> {
        Ok(Self {
            url: SimpleUrl::parse(url)?,
        })
    }

    /// POSTs a batch of JSONEachRow lines into the given table.
    fn insert(&self, table: &str, rows: &[String]) -> Result<(), String> {
        let body = rows.join("\n");
        let query = format!("INSERT INTO {table} FORMAT JSONEachRow");
        let target = format!("{}?query={}", self.url.path, percent_encode_query(&query));

        let stream = self
            .url
            .connect(Duration::from_secs(10))
            .map_err(|e| format!("clickhouse connect: {e}"))?;
        let mut conn = Conn::new(stream);

        let mut headers = Headers::new();
        headers.set("Host", &self.url.host_header());
        headers.set("Content-Type", "application/x-ndjson");
        headers.set("Content-Length", &body.len().to_string());
        headers.set("Connection", "close");
        if !self.url.username.is_empty() {
            let credentials = format!("{}:{}", self.url.username, self.url.password);
            headers.set(
                "Authorization",
                &format!("Basic {}", base64_encode(credentials.as_bytes())),
            );
        }

        {
            let out = conn.stream_mut();
            crate::httpx::write_request_head(out, "POST", &target, &headers)
                .map_err(|e| format!("clickhouse write: {e}"))?;
            out.write_all(body.as_bytes())
                .map_err(|e| format!("clickhouse write: {e}"))?;
            out.flush().map_err(|e| format!("clickhouse write: {e}"))?;
        }

        let resp = conn
            .read_response_head("POST")
            .map_err(|e| format!("clickhouse response: {e}"))?;
        let mut response_body = Vec::new();
        let _ = copy_body(&mut conn, resp.body, Some(&mut response_body));
        if resp.status != 200 {
            return Err(format!(
                "clickhouse insert failed: status {} body {}",
                resp.status,
                String::from_utf8_lossy(&response_body[..response_body.len().min(512)])
            ));
        }
        Ok(())
    }
}

fn percent_encode_query(s: &str) -> String {
    let mut out = String::with_capacity(s.len() * 3 / 2);
    for b in s.bytes() {
        match b {
            b'A'..=b'Z' | b'a'..=b'z' | b'0'..=b'9' | b'-' | b'_' | b'.' | b'~' => {
                out.push(b as char)
            }
            _ => out.push_str(&format!("%{b:02X}")),
        }
    }
    out
}

/// Buffered, drop-on-overflow sink for one ClickHouse table. The no-op
/// variant (no ClickHouse configured) accepts and discards rows.
pub struct Buffer<T> {
    tx: Option<SyncSender<T>>,
}

impl<T: Serialize + Send + 'static> Buffer<T> {
    pub fn noop() -> Self {
        Self { tx: None }
    }

    pub fn new(
        client: ClickHouseClient,
        table: &str,
        batch_size: usize,
        buffer_size: usize,
        consumers: usize,
    ) -> Self {
        let (tx, rx) = sync_channel::<T>(buffer_size.max(1));
        let rx = Arc::new(Mutex::new(rx));
        for _ in 0..consumers.max(1) {
            let rx = Arc::clone(&rx);
            let client = client.clone();
            let table = table.to_string();
            let batch_size = batch_size.max(1);
            std::thread::Builder::new()
                .name(format!("clickhouse-{table}"))
                .spawn(move || consume(rx, client, table, batch_size))
                .expect("spawn clickhouse consumer");
        }
        Self { tx: Some(tx) }
    }

    /// Buffers a row; drops it silently when the buffer is full.
    pub fn buffer(&self, item: T) {
        if let Some(tx) = &self.tx {
            let _ = tx.try_send(item);
        }
    }
}

fn consume<T: Serialize>(
    rx: Arc<Mutex<Receiver<T>>>,
    client: ClickHouseClient,
    table: String,
    batch_size: usize,
) {
    let mut batch: Vec<String> = Vec::with_capacity(batch_size);
    loop {
        let item = {
            let rx = rx.lock().unwrap();
            rx.recv_timeout(FLUSH_INTERVAL)
        };
        match item {
            Ok(item) => {
                if let Ok(line) = serde_json::to_string(&item) {
                    batch.push(line);
                }
                if batch.len() >= batch_size {
                    flush(&client, &table, &mut batch);
                }
            }
            Err(RecvTimeoutError::Timeout) => flush(&client, &table, &mut batch),
            Err(RecvTimeoutError::Disconnected) => {
                flush(&client, &table, &mut batch);
                return;
            }
        }
    }
}

fn flush(client: &ClickHouseClient, table: &str, batch: &mut Vec<String>) {
    if batch.is_empty() {
        return;
    }
    if let Err(err) = client.insert(table, batch) {
        crate::log_error!("clickhouse flush failed", "table" => table, "error" => err);
    }
    batch.clear();
}

/// Row schema of default.sentinel_requests_raw_v1 (pkg/clickhouse/schema's
/// SentinelRequest). Field names are the ClickHouse column names.
#[derive(Debug, Clone, Serialize, Default)]
pub struct SentinelRequest {
    pub request_id: String,
    pub time: i64,
    pub workspace_id: String,
    pub environment_id: String,
    pub project_id: String,
    pub sentinel_id: String,
    pub deployment_id: String,
    pub instance_id: String,
    pub instance_address: String,
    pub region: String,
    pub platform: String,
    pub method: String,
    pub host: String,
    pub path: String,
    pub query_string: String,
    pub query_params: std::collections::HashMap<String, Vec<String>>,
    pub request_headers: Vec<String>,
    pub request_body: String,
    pub response_status: i32,
    pub response_headers: Vec<String>,
    pub response_body: String,
    pub user_agent: String,
    pub ip_address: String,
    pub total_latency: i64,
    pub instance_latency: i64,
    pub sentinel_latency: i64,
}

/// Builds the SentinelRequest row from tracking (which snapshots the request
/// metadata) and the final response status/headers. Port of the row assembly
/// in middleware/clickhouse_logging.go.
pub fn build_sentinel_request(
    tracking: &RequestTracking,
    frontline_id: &str,
    region: &str,
    platform: &str,
    host: &str,
    resp_status: u16,
    response_headers: Vec<String>,
    ip_address: &str,
    end_unix_ms: i64,
) -> SentinelRequest {
    let total_latency = end_unix_ms - tracking.start_unix_ms;
    let (instance_latency, frontline_latency) =
        if tracking.instance_start_ms > 0 && tracking.instance_end_ms > 0 {
            let il = tracking.instance_end_ms - tracking.instance_start_ms;
            (il, total_latency - il)
        } else {
            (0, 0)
        };

    let mut query_params: std::collections::HashMap<String, Vec<String>> =
        std::collections::HashMap::new();
    for (k, v) in crate::encoding::parse_query(&tracking.raw_query) {
        query_params.entry(k).or_default().push(v);
    }

    SentinelRequest {
        request_id: tracking.request_id.clone(),
        time: tracking.start_unix_ms,
        workspace_id: tracking.workspace_id.clone(),
        environment_id: tracking.environment_id.clone(),
        project_id: tracking.project_id.clone(),
        sentinel_id: frontline_id.to_string(),
        deployment_id: tracking.deployment_id.clone(),
        instance_id: tracking.instance_id.clone(),
        instance_address: tracking.address.clone(),
        region: region.to_string(),
        platform: platform.to_string(),
        method: tracking.method.to_ascii_uppercase(),
        host: host.to_string(),
        path: tracking.path.clone(),
        query_string: tracking.raw_query.clone(),
        query_params,
        request_headers: tracking.request_headers.clone(),
        request_body: String::from_utf8_lossy(&tracking.request_body).into_owned(),
        response_status: resp_status as i32,
        response_headers,
        response_body: String::from_utf8_lossy(&tracking.response_body).into_owned(),
        user_agent: tracking.user_agent.clone(),
        ip_address: ip_address.to_string(),
        total_latency,
        instance_latency,
        sentinel_latency: frontline_latency,
    }
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn noop_buffer_accepts_rows() {
        let b: Buffer<SentinelRequest> = Buffer::noop();
        b.buffer(SentinelRequest::default());
    }

    #[test]
    fn builds_row_with_latencies() {
        let tracking = RequestTracking {
            request_id: "req_1".into(),
            start_unix_ms: 1000,
            deployment_id: "dep_1".into(),
            instance_id: "ins_1".into(),
            instance_start_ms: 1010,
            instance_end_ms: 1060,
            method: "post".into(),
            path: "/v1/keys".into(),
            raw_query: "a=1&a=2".into(),
            request_body: b"hello".to_vec(),
            user_agent: "curl".into(),
            ..Default::default()
        };
        let row = build_sentinel_request(
            &tracking,
            "fl_1",
            "us-east-1",
            "aws",
            "api.example.com",
            200,
            vec!["Content-Type: application/json".into()],
            "203.0.113.7",
            1100,
        );
        assert_eq!(row.method, "POST");
        assert_eq!(row.total_latency, 100);
        assert_eq!(row.instance_latency, 50);
        assert_eq!(row.sentinel_latency, 50);
        assert_eq!(row.query_params.get("a").unwrap().len(), 2);
        assert_eq!(row.request_body, "hello");
        assert_eq!(row.user_agent, "curl");
    }

    #[test]
    fn percent_encodes_insert_query() {
        assert_eq!(
            percent_encode_query("INSERT INTO t FORMAT JSONEachRow"),
            "INSERT%20INTO%20t%20FORMAT%20JSONEachRow"
        );
    }
}
