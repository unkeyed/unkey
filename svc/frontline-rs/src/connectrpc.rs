//! Minimal Connect RPC client (JSON codec, unary calls) over the hand-rolled
//! HTTP client. Replaces the generated connectrpc.com clients for the two
//! procedures frontline calls: vault.v1.VaultService/Decrypt and
//! ctrl.v1.AcmeService/VerifyCertificate.

use std::time::Duration;

use serde::de::DeserializeOwned;
use serde::{Deserialize, Serialize};

use crate::error::{urn, FrontlineError};
use crate::httpx::{copy_body, Conn, Headers};
use crate::tlsx::SimpleUrl;

/// Performs a unary Connect call: POST {base}{procedure} with a JSON body.
/// Non-200 responses carry a Connect error JSON ({"code", "message"}).
pub fn call<Req: Serialize, Resp: DeserializeOwned>(
    base: &SimpleUrl,
    procedure: &str,
    bearer_token: Option<&str>,
    request: &Req,
    timeout: Duration,
) -> Result<Resp, FrontlineError> {
    let fail = |msg: String| {
        FrontlineError::new(
            urn::INTERNAL_SERVER_ERROR,
            format!("connect call {procedure}: {msg}"),
            "",
        )
    };

    let body = serde_json::to_vec(request).map_err(|e| fail(format!("encode: {e}")))?;
    let base_path = base.path.trim_end_matches('/');
    let target = format!("{base_path}{procedure}");

    let stream = base
        .connect(timeout)
        .map_err(|e| fail(format!("dial: {e}")))?;
    let mut conn = Conn::new(stream);

    let mut headers = Headers::new();
    headers.set("Host", &base.host_header());
    headers.set("Content-Type", "application/json");
    headers.set("Content-Length", &body.len().to_string());
    headers.set("Connection", "close");
    if let Some(token) = bearer_token {
        headers.set("Authorization", &format!("Bearer {token}"));
    }

    {
        let out = conn.stream_mut();
        crate::httpx::write_request_head(out, "POST", &target, &headers)
            .map_err(|e| fail(format!("write: {e}")))?;
        out.write_all(&body)
            .map_err(|e| fail(format!("write: {e}")))?;
        out.flush().map_err(|e| fail(format!("write: {e}")))?;
    }

    let resp = conn
        .read_response_head("POST")
        .map_err(|e| fail(format!("read response: {e}")))?;
    let mut resp_body = Vec::new();
    copy_body(&mut conn, resp.body, Some(&mut resp_body))
        .map_err(|e| fail(format!("read body: {e}")))?;

    if resp.status != 200 {
        #[derive(Deserialize)]
        struct ConnectError {
            #[serde(default)]
            code: String,
            #[serde(default)]
            message: String,
        }
        let detail: ConnectError = serde_json::from_slice(&resp_body).unwrap_or(ConnectError {
            code: String::new(),
            message: String::from_utf8_lossy(&resp_body[..resp_body.len().min(256)]).into_owned(),
        });
        return Err(fail(format!(
            "status {}: {} {}",
            resp.status, detail.code, detail.message
        )));
    }

    serde_json::from_slice(&resp_body).map_err(|e| fail(format!("decode: {e}")))
}

/// Client for vault.v1.VaultService (Decrypt only).
pub struct VaultClient {
    base: SimpleUrl,
    token: String,
}

impl VaultClient {
    pub fn new(url: &str, token: &str) -> Result<Self, FrontlineError> {
        let base = SimpleUrl::parse(url).map_err(|e| {
            FrontlineError::new(
                urn::CONFIG_LOAD_FAILED,
                format!("invalid vault url: {e}"),
                "",
            )
        })?;
        Ok(Self {
            base,
            token: token.to_string(),
        })
    }

    /// Decrypts an encrypted blob within the given keyring (workspace).
    pub fn decrypt(&self, keyring: &str, encrypted: &str) -> Result<String, FrontlineError> {
        #[derive(Serialize)]
        struct DecryptRequest<'a> {
            keyring: &'a str,
            encrypted: &'a str,
        }
        #[derive(Deserialize)]
        struct DecryptResponse {
            #[serde(default)]
            plaintext: String,
        }
        let resp: DecryptResponse = call(
            &self.base,
            "/vault.v1.VaultService/Decrypt",
            Some(&self.token),
            &DecryptRequest { keyring, encrypted },
            Duration::from_secs(10),
        )?;
        Ok(resp.plaintext)
    }
}

/// Client for ctrl.v1.AcmeService (VerifyCertificate only).
pub struct AcmeClient {
    base: SimpleUrl,
}

impl AcmeClient {
    pub fn new(addr: &str) -> Result<Self, FrontlineError> {
        let base = SimpleUrl::parse(addr).map_err(|e| {
            FrontlineError::new(
                urn::CONFIG_LOAD_FAILED,
                format!("invalid ctrl addr: {e}"),
                "",
            )
        })?;
        Ok(Self { base })
    }

    /// Verifies an ACME HTTP-01 challenge and returns the authorization body
    /// to serve back to the ACME validator.
    pub fn verify_certificate(&self, domain: &str, token: &str) -> Result<String, FrontlineError> {
        #[derive(Serialize)]
        struct VerifyCertificateRequest<'a> {
            domain: &'a str,
            token: &'a str,
        }
        #[derive(Deserialize)]
        struct VerifyCertificateResponse {
            #[serde(default)]
            authorization: String,
        }
        let resp: VerifyCertificateResponse = call(
            &self.base,
            "/ctrl.v1.AcmeService/VerifyCertificate",
            None,
            &VerifyCertificateRequest { domain, token },
            Duration::from_secs(30),
        )?;
        Ok(resp.authorization)
    }
}
