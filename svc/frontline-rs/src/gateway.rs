//! The async request hot path on tokio + hyper: TLS termination, middleware
//! (reserved-header strip, observability, ClickHouse logging), the catchall
//! proxy with dial-failure retry, ACME relay, redirects, and health.
//!
//! Replaces the original hand-rolled thread-per-connection HTTP stack, which
//! benchmarks showed cost throughput and correctness under load (see
//! BENCHMARKS.md). Upstream pooling, connect-error classification, and
//! upgrade handling now come from hyper's client.

use std::net::SocketAddr;
use std::pin::Pin;
use std::sync::{Arc, Mutex};
use std::task::{Context, Poll};
use std::time::{Duration, Instant};

use bytes::Bytes;
use http::header::{HeaderMap, HeaderName, HeaderValue};
use http::uri::Uri;
use http::{Method, Request, Response, StatusCode};
use http_body_util::combinators::BoxBody;
use http_body_util::{BodyExt, Full};
use hyper::body::{Body as HttpBody, Frame, Incoming, SizeHint};
use hyper_util::client::legacy::connect::HttpConnector;
use hyper_util::client::legacy::Client as LegacyClient;
use hyper_util::rt::TokioExecutor;

use crate::clickhouse::{Buffer, SentinelRequest};
use crate::config::Config;
use crate::connectrpc::AcmeClient;
use crate::db::{Database, InstanceRow};
use crate::error::{error_page_info, urn, FrontlineError};
use crate::errorpage;
use crate::httpx::MAX_BODY_CAPTURE;
use crate::logger::unix_millis;
use crate::metrics;
use crate::router::{Destination, Router};
use crate::session::{extract_hostname, RequestTracking};
use crate::uid;

pub const HEADER_FRONTLINE_ID: &str = "x-unkey-frontline-id";
pub const HEADER_REGION: &str = "x-unkey-region";
pub const HEADER_REQUEST_ID: &str = "x-unkey-request-id";
pub const HEADER_PARENT_FRONTLINE_ID: &str = "x-unkey-parent-frontline-id";
pub const HEADER_PARENT_REQUEST_ID: &str = "x-unkey-parent-request-id";
pub const HEADER_FRONTLINE_HOPS: &str = "x-unkey-frontline-hops";
pub const HEADER_LANGUAGE: &str = "x-unkey-frontline-language";
/// Strict timing header (pkg/timing): name{scope=frontline}=12.3ms.
pub const HEADER_TIMING: &str = "x-unkey-timing";

const ACME_CHALLENGE_PREFIX: &str = "/.well-known/acme-challenge/";
const HEALTH_PATH: &str = "/_unkey/internal/health";
const INTERNAL_PREFIX: &str = "/_unkey/internal/";
const RESERVED_HEADER_PREFIX: &str = "x-unkey-";

const DESTINATION_INSTANCE: &str = "instance";
const DESTINATION_FRONTLINE: &str = "frontline";

pub type OutBody = BoxBody<Bytes, hyper::Error>;

fn full_body(data: impl Into<Bytes>) -> OutBody {
    Full::new(data.into())
        .map_err(|never| match never {})
        .boxed()
}

/// Shared application state (routes.Services + the hyper clients).
pub struct Gateway {
    pub cfg: Config,
    pub frontline_id: String,
    pub router: Arc<Router>,
    pub db: Database,
    pub acme: Option<Arc<AcmeClient>>,
    pub frontline_requests: Buffer<SentinelRequest>,
    /// Pooled client for instance forwards (plain HTTP, like Go's transport
    /// registry: MaxIdleConnsPerHost=50, IdleConnTimeout=90s).
    pub instance_client: LegacyClient<HttpConnector, ReplayBody>,
    /// Pooled HTTPS client for peer-frontline forwards.
    pub region_client: LegacyClient<hyper_rustls::HttpsConnector<HttpConnector>, ReplayBody>,
}

pub fn build_clients() -> (
    LegacyClient<HttpConnector, ReplayBody>,
    LegacyClient<hyper_rustls::HttpsConnector<HttpConnector>, ReplayBody>,
) {
    let mut http = HttpConnector::new();
    http.set_nodelay(true);
    http.set_connect_timeout(Some(Duration::from_secs(10)));
    http.set_keepalive(Some(Duration::from_secs(30)));

    let instance_client = LegacyClient::builder(TokioExecutor::new())
        .pool_idle_timeout(Duration::from_secs(90))
        .pool_max_idle_per_host(50)
        .build(http.clone());

    let https = hyper_rustls::HttpsConnectorBuilder::new()
        .with_webpki_roots()
        .https_only()
        .enable_http1()
        .wrap_connector(http);
    let region_client = LegacyClient::builder(TokioExecutor::new())
        .pool_idle_timeout(Duration::from_secs(90))
        .pool_max_idle_per_host(100)
        .build(https);

    (instance_client, region_client)
}

// ---------------------------------------------------------------------------
// Bodies: replayable request body (dial-retry) + tail-captured response body
// ---------------------------------------------------------------------------

struct ReplayState {
    incoming: Option<Incoming>,
    capture: Vec<u8>,
    consumed: bool,
}

/// Replayable wrapper around the inbound request body. Each proxy attempt
/// gets a fresh handle sharing the same underlying body: hyper only polls
/// the body after the connection is established, so an attempt that failed
/// to connect leaves the body untouched and the next attempt can take over.
/// This is what makes the dial-failure retry loop safe, mirroring Go where
/// the ReverseProxy never reads the body when the dial fails.
#[derive(Clone)]
pub struct ReplayBody {
    state: Arc<Mutex<ReplayState>>,
}

