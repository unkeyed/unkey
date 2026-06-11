//! Port of pkg/fault + pkg/codes as used by frontline.
//!
//! A [`FrontlineError`] carries a stable URN code, an internal message (logs
//! only), and a public message (shown to clients). The URN drives HTTP status
//! selection in the observability middleware, mirroring
//! `getErrorPageInfoFrontline` in middleware/observability.go.

use std::fmt;

/// Stable error code URNs, mirroring pkg/codes constants.
pub mod urn {
    pub const PROXY_BAD_GATEWAY: &str = "err:frontline:upstream:bad_gateway";
    pub const PROXY_SERVICE_UNAVAILABLE: &str = "err:frontline:upstream:service_unavailable";
    pub const PROXY_GATEWAY_TIMEOUT: &str = "err:frontline:upstream:gateway_timeout";
    pub const PROXY_FORWARD_FAILED: &str = "err:frontline:upstream:proxy_forward_failed";

    pub const ROUTING_CONFIG_NOT_FOUND: &str = "err:frontline:routing:config_not_found";
    pub const ROUTING_DEPLOYMENT_NOT_FOUND: &str = "err:frontline:routing:deployment_not_found";
    pub const ROUTING_DEPLOYMENT_SELECTION_FAILED: &str =
        "err:frontline:platform:deployment_selection_failed";
    pub const ROUTING_NO_RUNNING_INSTANCES: &str = "err:frontline:capacity:no_running_instances";

    pub const INTERNAL_SERVER_ERROR: &str = "err:frontline:platform:internal_server_error";
    pub const CONFIG_LOAD_FAILED: &str = "err:frontline:platform:config_load_failed";
    pub const INVALID_CONFIGURATION: &str = "err:frontline:config:invalid_configuration";

    pub const AUTH_MISSING_CREDENTIALS: &str = "err:frontline:client:missing_credentials";
    pub const AUTH_INVALID_KEY: &str = "err:frontline:client:invalid_key";
    pub const AUTH_INSUFFICIENT_PERMISSIONS: &str = "err:frontline:client:insufficient_permissions";
    pub const AUTH_RATE_LIMITED: &str = "err:frontline:client:rate_limited";

    pub const FIREWALL_DENIED: &str = "err:frontline:client:firewall_denied";
    pub const OPENAPI_INVALID_REQUEST: &str = "err:frontline:client:openapi_validation_failed";

    pub const CLIENT_CLOSED_REQUEST: &str = "err:user:bad_request:client_closed_request";
    pub const REQUEST_TIMEOUT: &str = "err:user:bad_request:request_timeout";
    pub const APP_UNEXPECTED_ERROR: &str = "err:unkey:application:unexpected_error";
}

/// Error produced anywhere on the request path. Equivalent to a
/// fault-wrapped error in the Go implementation.
#[derive(Debug, Clone)]
pub struct FrontlineError {
    /// Stable URN, e.g. "err:frontline:upstream:bad_gateway".
    pub urn: &'static str,
    /// Internal detail, logged but never shown to clients.
    pub internal: String,
    /// Public message rendered in the error page / JSON body. Empty string
    /// means "use the default message for this URN's status".
    pub public: String,
    /// True when the error happened during the dial phase of a proxy attempt,
    /// i.e. no TCP connection was ever established and the request body is
    /// untouched, so the request is safe to replay against another instance.
    /// Port of proxy.IsDialError.
    pub dial: bool,
}

impl FrontlineError {
    pub fn new(urn: &'static str, internal: impl Into<String>, public: impl Into<String>) -> Self {
        Self {
            urn,
            internal: internal.into(),
            public: public.into(),
            dial: false,
        }
    }

    pub fn dial(mut self) -> Self {
        self.dial = true;
        self
    }

    /// The attribution domain for metrics: the category segment of the URN
    /// (e.g. "client", "upstream", "platform"). Unattributed errors default
    /// to "platform" so they ring our own alert instead of vanishing.
    pub fn fault_domain(&self) -> &str {
        self.urn.split(':').nth(2).unwrap_or("platform")
    }

    /// Docs link surfaced on the error page next to the URN.
    pub fn docs_url(&self) -> String {
        format!(
            "https://unkey.com/docs/errors/{}",
            self.urn.replace(':', "/")
        )
    }
}

impl fmt::Display for FrontlineError {
    fn fmt(&self, f: &mut fmt::Formatter<'_>) -> fmt::Result {
        write!(f, "{}: {}", self.urn, self.internal)
    }
}

impl std::error::Error for FrontlineError {}

/// Page info derived from a URN: HTTP status, title, and default message.
/// Port of getErrorPageInfoFrontline in middleware/observability.go.
pub struct ErrorPageInfo {
    pub status: u16,
    pub title: &'static str,
    pub message: &'static str,
}

