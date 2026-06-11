//! Port of svc/frontline/internal/db (sqlc-generated queries) on the
//! synchronous `mysql` driver. Routing and certificate lookups read from the
//! read-only replica; when no replica DSN is configured the pool targets the
//! primary, same as the Go service.

use mysql::prelude::Queryable;
use mysql::{Opts, OptsBuilder, Pool, PoolConstraints, PoolOpts, Row};

use crate::error::{urn, FrontlineError};

/// Upstream protocol configured on the deployment. h2c is accepted but
/// forwarded over HTTP/1.1 in this port (see httpx module docs); unknown
/// values also fall back to http1, mirroring the Go TransportRegistry.
#[derive(Debug, Clone, Copy, PartialEq, Eq, Hash)]
pub enum UpstreamProtocol {
    Http1,
    H2c,
    Other,
}

impl From<&str> for UpstreamProtocol {
    fn from(s: &str) -> Self {
        match s {
            "http1" => Self::Http1,
            "h2c" => Self::H2c,
            _ => Self::Other,
        }
    }
}

#[derive(Debug, Clone, Copy, PartialEq, Eq)]
pub enum InstanceStatus {
    Inactive,
    Pending,
    Running,
    Failed,
    Other,
}

impl From<&str> for InstanceStatus {
    fn from(s: &str) -> Self {
        match s {
            "inactive" => Self::Inactive,
            "pending" => Self::Pending,
            "running" => Self::Running,
            "failed" => Self::Failed,
            _ => Self::Other,
        }
    }
}

/// Row of FindFrontlineRouteByFQDN.
#[derive(Debug, Clone)]
pub struct FrontlineRouteRow {
    pub environment_id: String,
    pub deployment_id: String,
    pub upstream_protocol: UpstreamProtocol,
}

/// Row of FindInstancesByDeploymentID (the fields frontline reads).
#[derive(Debug, Clone)]
pub struct InstanceRow {
    pub id: String,
    pub workspace_id: String,
    pub project_id: String,
    pub address: String,
    pub status: InstanceStatus,
    pub region_name: String,
    pub region_platform: String,
}

/// Row of FindBestCertificateByCandidates.
#[derive(Debug, Clone)]
pub struct CertificateRow {
    pub hostname: String,
    pub workspace_id: String,
    pub certificate: String,
    pub encrypted_private_key: String,
}

fn wrap(err: impl std::fmt::Display, what: &str) -> FrontlineError {
    FrontlineError::new(
        urn::CONFIG_LOAD_FAILED,
        format!("{what}: {err}"),
        "Failed to load route configuration",
    )
}

/// Converts a Go-style MySQL DSN ("user:pass@tcp(host:port)/db?parseTime=true")
/// into the URL form the driver expects. URLs pass through unchanged so both
/// formats work in config files.
pub fn normalize_dsn(dsn: &str) -> String {
    if dsn.starts_with("mysql://") || dsn.starts_with("mariadb://") {
        return dsn.to_string();
    }
    // user[:pass]@tcp(host:port)/db[?params]
    if let Some(tcp_start) = dsn.find("@tcp(") {
        let creds = &dsn[..tcp_start];
        let rest = &dsn[tcp_start + 5..];
        if let Some(paren) = rest.find(')') {
            let hostport = &rest[..paren];
            let tail = rest[paren + 1..].trim_start_matches('/');
            // Strip Go-driver-specific query params like parseTime.
            let db = tail.split('?').next().unwrap_or(tail);
            return format!("mysql://{creds}@{hostport}/{db}");
        }
    }
    dsn.to_string()
}

#[derive(Clone)]
pub struct Database {
    pool: Pool,
}

impl Database {
    pub fn connect(dsn: &str) -> Result<Self, FrontlineError> {
        let opts = Opts::from_url(&normalize_dsn(dsn)).map_err(|e| {
            FrontlineError::new(
                urn::CONFIG_LOAD_FAILED,
                format!("invalid database DSN: {e}"),
                "",
            )
        })?;
        // Lazy pool (no eager connections): readiness is checked explicitly
        // via ping() before the listeners start.
        let constraints = PoolConstraints::new(0, 20).expect("0 <= 20");
        let opts = OptsBuilder::from_opts(opts)
            .pool_opts(PoolOpts::default().with_constraints(constraints));
        let pool = Pool::new(opts).map_err(|e| {
            FrontlineError::new(
                urn::CONFIG_LOAD_FAILED,
                format!("unable to connect to database: {e}"),
                "",
            )
        })?;
        Ok(Self { pool })
    }

    pub fn ping(&self) -> Result<(), FrontlineError> {
        let mut conn = self.pool.get_conn().map_err(|e| wrap(e, "ping"))?;
        conn.query_drop("SELECT 1").map_err(|e| wrap(e, "ping"))
    }