impl ReplayBody {
    pub fn new(incoming: Incoming) -> Self {
        Self {
            state: Arc::new(Mutex::new(ReplayState {
                incoming: Some(incoming),
                capture: Vec::new(),
                consumed: false,
            })),
        }
    }

    fn consumed(&self) -> bool {
        self.state.lock().unwrap().consumed
    }

    fn take_capture(&self) -> Vec<u8> {
        std::mem::take(&mut self.state.lock().unwrap().capture)
    }
}

impl HttpBody for ReplayBody {
    type Data = Bytes;
    type Error = hyper::Error;

    fn poll_frame(
        self: Pin<&mut Self>,
        cx: &mut Context<'_>,
    ) -> Poll<Option<Result<Frame<Self::Data>, Self::Error>>> {
        let mut st = self.state.lock().unwrap();
        let Some(incoming) = st.incoming.as_mut() else {
            return Poll::Ready(None);
        };
        match Pin::new(incoming).poll_frame(cx) {
            Poll::Ready(Some(Ok(frame))) => {
                st.consumed = true;
                if let Some(data) = frame.data_ref() {
                    let room = MAX_BODY_CAPTURE.saturating_sub(st.capture.len());
                    if room > 0 {
                        let take = data.len().min(room);
                        st.capture.extend_from_slice(&data[..take]);
                    }
                }
                Poll::Ready(Some(Ok(frame)))
            }
            other => other,
        }
    }

    fn is_end_stream(&self) -> bool {
        let st = self.state.lock().unwrap();
        match &st.incoming {
            None => true,
            Some(i) => i.is_end_stream(),
        }
    }

    fn size_hint(&self) -> SizeHint {
        let st = self.state.lock().unwrap();
        match &st.incoming {
            None => SizeHint::with_exact(0),
            Some(i) => i.size_hint(),
        }
    }
}

type OnDone = Box<dyn FnOnce(Vec<u8>) + Send + Sync>;

/// Response body passthrough that tees a capped copy for analytics and fires
/// a completion callback when the stream ends (or is dropped/cancelled). The
/// callback receives the captured bytes and emits the ClickHouse row — the
/// row must reflect the full stream, which outlives the handler.
struct TailBody {
    inner: Incoming,
    capture: Vec<u8>,
    on_done: Option<OnDone>,
}

impl TailBody {
    fn finish(&mut self) {
        if let Some(f) = self.on_done.take() {
            f(std::mem::take(&mut self.capture));
        }
    }
}

impl HttpBody for TailBody {
    type Data = Bytes;
    type Error = hyper::Error;

    fn poll_frame(
        mut self: Pin<&mut Self>,
        cx: &mut Context<'_>,
    ) -> Poll<Option<Result<Frame<Self::Data>, Self::Error>>> {
        match Pin::new(&mut self.inner).poll_frame(cx) {
            Poll::Ready(Some(Ok(frame))) => {
                if let Some(data) = frame.data_ref() {
                    let room = MAX_BODY_CAPTURE.saturating_sub(self.capture.len());
                    if room > 0 {
                        let take = data.len().min(room);
                        self.capture.extend_from_slice(&data[..take]);
                    }
                }
                Poll::Ready(Some(Ok(frame)))
            }
            Poll::Ready(None) => {
                self.finish();
                Poll::Ready(None)
            }
            Poll::Ready(Some(Err(e))) => {
                self.finish();
                Poll::Ready(Some(Err(e)))
            }
            Poll::Pending => Poll::Pending,
        }
    }

    fn is_end_stream(&self) -> bool {
        self.inner.is_end_stream()
    }

    fn size_hint(&self) -> SizeHint {
        self.inner.size_hint()
    }
}

impl Drop for TailBody {
    fn drop(&mut self) {
        self.finish();
    }
}

// ---------------------------------------------------------------------------
// Per-request context
// ---------------------------------------------------------------------------

struct Ctx {
    request_id: String,
    host: String,
    location: String,
    start: Instant,
    deadline: Instant,
    tracking: RequestTracking,
    /// Headers merged into the final response (frontline ID, region, timing).
    resp_headers: HeaderMap,
}

impl Ctx {
    fn new(peer: SocketAddr, host: &str, headers: &HeaderMap, timeout: Duration) -> Self {
        let request_id = uid::new(uid::REQUEST_PREFIX);
        let mut resp_headers = HeaderMap::new();
        // Identifies which implementation served the request, so traffic can
        // be attributed during the Go -> Rust cutover.
        resp_headers.insert(
            HeaderName::from_static(HEADER_LANGUAGE),
            HeaderValue::from_static("rust"),
        );
        let location = headers
            .get("x-forwarded-for")
            .and_then(|v| v.to_str().ok())
            .and_then(|xff| {
                xff.split(',')
                    .map(str::trim)
                    .find(|ip| !ip.is_empty())
                    .map(crate::session::strip_port)
            })
            .unwrap_or_else(|| peer.ip().to_string());
        let now = Instant::now();
        Self {
            tracking: RequestTracking {
                request_id: request_id.clone(),
                start_unix_ms: unix_millis(),
                ..Default::default()
            },
            request_id,
            host: host.to_string(),
            location,
            start: now,
            deadline: now + timeout,
            resp_headers,
        }
    }
}