pub fn error_page_info(urn: &str) -> ErrorPageInfo {
    match urn {
        urn::CLIENT_CLOSED_REQUEST => ErrorPageInfo {
            status: 499,
            title: "Client Closed Request",
            message: "The client closed the connection before the request completed.",
        },
        urn::REQUEST_TIMEOUT => ErrorPageInfo {
            status: 504,
            title: "Gateway Timeout",
            message: "The request took too long to process. Please try again later.",
        },
        urn::ROUTING_CONFIG_NOT_FOUND => ErrorPageInfo {
            status: 404,
            title: "Not Found",
            message: "No deployment found for this hostname. Please check your domain configuration or contact support at support@unkey.com.",
        },
        urn::PROXY_BAD_GATEWAY | urn::PROXY_FORWARD_FAILED => ErrorPageInfo {
            status: 502,
            title: "Bad Gateway",
            message: "Unable to connect. Please try again in a few moments.",
        },
        urn::PROXY_SERVICE_UNAVAILABLE => ErrorPageInfo {
            status: 503,
            title: "Service Unavailable",
            message: "The service is temporarily unavailable. Please try again later.",
        },
        urn::PROXY_GATEWAY_TIMEOUT => ErrorPageInfo {
            status: 504,
            title: "Gateway Timeout",
            message: "The request took too long to process. Please try again later.",
        },
        urn::AUTH_MISSING_CREDENTIALS => ErrorPageInfo {
            status: 401,
            title: "Unauthorized",
            message: "Authentication required. Please provide a valid API key.",
        },
        urn::AUTH_INVALID_KEY => ErrorPageInfo {
            status: 401,
            title: "Unauthorized",
            message: "Authentication failed. The provided API key is invalid.",
        },
        urn::AUTH_INSUFFICIENT_PERMISSIONS => ErrorPageInfo {
            status: 403,
            title: "Forbidden",
            message: "Access denied. The API key does not have the required permissions.",
        },
        urn::AUTH_RATE_LIMITED => ErrorPageInfo {
            status: 429,
            title: "Too Many Requests",
            message: "Rate limit exceeded. Please try again later.",
        },
        urn::FIREWALL_DENIED => ErrorPageInfo {
            status: 403,
            title: "Forbidden",
            message: "Forbidden",
        },
        urn::OPENAPI_INVALID_REQUEST => ErrorPageInfo {
            status: 400,
            title: "Bad Request",
            message: "",
        },
        urn::ROUTING_DEPLOYMENT_NOT_FOUND => ErrorPageInfo {
            status: 404,
            title: "Not Found",
            message: "The requested deployment could not be found.",
        },
        urn::ROUTING_NO_RUNNING_INSTANCES => ErrorPageInfo {
            status: 503,
            title: "Service Unavailable",
            message: "No running instances are available to handle this request.",
        },
        urn::ROUTING_DEPLOYMENT_SELECTION_FAILED => ErrorPageInfo {
            status: 500,
            title: "Internal Server Error",
            message: "Failed to select an instance to handle your request.",
        },
        urn::INVALID_CONFIGURATION => ErrorPageInfo {
            // The deployment's own policy config could not be parsed. That is
            // the config author's problem, not a frontline fault — 422, not
            // 500, so it doesn't trigger server-error alerting.
            status: 422,
            title: "Unprocessable Entity",
            message: "The deployment configuration is invalid. Please check your config or contact support at support@unkey.com.",
        },
        urn::INTERNAL_SERVER_ERROR => ErrorPageInfo {
            status: 500,
            title: "Internal Server Error",
            message: "An unexpected error occurred. Please try again later.",
        },
        _ => ErrorPageInfo {
            status: 500,
            title: "Internal Server Error",
            message: "",
        },
    }
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn fault_domain_is_third_urn_segment() {
        let err = FrontlineError::new(urn::PROXY_BAD_GATEWAY, "x", "y");
        assert_eq!(err.fault_domain(), "upstream");
        let err = FrontlineError::new(urn::AUTH_INVALID_KEY, "x", "y");
        assert_eq!(err.fault_domain(), "client");
    }

    #[test]
    fn status_mapping_matches_go() {
        assert_eq!(error_page_info(urn::CLIENT_CLOSED_REQUEST).status, 499);
        assert_eq!(error_page_info(urn::FIREWALL_DENIED).status, 403);
        assert_eq!(error_page_info(urn::INVALID_CONFIGURATION).status, 422);
        assert_eq!(error_page_info(urn::AUTH_RATE_LIMITED).status, 429);
        assert_eq!(error_page_info("err:something:unknown:unknown").status, 500);
    }
}
