//! Port of run.go: wires config, database, vault, certificate manager,
//! router, ClickHouse buffers, the hyper clients, and the three listeners
//! (HTTPS, HTTP, Prometheus) on a tokio runtime.

use std::sync::Arc;

use hyper::service::service_fn;
use hyper_util::rt::TokioIo;
use tokio::net::TcpListener;
use tokio_rustls::TlsAcceptor;

use crate::certmanager::{build_tls_config, CertManager};
use crate::clickhouse::{Buffer, ClickHouseClient, SentinelRequest};
use crate::config::Config;
use crate::connectrpc::{AcmeClient, VaultClient};
use crate::db::Database;
use crate::error::FrontlineError;
use crate::gateway::{build_clients, handle_http, handle_https, Gateway};
use crate::httpx::{write_response_head, Conn, Headers};
use crate::metrics;
use crate::router::Router;
use crate::uid;

/// Starts the frontline server and blocks until the listeners stop.
///
/// Frontline is the multi-tenant ingress that:
///   - terminates TLS for customer domains
///   - resolves the hostname to a deployment
///   - forwards directly to a running deployment instance in this region,
///     or hops to a peer frontline in another region when no local instance
///     exists.
pub fn run(mut cfg: Config) -> Result<(), FrontlineError> {
    cfg.validate().map_err(|e| {
        FrontlineError::new(
            crate::error::urn::CONFIG_LOAD_FAILED,
            format!("bad config: {e}"),
            "",
        )
    })?;

    if cfg.instance_id.is_empty() {
        cfg.instance_id = uid::new(uid::INSTANCE_PREFIX);
    }
    crate::logger::add_base_attr("instanceID", &cfg.instance_id);
    crate::logger::add_base_attr("region", &cfg.region);

    // internal/db drives the routing/cert lookups against a read-only
    // replica. When the operator omits a dedicated replica DSN we fall back
    // to the primary.
    let read_dsn = if cfg.database.readonly_replica.is_empty() {
        cfg.database.primary.clone()
    } else {
        cfg.database.readonly_replica.clone()
    };
    let database = Database::connect(&read_dsn)?;

    // Vault is optional: without it, dynamic TLS certificate decryption is
    // unavailable (static cert files still work).
    let vault = if !cfg.vault.url.is_empty() {
        let client = VaultClient::new(&cfg.vault.url, &cfg.vault.token)?;
        crate::log_info!("Vault client initialized", "url" => cfg.vault.url);
        Some(Arc::new(client))
    } else {
        crate::log_warn!(
            "Vault not configured, dynamic TLS certificate decryption will be unavailable"
        );
        None
    };

    let cert_manager = match &vault {
        Some(vault) => {
            crate::log_info!("Certificate manager initialized with vault-backed decryption");
            Some(Arc::new(CertManager::new(
                database.clone(),
                Arc::clone(vault),
            )))
        }
        None => {
            crate::log_warn!("Certificate manager not initialized, vault client is nil");
            None
        }
    };

    if !cfg.redis.url.is_empty() {
        crate::log_warn!(
            "Redis is configured but unused: the policy engine is out of scope in frontline-rs"
        );
    }
    if cfg.pprof.is_some() {
        crate::log_warn!("pprof is configured but unused: not supported by frontline-rs");
    }

    // ClickHouse analytics; no-op buffer when not configured. This port
    // speaks the HTTP interface only — a native-protocol URL cannot work,
    // so fall back to dropping analytics instead of logging errors forever.
    let frontline_requests: Buffer<SentinelRequest> = if cfg.clickhouse.url.is_empty() {
        Buffer::noop()
    } else if !cfg.clickhouse.url.starts_with("http://")
        && !cfg.clickhouse.url.starts_with("https://")
    {
        crate::log_warn!(
            "ClickHouse URL is not an HTTP endpoint; analytics disabled. frontline-rs uses the ClickHouse HTTP interface (e.g. http://user:pass@host:8123), not the native protocol",
            "url" => cfg.clickhouse.url);
        Buffer::noop()
    } else {
        let client = ClickHouseClient::new(&cfg.clickhouse.url).map_err(|e| {
            FrontlineError::new(
                crate::error::urn::CONFIG_LOAD_FAILED,
                format!("unable to create clickhouse: {e}"),
                "",
            )
        })?;
        Buffer::new(
            client,
            "default.sentinel_requests_raw_v1",
            cfg.clickhouse.batch_size,
            cfg.clickhouse.buffer_size,
            cfg.clickhouse.consumers,
        )
    };

    let router = Arc::new(Router::new(&cfg.platform, &cfg.region, database.clone()));

    let tls_config = build_tls_config(&cfg, cert_manager)?;

    let acme = match AcmeClient::new(&cfg.ctrl_addr) {
        Ok(c) => Some(Arc::new(c)),
        Err(e) => {
            crate::log_warn!("ACME client not configured", "error" => e);
            None
        }
    };

    let (instance_client, region_client) = build_clients();

    let gateway = Arc::new(Gateway {
        frontline_id: cfg.instance_id.clone(),
        router,
        db: database.clone(),
        acme,
        frontline_requests,
        instance_client,
        region_client,
        cfg: cfg.clone(),
    });

    // Readiness: confirm the database answers before accepting traffic.
    database.ping()?;

    // Prometheus stays on a plain thread: one tiny endpoint, no need to
    // involve the async runtime.
    if cfg.prometheus_port > 0 {
        let listener = std::net::TcpListener::bind(("0.0.0.0", cfg.prometheus_port))
            .map_err(|e| bind_err(cfg.prometheus_port, e))?;
        crate::log_info!("Prometheus server started", "addr" => format!(":{}", cfg.prometheus_port));
        std::thread::spawn(move || serve_metrics(listener));
    } else {
        crate::log_warn!("Prometheus not configured, skipping metrics server");
    }

    let runtime = tokio::runtime::Builder::new_multi_thread()
        .enable_all()
        .build()
        .map_err(|e| {
            FrontlineError::new(
                crate::error::urn::INTERNAL_SERVER_ERROR,
                format!("unable to start tokio runtime: {e}"),
                "",
            )
        })?;

    runtime.block_on(async move {
        let mut handles = Vec::new();

        if cfg.http_port > 0 {
            let listener = bind(cfg.http_port).await?;
            crate::log_info!("HTTP server started", "addr" => format!(":{}", cfg.http_port));
            let gw = Arc::clone(&gateway);
            handles.push(tokio::spawn(serve(gw, listener, None, false)));
        } else {
            crate::log_warn!("HTTP server not configured, ACME HTTP-01 challenges and HTTP→HTTPS redirects will not work");
        }

        if cfg.https_port > 0 {
            let listener = bind(cfg.https_port).await?;
            let acceptor = tls_config.clone().map(TlsAcceptor::from);
            crate::log_info!("HTTPS frontline server started",
                "addr" => format!(":{}", cfg.https_port),
                "tlsEnabled" => acceptor.is_some());
            let gw = Arc::clone(&gateway);
            handles.push(tokio::spawn(serve(gw, listener, acceptor, true)));
        } else {
            crate::log_warn!("HTTPS server not configured, skipping");
        }

        crate::log_info!("Frontline server initialized",
            "region" => cfg.region, "apexDomain" => cfg.apex_domain);

        for h in handles {
            let _ = h.await;
        }
        Ok::<(), FrontlineError>(())
    })?;

    crate::log_info!("Frontline server shut down");
    Ok(())
}