/// Snapshots request metadata into tracking: the ClickHouse row is emitted
/// after the response stream completes, when the request is long gone.
fn snapshot_request(ctx: &mut Ctx, req: &Request<Incoming>) {
    ctx.tracking.method = req.method().as_str().to_string();
    ctx.tracking.path = req.uri().path().to_string();
    ctx.tracking.raw_query = req.uri().query().unwrap_or("").to_string();
    ctx.tracking.request_headers = format_headers(req.headers());
    ctx.tracking.user_agent = req
        .headers()
        .get(http::header::USER_AGENT)
        .and_then(|v| v.to_str().ok())
        .unwrap_or("")
        .to_string();
    ctx.tracking.accept = req
        .headers()
        .get(http::header::ACCEPT)
        .and_then(|v| v.to_str().ok())
        .unwrap_or("")
        .to_string();
}

fn format_timing(name: &str, d: Duration) -> String {
    let ms = d.as_secs_f64() * 1000.0;
    format!("{name}{{scope=frontline}}={ms:.3}ms")
}

/// Maps a hyper client error into a stable URN plus public message, port of
/// categorizeProxyError. Dial-phase failures (is_connect) keep retry-safety.
fn categorize_client_error(
    err: &hyper_util::client::legacy::Error,
    target: &str,
) -> (&'static str, String) {
    // Walk the source chain for the underlying io error.
    let mut source: Option<&(dyn std::error::Error + 'static)> = Some(err);
    while let Some(e) = source {
        if let Some(io_err) = e.downcast_ref::<std::io::Error>() {
            return categorize_io_error(io_err.kind(), target);
        }
        source = e.source();
    }
    if err.is_connect() {
        return (
            urn::PROXY_SERVICE_UNAVAILABLE,
            format!("Failed to connect to the {target}. Please try again or contact support at support@unkey.com."),
        );
    }
    (
        urn::PROXY_BAD_GATEWAY,
        format!("Failed to connect to the {target}. Please try again or contact support at support@unkey.com."),
    )
}

fn categorize_io_error(kind: std::io::ErrorKind, target: &str) -> (&'static str, String) {
    use std::io::ErrorKind;
    match kind {
        ErrorKind::TimedOut | ErrorKind::WouldBlock => (
            urn::PROXY_GATEWAY_TIMEOUT,
            format!("The {target} did not respond in time. Please try again later."),
        ),
        ErrorKind::ConnectionRefused => (
            urn::PROXY_SERVICE_UNAVAILABLE,
            format!("The {target} refused the connection. It may be restarting — please try again in a few seconds."),
        ),
        ErrorKind::ConnectionReset | ErrorKind::ConnectionAborted | ErrorKind::BrokenPipe => (
            urn::PROXY_BAD_GATEWAY,
            format!("The {target} reset the connection unexpectedly. Please try again."),
        ),
        ErrorKind::HostUnreachable | ErrorKind::NetworkUnreachable => (
            urn::PROXY_SERVICE_UNAVAILABLE,
            format!("The {target} is unreachable. Please try again later or contact support at support@unkey.com."),
        ),
        ErrorKind::NotFound => (
            urn::PROXY_SERVICE_UNAVAILABLE,
            format!("DNS resolution failed for the {target}. Please check your configuration or contact support at support@unkey.com."),
        ),
        _ => (
            urn::PROXY_BAD_GATEWAY,
            format!("Failed to connect to the {target}. Please try again or contact support at support@unkey.com."),
        ),
    }
}

/// Hop-by-hop headers (RFC 9110 §7.6.1), as stripped by httputil.ReverseProxy.
const HOP_HEADERS: &[&str] = &[
    "connection",
    "proxy-connection",
    "keep-alive",
    "proxy-authenticate",
    "proxy-authorization",
    "te",
    "trailer",
    "transfer-encoding",
    "upgrade",
];

fn strip_hop_headers(headers: &mut HeaderMap) {
    let named: Vec<String> = headers
        .get_all("connection")
        .iter()
        .filter_map(|v| v.to_str().ok())
        .flat_map(|v| v.split(','))
        .map(|s| s.trim().to_ascii_lowercase())
        .filter(|s| !s.is_empty())
        .collect();
    for name in named {
        headers.remove(&name);
    }
    for h in HOP_HEADERS {
        headers.remove(*h);
    }
}

fn is_upgrade_request(headers: &HeaderMap) -> bool {
    headers
        .get_all("connection")
        .iter()
        .filter_map(|v| v.to_str().ok())
        .flat_map(|v| v.split(','))
        .any(|t| t.trim().eq_ignore_ascii_case("upgrade"))
        && headers.contains_key("upgrade")
}

/// The k8s manifests probe /_unkey/internal/health/ready and /live.
fn is_health_path(path: &str) -> bool {
    path == HEALTH_PATH
        || path
            .strip_prefix(HEALTH_PATH)
            .is_some_and(|rest| rest.starts_with('/'))
}

// ---------------------------------------------------------------------------
// Entry points (one per listener)
// ---------------------------------------------------------------------------

/// HTTPS listener: catchall proxy with the full middleware chain.
pub async fn handle_https(
    gw: Arc<Gateway>,
    mut req: Request<Incoming>,
    peer: SocketAddr,
) -> Response<OutBody> {
    // Reserved-header strip: the single guaranteed sanitization point.
    sanitize_headers(req.headers_mut());

    if req.method() == Method::GET && is_health_path(req.uri().path()) {
        return plain_response(StatusCode::OK, "OK");
    }

    let host = host_of(&req);
    let mut ctx = Ctx::new(peer, &host, req.headers(), gw.cfg.request_timeout);
    snapshot_request(&mut ctx, &req);
    let request_id = ctx.request_id.clone();

    metrics::INFLIGHT_REQUESTS.gauge_with(&[]).inc();
    let result = handle_proxy(&gw, ctx, req).await;
    metrics::INFLIGHT_REQUESTS.gauge_with(&[]).dec();
    let _ = request_id;

    match result {
        Ok(resp) => resp,
        Err((ctx, err)) => render_error(&ctx, &err),
    }
}

/// HTTP listener: ACME challenges + 308 redirect to https.
pub async fn handle_http(
    gw: Arc<Gateway>,
    mut req: Request<Incoming>,
    peer: SocketAddr,
) -> Response<OutBody> {
    sanitize_headers(req.headers_mut());

    if req.method() == Method::GET && is_health_path(req.uri().path()) {
        return plain_response(StatusCode::OK, "OK");
    }

    let host = host_of(&req);

    // ACME's path is more specific, so it dispatches before the redirect.
    if req.uri().path().starts_with(ACME_CHALLENGE_PREFIX) {
        let mut ctx = Ctx::new(peer, &host, req.headers(), gw.cfg.request_timeout);
        snapshot_request(&mut ctx, &req);
        return match handle_acme(&gw, &ctx, &req).await {
            Ok(body) => {
                metrics::REQUESTS_TOTAL.counter_with(&["2xx", "", ""]).inc();
                plain_response(StatusCode::OK, &body)
            }
            Err(e) => render_error(&ctx, &e),
        };
    }

    // Catchall: 308 to https. Deliberately cheap — no router lookups, no
    // observability middleware; volume tracked via the counter.
    let hostname = extract_hostname(&host);
    let target = format!(
        "https://{hostname}{}",
        req.uri()
            .path_and_query()
            .map(|pq| pq.as_str())
            .unwrap_or("/")
    );
    metrics::HTTPS_REDIRECTS_TOTAL.counter_with(&[]).inc();
    let mut resp = Response::builder()
        .status(StatusCode::PERMANENT_REDIRECT)
        .body(full_body(""))
        .unwrap();
    if let Ok(loc) = HeaderValue::from_str(&target) {
        resp.headers_mut().insert(http::header::LOCATION, loc);
    }
    resp
}

fn sanitize_headers(headers: &mut HeaderMap) {
    let reserved: Vec<HeaderName> = headers
        .keys()
        .filter(|k| {
            let s = k.as_str();
            s.len() >= RESERVED_HEADER_PREFIX.len()
                && s[..RESERVED_HEADER_PREFIX.len()].eq_ignore_ascii_case(RESERVED_HEADER_PREFIX)
        })
        .cloned()
        .collect();
    for name in reserved {
        headers.remove(&name);
    }
}

fn host_of(req: &Request<Incoming>) -> String {
    req.headers()
        .get(http::header::HOST)
        .and_then(|v| v.to_str().ok())
        .map(|s| s.to_string())
        .or_else(|| req.uri().authority().map(|a| a.to_string()))
        .unwrap_or_default()
}

fn plain_response(status: StatusCode, body: &str) -> Response<OutBody> {
    let mut resp = Response::builder()
        .status(status)
        .body(full_body(body.to_string()))
        .unwrap();
    resp.headers_mut().insert(
        http::header::CONTENT_TYPE,
        HeaderValue::from_static("text/plain; charset=utf-8"),
    );
    resp
}

// ---------------------------------------------------------------------------
// Catchall proxy handler (routes/proxy/handler.go, engine out of scope)
// ---------------------------------------------------------------------------

async fn handle_proxy(
    gw: &Arc<Gateway>,
    mut ctx: Ctx,
    req: Request<Incoming>,
) -> Result<Response<OutBody>, (Ctx, FrontlineError)> {
    let hostname = extract_hostname(&ctx.host);

    // Route resolution runs sync MySQL + caches: hop to the blocking pool.
    let decision = {
        let router = Arc::clone(&gw.router);
        let hostname = hostname.clone();
        match tokio::task::spawn_blocking(move || router.route(&hostname)).await {
            Ok(Ok(d)) => d,
            Ok(Err(e)) => return Err((ctx, e)),
            Err(join_err) => {
                return Err((
                    ctx,
                    FrontlineError::new(
                        urn::INTERNAL_SERVER_ERROR,
                        format!("routing task failed: {join_err}"),
                        "",
                    ),
                ))
            }
        }
    };

    stamp_response_headers(gw, &mut ctx);

    let (mut parts, body) = req.into_parts();
    let shared_body = ReplayBody::new(body);
    // Take the inbound upgrade handle before parts are reused per attempt.
    let mut inbound_upgrade = parts.extensions.remove::<hyper::upgrade::OnUpgrade>();

    if decision.destination != Destination::LocalInstance {
        return forward_to_region(
            gw,
            ctx,
            &parts,
            shared_body,
            inbound_upgrade,
            &decision.remote_region_address,
        )
        .await;
    }

    // h2c deployments are forwarded over HTTP/1.1 in this port (hyper http1
    // client); the Go TransportRegistry's http1 fallback extended to h2c.
    if decision.upstream_protocol == crate::db::UpstreamProtocol::H2c {
        crate::log_debug!("h2c upstream configured, forwarding over HTTP/1.1",
            "deploymentId" => decision.deployment_id);
    }

    // Populate tracking now that the route resolved.
    ctx.tracking.deployment_id = decision.deployment_id.clone();
    ctx.tracking.workspace_id = decision.workspace_id.clone();
    ctx.tracking.environment_id = decision.environment_id.clone();
    ctx.tracking.project_id = decision.project_id.clone();

    // Try each candidate instance in turn. We only advance on dial-phase
    // failures (hyper: is_connect, body unpolled) — anything else surfaces
    // unchanged, since the upstream may already be processing the request.
    let mut saw_dial_failure = false;
    let mut forward_err: Option<FrontlineError> = None;
    for instance in &decision.local_instances {
        ctx.tracking.instance_id = instance.id.clone();
        ctx.tracking.address = instance.address.clone();

        match forward_to_instance(
            gw,
            &mut ctx,
            &parts,
            shared_body.clone(),
            inbound_upgrade.take(),
            instance,
        )
        .await
        {
            Ok(resp) => {
                if saw_dial_failure {
                    metrics::LOCAL_REQUEST_RETRIES_TOTAL
                        .counter_with(&["recovered"])
                        .inc();
                }
                return Ok(resp);
            }
            Err(RetryableError {
                err,
                dial: true,
                upgrade,
            }) if !shared_body.consumed() => {
                saw_dial_failure = true;
                forward_err = Some(err);
                inbound_upgrade = upgrade;
            }
            Err(RetryableError { err, .. }) => return Err((ctx, err)),
        }
    }
    if saw_dial_failure {
        metrics::LOCAL_REQUEST_RETRIES_TOTAL
            .counter_with(&["exhausted"])
            .inc();
    }

    // Every local instance dial-failed: fall through to the standby peer
    // region when the router provided one.
    if !decision.remote_region_address.is_empty() {
        metrics::REGION_FALLBACKS_TOTAL
            .counter_with(&[&decision.remote_region_address])
            .inc();
        return forward_to_region(
            gw,
            ctx,
            &parts,
            shared_body,
            inbound_upgrade,
            &decision.remote_region_address,
        )
        .await;
    }

    match forward_err {
        Some(e) => Err((ctx, e)),
        // Invariant violation by the router: fail closed with an explicit
        // 503 so the bug is visible.
        None => Err((
            ctx,
            FrontlineError::new(
                urn::ROUTING_NO_RUNNING_INSTANCES,
                "router returned DestinationLocalInstance with empty LocalInstances",
                "Service temporarily unavailable",
            ),
        )),
    }
}

fn stamp_response_headers(gw: &Gateway, ctx: &mut Ctx) {
    let set = |h: &mut HeaderMap, name: &'static str, value: &str| {
        if let Ok(v) = HeaderValue::from_str(value) {
            h.insert(HeaderName::from_static(name), v);
        }
    };
    set(&mut ctx.resp_headers, HEADER_FRONTLINE_ID, &gw.frontline_id);
    set(
        &mut ctx.resp_headers,
        HEADER_REGION,
        &format!("{}::{}", gw.cfg.platform, gw.cfg.region),
    );
    set(&mut ctx.resp_headers, HEADER_REQUEST_ID, &ctx.request_id);
}

