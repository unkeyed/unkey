//! Port of internal/certmanager + tls.go: serves TLS certificates for
//! customer domains, fetched from the database and decrypted via vault,
//! cached with SWR semantics (Fresh 1h / Stale 12h / 10k entries).

use std::fmt::Debug;
use std::sync::Arc;
use std::time::Duration;

use rustls::crypto::ring::sign::any_supported_type;
use rustls::pki_types::{CertificateDer, PrivateKeyDer};
use rustls::server::{ClientHello, ResolvesServerCert};
use rustls::sign::CertifiedKey;
use rustls::ServerConfig;

use crate::cache::SwrCache;
use crate::config::Config;
use crate::connectrpc::VaultClient;
use crate::db::Database;
use crate::error::{urn, FrontlineError};

pub struct CertManager {
    db: Database,
    vault: Arc<VaultClient>,
    cache: Arc<SwrCache<Arc<CertifiedKey>>>,
}

impl CertManager {
    pub fn new(db: Database, vault: Arc<VaultClient>) -> Self {
        Self {
            db,
            vault,
            cache: SwrCache::new(
                "tls_certificate",
                Duration::from_secs(3600),
                Duration::from_secs(12 * 3600),
                10_000,
            ),
        }
    }

    /// Resolves the certificate for a domain. Lookup candidates are the
    /// exact domain plus its immediate wildcard (api.example.com ->
    /// [api.example.com, *.example.com]); the cache stores the cert under
    /// its canonical hostname so sibling subdomains share a wildcard entry.
    pub fn get_certificate(
        &self,
        domain: &str,
    ) -> Result<Option<Arc<CertifiedKey>>, FrontlineError> {
        let mut candidates = vec![domain.to_string()];
        if let Some((_, parent)) = domain.split_once('.') {
            candidates.push(format!("*.{parent}"));
        }

        let db = self.db.clone();
        let vault = Arc::clone(&self.vault);
        let exact = domain.to_string();
        let lookup = candidates.clone();

        self.cache.swr_with_fallback(&candidates, move || {
            let Some(material) = db.find_best_certificate_by_candidates(&lookup, &exact)? else {
                return Ok(None);
            };

            let key_pem = vault.decrypt(&material.workspace_id, &material.encrypted_private_key)?;
            let certified = build_certified_key(&material.certificate, &key_pem)?;

            // Cache under the cert's actual hostname for proper sharing.
            Ok(Some((Arc::new(certified), material.hostname)))
        })
    }
}

/// Builds a rustls CertifiedKey from PEM cert chain + PEM private key.
/// Port of tls.X509KeyPair.
pub fn build_certified_key(cert_pem: &str, key_pem: &str) -> Result<CertifiedKey, FrontlineError> {
    let fail = |msg: String| FrontlineError::new(urn::INTERNAL_SERVER_ERROR, msg, "");

    let certs: Vec<CertificateDer<'static>> = rustls_pemfile::certs(&mut cert_pem.as_bytes())
        .collect::<Result<_, _>>()
        .map_err(|e| fail(format!("parse certificate PEM: {e}")))?;
    if certs.is_empty() {
        return Err(fail("certificate PEM contains no certificates".into()));
    }

    let key: PrivateKeyDer<'static> = rustls_pemfile::private_key(&mut key_pem.as_bytes())
        .map_err(|e| fail(format!("parse private key PEM: {e}")))?
        .ok_or_else(|| fail("private key PEM contains no key".into()))?;

    let signing_key =
        any_supported_type(&key).map_err(|e| fail(format!("unsupported private key: {e}")))?;

    Ok(CertifiedKey::new(certs, signing_key))
}

/// SNI-driven resolver backed by the cert manager (production mode).
struct DynamicResolver {
    manager: Arc<CertManager>,
}

impl Debug for DynamicResolver {
    fn fmt(&self, f: &mut std::fmt::Formatter<'_>) -> std::fmt::Result {
        f.write_str("DynamicResolver")
    }
}

impl ResolvesServerCert for DynamicResolver {
    fn resolve(&self, client_hello: ClientHello<'_>) -> Option<Arc<CertifiedKey>> {
        let name = client_hello.server_name()?;
        match self.manager.get_certificate(name) {
            Ok(Some(cert)) => Some(cert),
            Ok(None) => {
                crate::log_warn!("no certificate found for SNI", "serverName" => name);
                None
            }
            Err(err) => {
                crate::log_error!("Failed to get certificate", "serverName" => name, "error" => err);
                None
            }
        }
    }
}

/// Static single-certificate resolver (development mode).
#[derive(Debug)]
struct StaticResolver {
    cert: Arc<CertifiedKey>,
}

impl ResolvesServerCert for StaticResolver {
    fn resolve(&self, _client_hello: ClientHello<'_>) -> Option<Arc<CertifiedKey>> {
        Some(Arc::clone(&self.cert))
    }
}

/// Builds the server TLS config. Port of buildTlsConfig in tls.go:
///   - Disabled: TLS explicitly disabled via config -> None
///   - Static:   cert/key files from disk (development)
///   - Dynamic:  certificates from database + vault (production)
pub fn build_tls_config(
    cfg: &Config,
    cert_manager: Option<Arc<CertManager>>,
) -> Result<Option<Arc<ServerConfig>>, FrontlineError> {
    let tls_disabled = cfg.tls.as_ref().is_some_and(|t| t.disabled);
    if tls_disabled {
        crate::log_warn!("TLS explicitly disabled via config");
        return Ok(None);
    }

    if let Some(tls) = &cfg.tls {
        if !tls.cert_file.is_empty() && !tls.key_file.is_empty() {
            crate::log_info!("TLS configured with static certificate files",
                "certFile" => tls.cert_file, "keyFile" => tls.key_file);
            let cert_pem = std::fs::read_to_string(&tls.cert_file).map_err(|e| {
                FrontlineError::new(
                    urn::CONFIG_LOAD_FAILED,
                    format!("read cert file {}: {e}", tls.cert_file),
                    "",
                )
            })?;
            let key_pem = std::fs::read_to_string(&tls.key_file).map_err(|e| {
                FrontlineError::new(
                    urn::CONFIG_LOAD_FAILED,
                    format!("read key file {}: {e}", tls.key_file),
                    "",
                )
            })?;
            let certified = build_certified_key(&cert_pem, &key_pem)?;
            let config = ServerConfig::builder()
                .with_no_client_auth()
                .with_cert_resolver(Arc::new(StaticResolver {
                    cert: Arc::new(certified),
                }));
            return Ok(Some(Arc::new(config)));
        }
    }

    if let Some(manager) = cert_manager {
        crate::log_info!("TLS configured with dynamic certificate manager");
        let config = ServerConfig::builder()
            .with_no_client_auth()
            .with_cert_resolver(Arc::new(DynamicResolver { manager }));
        return Ok(Some(Arc::new(config)));
    }

    Err(FrontlineError::new(
        urn::CONFIG_LOAD_FAILED,
        "TLS is required but no certificate source configured: either enable Vault for dynamic \
         certificates, provide [tls] cert_file and key_file, or explicitly disable TLS with \
         [tls] disabled = true",
        "",
    ))
}