    /// FindFrontlineRouteByFQDN: hostname -> deployment + policy bytes +
    /// upstream protocol in a single round trip.
    pub fn find_frontline_route_by_fqdn(
        &self,
        fqdn: &str,
    ) -> Result<Option<FrontlineRouteRow>, FrontlineError> {
        let mut conn = self
            .pool
            .get_conn()
            .map_err(|e| wrap(e, "FindFrontlineRouteByFQDN"))?;
        let row: Option<Row> = conn
            .exec_first(
                r#"
                SELECT
                  fr.environment_id,
                  fr.deployment_id,
                  d.upstream_protocol
                FROM frontline_routes fr
                INNER JOIN deployments d ON fr.deployment_id = d.id
                WHERE fr.fully_qualified_domain_name = ?
                "#,
                (fqdn,),
            )
            .map_err(|e| wrap(e, "FindFrontlineRouteByFQDN"))?;

        Ok(row.map(|mut r| FrontlineRouteRow {
            environment_id: r.take("environment_id").unwrap_or_default(),
            deployment_id: r.take("deployment_id").unwrap_or_default(),
            upstream_protocol: UpstreamProtocol::from(
                r.take::<Option<String>, _>("upstream_protocol")
                    .flatten()
                    .unwrap_or_default()
                    .as_str(),
            ),
        }))
    }

    /// FindInstancesByDeploymentID: all instances with region metadata.
    pub fn find_instances_by_deployment_id(
        &self,
        deployment_id: &str,
    ) -> Result<Vec<InstanceRow>, FrontlineError> {
        let mut conn = self
            .pool
            .get_conn()
            .map_err(|e| wrap(e, "FindInstancesByDeploymentID"))?;
        let rows: Vec<Row> = conn
            .exec(
                r#"
                SELECT
                  i.id,
                  i.workspace_id,
                  i.project_id,
                  i.address,
                  i.status,
                  r.name AS region_name,
                  r.platform AS region_platform
                FROM instances i
                INNER JOIN regions r ON i.region_id = r.id
                WHERE i.deployment_id = ?
                "#,
                (deployment_id,),
            )
            .map_err(|e| wrap(e, "FindInstancesByDeploymentID"))?;

        Ok(rows
            .into_iter()
            .map(|mut r| InstanceRow {
                id: r.take("id").unwrap_or_default(),
                workspace_id: r.take("workspace_id").unwrap_or_default(),
                project_id: r.take("project_id").unwrap_or_default(),
                address: r.take("address").unwrap_or_default(),
                status: InstanceStatus::from(
                    r.take::<Option<String>, _>("status")
                        .flatten()
                        .unwrap_or_default()
                        .as_str(),
                ),
                region_name: r.take("region_name").unwrap_or_default(),
                region_platform: r.take("region_platform").unwrap_or_default(),
            })
            .collect())
    }

    /// FindBestCertificateByCandidates: one certificate row for the
    /// candidates, preferring an exact hostname over wildcard matches.
    pub fn find_best_certificate_by_candidates(
        &self,
        hostnames: &[String],
        exact_hostname: &str,
    ) -> Result<Option<CertificateRow>, FrontlineError> {
        if hostnames.is_empty() {
            return Ok(None);
        }
        let mut conn = self
            .pool
            .get_conn()
            .map_err(|e| wrap(e, "FindBestCertificateByCandidates"))?;
        let placeholders = vec!["?"; hostnames.len()].join(",");
        let sql = format!(
            r#"
            SELECT hostname, workspace_id, certificate, encrypted_private_key
            FROM certificates
            WHERE hostname IN ({placeholders})
            ORDER BY hostname = ? DESC
            LIMIT 1
            "#
        );
        let mut params: Vec<mysql::Value> = hostnames
            .iter()
            .map(|h| mysql::Value::from(h.as_str()))
            .collect();
        params.push(mysql::Value::from(exact_hostname));

        let row: Option<Row> = conn
            .exec_first(&sql, params)
            .map_err(|e| wrap(e, "FindBestCertificateByCandidates"))?;

        Ok(row.map(|mut r| CertificateRow {
            hostname: r.take("hostname").unwrap_or_default(),
            workspace_id: r.take("workspace_id").unwrap_or_default(),
            certificate: r.take("certificate").unwrap_or_default(),
            encrypted_private_key: r.take("encrypted_private_key").unwrap_or_default(),
        }))
    }

    /// FindCustomDomainIDByDomain: ownership check for ACME HTTP-01.
    pub fn find_custom_domain_id_by_domain(
        &self,
        domain: &str,
    ) -> Result<Option<String>, FrontlineError> {
        let mut conn = self
            .pool
            .get_conn()
            .map_err(|e| wrap(e, "FindCustomDomainIDByDomain"))?;
        conn.exec_first("SELECT id FROM custom_domains WHERE domain = ?", (domain,))
            .map_err(|e| wrap(e, "FindCustomDomainIDByDomain"))
    }
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn normalizes_go_dsn() {
        assert_eq!(
            normalize_dsn("unkey:pw@tcp(mysql.local:3306)/unkey?parseTime=true"),
            "mysql://unkey:pw@mysql.local:3306/unkey"
        );
        assert_eq!(
            normalize_dsn("mysql://unkey:pw@mysql.local:3306/unkey"),
            "mysql://unkey:pw@mysql.local:3306/unkey"
        );
    }

    #[test]
    fn parses_enums() {
        assert_eq!(UpstreamProtocol::from("http1"), UpstreamProtocol::Http1);
        assert_eq!(UpstreamProtocol::from("h2c"), UpstreamProtocol::H2c);
        assert_eq!(UpstreamProtocol::from("h3"), UpstreamProtocol::Other);
        assert_eq!(InstanceStatus::from("running"), InstanceStatus::Running);
    }
}