struct RetryableError {
    err: FrontlineError,
    dial: bool,
    /// The inbound upgrade handle survives dial failures for the next attempt.
    upgrade: Option<hyper::upgrade::OnUpgrade>,
}

/// Builds the outbound request for one attempt: original method/path,
/// hop-by-hop stripped, forwarding headers added. Port of the directors.
fn build_outbound(
    gw: &Gateway,
    ctx: &Ctx,
    parts: &http::request::Parts,
    body: ReplayBody,
    scheme_authority: &str,
    region_hop: Option<u32>,
) -> Result<Request<ReplayBody>, FrontlineError> {
    let path_and_query = parts
        .uri
        .path_and_query()
        .map(|pq| pq.as_str())
        .unwrap_or("/");
    let uri: Uri = format!("{scheme_authority}{path_and_query}")
        .parse()
        .map_err(|e| {
            FrontlineError::new(
                urn::INTERNAL_SERVER_ERROR,
                format!("failed to parse upstream URL: {e}"),
                "",
            )
        })?;

    let mut headers = parts.headers.clone();
    let upgrade = if is_upgrade_request(&headers) {
        headers.get("upgrade").cloned()
    } else {
        None
    };
    strip_hop_headers(&mut headers);
    if let Some(up) = upgrade {
        headers.insert(
            http::header::CONNECTION,
            HeaderValue::from_static("Upgrade"),
        );
        headers.insert(http::header::UPGRADE, up);
    }

    let set = |h: &mut HeaderMap, name: &'static str, value: &str| {
        if let Ok(v) = HeaderValue::from_str(value) {
            h.insert(HeaderName::from_static(name), v);
        }
    };

    // Preserve original Host so the upstream sees what the client asked for.
    if let Ok(host) = HeaderValue::from_str(&ctx.host) {
        headers.insert(http::header::HOST, host);
    }
    set(&mut headers, HEADER_FRONTLINE_ID, &gw.frontline_id);
    set(
        &mut headers,
        HEADER_REGION,
        &format!("{}::{}", gw.cfg.platform, gw.cfg.region),
    );
    set(&mut headers, HEADER_REQUEST_ID, &ctx.request_id);

    match region_hop {
        None => {
            set(&mut headers, "x-forwarded-for", &ctx.location);
            set(&mut headers, "x-forwarded-host", &ctx.host);
            set(&mut headers, "x-forwarded-proto", "https");
        }
        Some(current_hops) => {
            set(&mut headers, HEADER_PARENT_FRONTLINE_ID, &gw.frontline_id);
            set(&mut headers, HEADER_PARENT_REQUEST_ID, &ctx.request_id);
            let next = current_hops + 1;
            set(&mut headers, HEADER_FRONTLINE_HOPS, &next.to_string());
            if next >= gw.cfg.max_hops.saturating_sub(1) {
                crate::log_warn!("approaching max hops limit",
                    "currentHops" => next, "maxHops" => gw.cfg.max_hops, "hostname" => ctx.host);
            }
        }
    }

    let mut req = Request::builder()
        .method(parts.method.clone())
        .uri(uri)
        .body(body)
        .expect("request build");
    *req.headers_mut() = headers;
    Ok(req)
}