async fn bind(port: u16) -> Result<TcpListener, FrontlineError> {
    // Dual-stack like Go's ":port" — bind IPv6-any (accepts v4-mapped on
    // platforms with v6only off), falling back to IPv4-any.
    match TcpListener::bind(("::", port)).await {
        Ok(l) => Ok(l),
        Err(_) => TcpListener::bind(("0.0.0.0", port))
            .await
            .map_err(|e| bind_err(port, e)),
    }
}

fn bind_err(port: u16, e: std::io::Error) -> FrontlineError {
    FrontlineError::new(
        crate::error::urn::CONFIG_LOAD_FAILED,
        format!("unable to listen on port {port}: {e}"),
        "",
    )
}

/// Accept loop: one tokio task per connection, hyper http1 with upgrades.
async fn serve(
    gw: Arc<Gateway>,
    listener: TcpListener,
    tls: Option<TlsAcceptor>,
    https_mode: bool,
) {
    loop {
        let (stream, peer) = match listener.accept().await {
            Ok(x) => x,
            Err(e) => {
                crate::log_warn!("accept failed", "error" => e);
                continue;
            }
        };
        let _ = stream.set_nodelay(true);
        let gw = Arc::clone(&gw);
        let tls = tls.clone();

        tokio::spawn(async move {
            let service = service_fn(move |req| {
                let gw = Arc::clone(&gw);
                async move {
                    let resp = if https_mode {
                        handle_https(gw, req, peer).await
                    } else {
                        handle_http(gw, req, peer).await
                    };
                    Ok::<_, std::convert::Infallible>(resp)
                }
            });

            let result = match tls {
                Some(acceptor) => match acceptor.accept(stream).await {
                    Ok(tls_stream) => {
                        hyper::server::conn::http1::Builder::new()
                            .serve_connection(TokioIo::new(tls_stream), service)
                            .with_upgrades()
                            .await
                    }
                    Err(e) => {
                        crate::log_debug!("tls handshake failed", "error" => e, "peer" => peer);
                        return;
                    }
                },
                None => {
                    hyper::server::conn::http1::Builder::new()
                        .serve_connection(TokioIo::new(stream), service)
                        .with_upgrades()
                        .await
                }
            };
            if let Err(e) = result {
                crate::log_debug!("connection closed", "error" => e, "peer" => peer);
            }
        });
    }
}

/// Tiny /metrics + health server (pkg/prometheus equivalent), synchronous.
fn serve_metrics(listener: std::net::TcpListener) {
    for stream in listener.incoming().flatten() {
        std::thread::spawn(move || {
            let _ = stream.set_read_timeout(Some(std::time::Duration::from_secs(10)));
            let mut conn = Conn::new(Box::new(stream));
            while let Ok(Some(req)) = conn.read_request_head() {
                let (status, body) = if req.path == "/metrics" {
                    (200, metrics::REGISTRY.gather())
                } else if req.path.starts_with("/_unkey/internal/health") {
                    (200, "OK".to_string())
                } else {
                    (404, "not found".to_string())
                };
                let mut headers = Headers::new();
                headers.set("Content-Type", "text/plain; version=0.0.4");
                headers.set("Content-Length", &body.len().to_string());
                if write_response_head(conn.stream_mut(), status, "", &headers)
                    .and_then(|_| conn.write_all(body.as_bytes()))
                    .and_then(|_| conn.flush())
                    .is_err()
                {
                    break;
                }
                if !req.keep_alive {
                    break;
                }
            }
        });
    }
}