async fn forward_to_instance(
    gw: &Arc<Gateway>,
    ctx: &mut Ctx,
    parts: &http::request::Parts,
    body: ReplayBody,
    inbound_upgrade: Option<hyper::upgrade::OnUpgrade>,
    instance: &InstanceRow,
) -> Result<Response<OutBody>, RetryableError> {
    let outbound = build_outbound(
        gw,
        ctx,
        parts,
        body.clone(),
        &format!("http://{}", instance.address),
        None,
    )
    .map_err(|err| RetryableError {
        err,
        dial: false,
        upgrade: None,
    })?;

    ctx.tracking.instance_start_ms = unix_millis();
    let proxy_start = Instant::now();

    let remaining = ctx.deadline.saturating_duration_since(Instant::now());
    let result = tokio::time::timeout(remaining, gw.instance_client.request(outbound)).await;

    let resp = match result {
        Err(_elapsed) => {
            metrics::UPSTREAM_SECONDS
                .histogram_with(&[DESTINATION_INSTANCE])
                .observe(proxy_start.elapsed().as_secs_f64());
            return Err(RetryableError {
                err: FrontlineError::new(
                    urn::PROXY_GATEWAY_TIMEOUT,
                    format!("proxy timeout forwarding to {}", instance.address),
                    "The deployment instance did not respond in time. Please try again later.",
                ),
                dial: false,
                upgrade: None,
            });
        }
        Ok(Err(e)) => {
            let dial = e.is_connect() && !body.consumed();
            metrics::UPSTREAM_DIALS_TOTAL
                .counter_with(&[DESTINATION_INSTANCE, if dial { "error" } else { "success" }])
                .inc();
            let (code, message) = categorize_client_error(&e, "deployment instance");
            let mut err = FrontlineError::new(
                code,
                format!("proxy error forwarding to {}: {e}", instance.address),
                message,
            );
            if dial {
                err = err.dial();
            }
            return Err(RetryableError {
                err,
                dial,
                upgrade: inbound_upgrade,
            });
        }
        Ok(Ok(resp)) => {
            metrics::UPSTREAM_DIALS_TOTAL
                .counter_with(&[DESTINATION_INSTANCE, "success"])
                .inc();
            resp
        }
    };

    Ok(relay_response(
        gw,
        ctx,
        body,
        resp,
        inbound_upgrade,
        DESTINATION_INSTANCE,
        proxy_start,
    ))
}

async fn forward_to_region(
    gw: &Arc<Gateway>,
    mut ctx: Ctx,
    parts: &http::request::Parts,
    body: ReplayBody,
    inbound_upgrade: Option<hyper::upgrade::OnUpgrade>,
    target_region_platform: &str,
) -> Result<Response<OutBody>, (Ctx, FrontlineError)> {
    stamp_response_headers(gw, &mut ctx);

    // Hop accounting and loop prevention.
    let mut current_hops: u32 = 0;
    if let Some(hops) = parts
        .headers
        .get(HEADER_FRONTLINE_HOPS)
        .and_then(|v| v.to_str().ok())
        .and_then(|v| v.trim().parse::<u32>().ok())
    {
        current_hops = hops;
        let src_region = parts
            .headers
            .get(HEADER_REGION)
            .and_then(|v| v.to_str().ok())
            .map(|s| s.to_string())
            .unwrap_or_else(|| format!("{}::{}", gw.cfg.platform, gw.cfg.region));
        metrics::HOPS_HISTOGRAM
            .histogram_with(&[&src_region, target_region_platform])
            .observe(hops as f64);

        if hops >= gw.cfg.max_hops {
            crate::log_error!("too many frontline hops - rejecting request",
                "hops" => hops, "maxHops" => gw.cfg.max_hops,
                "hostname" => ctx.host, "requestID" => ctx.request_id);
            return Err((
                ctx,
                FrontlineError::new(
                    urn::INTERNAL_SERVER_ERROR,
                    format!("request exceeded maximum hop count: {hops}"),
                    "Request routing limit exceeded",
                ),
            ));
        }
    }

    let target_host = format!("frontline.{target_region_platform}.{}", gw.cfg.apex_domain);
    let outbound = match build_outbound(
        gw,
        &ctx,
        parts,
        body.clone(),
        &format!("https://{target_host}"),
        Some(current_hops),
    ) {
        Ok(o) => o,
        Err(e) => return Err((ctx, e)),
    };

    let proxy_start = Instant::now();
    let remaining = ctx.deadline.saturating_duration_since(Instant::now());
    let result = tokio::time::timeout(remaining, gw.region_client.request(outbound)).await;

    let resp = match result {
        Err(_elapsed) => {
            return Err((
                ctx,
                FrontlineError::new(
                    urn::PROXY_GATEWAY_TIMEOUT,
                    format!("proxy timeout forwarding to {target_host}"),
                    "The peer frontline did not respond in time. Please try again later.",
                ),
            ));
        }
        Ok(Err(e)) => {
            metrics::UPSTREAM_DIALS_TOTAL
                .counter_with(&[
                    DESTINATION_FRONTLINE,
                    if e.is_connect() { "error" } else { "success" },
                ])
                .inc();
            let (code, message) = categorize_client_error(&e, "peer frontline");
            return Err((
                ctx,
                FrontlineError::new(
                    code,
                    format!("proxy error forwarding to {target_host}: {e}"),
                    message,
                ),
            ));
        }
        Ok(Ok(resp)) => {
            metrics::UPSTREAM_DIALS_TOTAL
                .counter_with(&[DESTINATION_FRONTLINE, "success"])
                .inc();
            resp
        }
    };

    Ok(relay_response(
        gw,
        &mut ctx,
        body,
        resp,
        inbound_upgrade,
        DESTINATION_FRONTLINE,
        proxy_start,
    ))
}

/// Relays the upstream response to the client: hop headers stripped,
/// frontline + timing headers merged, body tee'd for analytics, completion
/// callback emitting the ClickHouse row, 101 upgrades spliced.
fn relay_response(
    gw: &Arc<Gateway>,
    ctx: &mut Ctx,
    req_body: ReplayBody,
    mut resp: Response<Incoming>,
    inbound_upgrade: Option<hyper::upgrade::OnUpgrade>,
    destination: &'static str,
    proxy_start: Instant,
) -> Response<OutBody> {
    let status = resp.status();
    let upgrading = status == StatusCode::SWITCHING_PROTOCOLS;

    // Capture what analytics needs before the response is consumed.
    ctx.tracking.request_body = req_body.take_capture();
    let upgrade_value = resp.headers().get(http::header::UPGRADE).cloned();

    let mut headers = resp.headers().clone();
    strip_hop_headers(&mut headers);
    for (k, v) in ctx.resp_headers.iter() {
        headers.insert(k.clone(), v.clone());
    }
    let elapsed = ctx.start.elapsed();
    for (name, d) in [("frontline", elapsed), ("total", elapsed)] {
        if let Ok(v) = HeaderValue::from_str(&format_timing(name, d)) {
            headers.append(HeaderName::from_static(HEADER_TIMING), v);
        }
    }

    if upgrading {
        // Splice the two upgraded streams once both sides complete.
        if let (Some(client_up), upstream_resp) = (inbound_upgrade, &mut resp) {
            let upstream_up = hyper::upgrade::on(upstream_resp);
            tokio::spawn(async move {
                match (client_up.await, upstream_up.await) {
                    (Ok(client_io), Ok(upstream_io)) => {
                        let mut a = hyper_util::rt::TokioIo::new(client_io);
                        let mut b = hyper_util::rt::TokioIo::new(upstream_io);
                        let _ = tokio::io::copy_bidirectional(&mut a, &mut b).await;
                    }
                    (c, u) => {
                        crate::log_debug!("upgrade splice failed",
                            "client" => c.is_err(), "upstream" => u.is_err());
                    }
                }
            });
        }
        let mut out = Response::builder()
            .status(status)
            .body(full_body(""))
            .unwrap();
        *out.headers_mut() = headers;
        if let Some(up) = upgrade_value {
            out.headers_mut().insert(
                http::header::CONNECTION,
                HeaderValue::from_static("Upgrade"),
            );
            out.headers_mut().insert(http::header::UPGRADE, up);
        }
        metrics::UPSTREAM_SECONDS
            .histogram_with(&[destination])
            .observe(proxy_start.elapsed().as_secs_f64());
        emit_request_log_and_metrics(ctx, status.as_u16(), "", "");
        return out;
    }

    // Completion callback: fires when the response stream finishes (or the
    // client goes away), emitting upstream timing + the ClickHouse row.
    let on_done = make_completion(gw, ctx, status.as_u16(), &headers, destination, proxy_start);

    let (parts_out, body) = resp.into_parts();
    let _ = parts_out;
    let tail = TailBody {
        inner: body,
        capture: Vec::new(),
        on_done: Some(on_done),
    };

    let mut out = Response::builder()
        .status(status)
        .body(tail.boxed())
        .unwrap();
    *out.headers_mut() = headers;

    emit_request_log_and_metrics(ctx, status.as_u16(), "", "");
    out
}

/// Builds the deferred completion closure carrying everything the ClickHouse
/// row needs; the response body outlives the handler.
fn make_completion(
    gw: &Arc<Gateway>,
    ctx: &mut Ctx,
    status: u16,
    resp_headers: &HeaderMap,
    destination: &'static str,
    proxy_start: Instant,
) -> OnDone {
    let gw = Arc::clone(gw);
    let mut tracking = std::mem::take(&mut ctx.tracking);
    let resp_headers_fmt = format_headers(resp_headers);
    let location = ctx.location.clone();
    let host = ctx.host.clone();

    Box::new(move |response_body: Vec<u8>| {
        metrics::UPSTREAM_SECONDS
            .histogram_with(&[destination])
            .observe(proxy_start.elapsed().as_secs_f64());
        tracking.instance_end_ms = unix_millis();
        tracking.response_body = response_body;

        // Local-instance path only; cross-region requests are logged by the
        // peer frontline.
        if tracking.deployment_id.is_empty() || tracking.instance_id.is_empty() {
            return;
        }
        let row = crate::clickhouse::build_sentinel_request(
            &tracking,
            &gw.frontline_id,
            &gw.cfg.region,
            &gw.cfg.platform,
            &host,
            status,
            resp_headers_fmt,
            &location,
            unix_millis(),
        );
        gw.frontline_requests.buffer(row);
    })
}

/// "Name: value" pairs with Authorization redacted (analytics contract).
fn format_headers(headers: &HeaderMap) -> Vec<String> {
    let mut out = Vec::with_capacity(headers.len());
    let mut redacted_auth = false;
    for (k, v) in headers.iter() {
        if k == http::header::AUTHORIZATION {
            if !redacted_auth {
                out.push(format!("{}: [REDACTED]", k.as_str()));
                redacted_auth = true;
            }
        } else {
            out.push(format!("{}: {}", k.as_str(), v.to_str().unwrap_or("?")));
        }
    }
    out
}

// ---------------------------------------------------------------------------
// ACME (routes/acme/handler.go)
// ---------------------------------------------------------------------------

async fn handle_acme(
    gw: &Arc<Gateway>,
    ctx: &Ctx,
    req: &Request<Incoming>,
) -> Result<String, FrontlineError> {
    let hostname = extract_hostname(&ctx.host);

    let domain = {
        let db = gw.db.clone();
        let h = hostname.clone();
        tokio::task::spawn_blocking(move || db.find_custom_domain_id_by_domain(&h))
            .await
            .map_err(|e| {
                FrontlineError::new(urn::INTERNAL_SERVER_ERROR, format!("join: {e}"), "")
            })??
    };
    if domain.is_none() {
        return Err(FrontlineError::new(
            urn::ROUTING_CONFIG_NOT_FOUND,
            format!("no custom domain registered for hostname: {hostname}"),
            "Domain not configured",
        ));
    }

    let Some(acme) = gw.acme.clone() else {
        return Err(FrontlineError::new(
            urn::INTERNAL_SERVER_ERROR,
            "acme client not configured",
            "Failed to handle ACME challenge",
        ));
    };

    let token = req
        .uri()
        .path()
        .rsplit('/')
        .next()
        .unwrap_or("")
        .to_string();
    crate::log_info!("Handling ACME challenge", "hostname" => hostname, "token" => token);

    let domain_for_call = hostname.clone();
    let authorization =
        tokio::task::spawn_blocking(move || acme.verify_certificate(&domain_for_call, &token))
            .await
            .map_err(|e| FrontlineError::new(urn::INTERNAL_SERVER_ERROR, format!("join: {e}"), ""))?
            .map_err(|e| {
                crate::log_error!("Failed to handle certificate verification", "error" => e);
                FrontlineError::new(
                    urn::APP_UNEXPECTED_ERROR,
                    format!("failed to handle ACME challenge: {e}"),
                    "Failed to handle ACME challenge",
                )
            })?;

    crate::log_info!("Certificate verification handled", "response" => authorization);
    Ok(authorization)
}

// ---------------------------------------------------------------------------
// Observability tail: error rendering, request log, requests_total
// ---------------------------------------------------------------------------

fn emit_request_log_and_metrics(ctx: &Ctx, status: u16, fault_domain: &str, code: &str) {
    metrics::REQUESTS_TOTAL
        .counter_with(&[metrics::status_class(status), fault_domain, code])
        .inc();
    if !ctx.tracking.path.starts_with(INTERNAL_PREFIX) {
        crate::log_info!("request",
            "method" => ctx.tracking.method,
            "path" => ctx.tracking.path,
            "host" => ctx.host,
            "status" => status,
            "latencyMs" => ctx.start.elapsed().as_millis(),
            "requestId" => ctx.request_id);
    }
}

/// Renders an error as JSON or the HTML error page based on Accept. Port of
/// the rendering half of middleware/observability.go.
fn render_error(ctx: &Ctx, err: &FrontlineError) -> Response<OutBody> {
    let page = error_page_info(err.urn);
    let status = page.status;

    let user_message = if page.message.is_empty() {
        err.public.as_str()
    } else {
        page.message
    };

    if status >= 500 {
        crate::log_error!("frontline error",
            "error" => err.internal, "requestId" => ctx.request_id,
            "code" => err.urn, "faultDomain" => err.fault_domain(),
            "status" => status, "host" => ctx.host);
    } else {
        crate::log_info!("frontline request error",
            "error" => err.internal, "requestId" => ctx.request_id,
            "code" => err.urn, "faultDomain" => err.fault_domain(),
            "status" => status, "host" => ctx.host);
    }

    metrics::REQUESTS_TOTAL
        .counter_with(&[metrics::status_class(status), err.fault_domain(), err.urn])
        .inc();

    let accept = ctx.tracking.accept.as_str();
    let prefer_json = accept.contains("application/json")
        || accept.contains("application/*")
        || (accept.contains("*/*") && !accept.contains("text/html"));

    let (content_type, body) = if prefer_json {
        let body = serde_json::json!({
            "meta": { "requestId": ctx.request_id },
            "error": { "code": err.urn, "message": user_message },
        })
        .to_string();
        ("application/json", body)
    } else {
        let html = errorpage::render(&errorpage::Data {
            status_code: status,
            title: page.title,
            message: user_message,
            error_code: err.urn,
            docs_url: &err.docs_url(),
            request_id: &ctx.request_id,
        });
        ("text/html; charset=utf-8", html)
    };

    let mut resp = Response::builder()
        .status(StatusCode::from_u16(status).unwrap_or(StatusCode::INTERNAL_SERVER_ERROR))
        .body(full_body(body))
        .unwrap();
    for (k, v) in ctx.resp_headers.iter() {
        resp.headers_mut().insert(k.clone(), v.clone());
    }
    resp.headers_mut().insert(
        http::header::CONTENT_TYPE,
        HeaderValue::from_static(match content_type {
            "application/json" => "application/json",
            _ => "text/html; charset=utf-8",
        }),
    );
    resp
}
