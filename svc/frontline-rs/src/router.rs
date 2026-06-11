//! Port of internal/router: resolves a hostname to a forwarding decision —
//! either local deployment instances (with a standby peer region) or a peer
//! frontline in another region.

use std::sync::Arc;
use std::time::{Duration, Instant};

use crate::cache::SwrCache;
use crate::db::{Database, FrontlineRouteRow, InstanceRow, InstanceStatus};
use crate::error::{urn, FrontlineError};
use crate::metrics;
use crate::uid;

/// Static proximity table: for each region, peers in latency order. Port of
/// regionProximity in routing.go (incomplete in places by design; map-order
/// fallback handles the gaps).
fn region_proximity(region_platform: &str) -> Option<&'static [&'static str]> {
    Some(match region_platform {
        // US East
        "us-east-1.aws" => &[
            "us-east-2.aws",
            "us-west-2.aws",
            "us-west-1.aws",
            "ca-central-1.aws",
            "eu-west-1.aws",
            "eu-central-1.aws",
            "ap-northeast-1.aws",
            "ap-southeast-1.aws",
            "ap-southeast-2.aws",
        ],
        "us-east-2.aws" => &[
            "us-east-1.aws",
            "us-west-2.aws",
            "us-west-1.aws",
            "ca-central-1.aws",
            "eu-west-1.aws",
            "eu-central-1.aws",
            "ap-northeast-1.aws",
            "ap-southeast-1.aws",
            "ap-southeast-2.aws",
        ],
        // US West
        "us-west-1.aws" => &[
            "us-west-2.aws",
            "us-east-2.aws",
            "us-east-1.aws",
            "ca-central-1.aws",
            "ap-northeast-1.aws",
            "ap-southeast-1.aws",
            "ap-southeast-2.aws",
            "eu-west-1.aws",
            "eu-central-1.aws",
        ],
        "us-west-2.aws" => &[
            "us-west-1.aws",
            "us-east-2.aws",
            "us-east-1.aws",
            "ca-central-1.aws",
            "ap-northeast-1.aws",
            "ap-southeast-1.aws",
            "ap-southeast-2.aws",
            "eu-west-1.aws",
            "eu-central-1.aws",
        ],
        // Canada
        "ca-central-1.aws" => &[
            "us-east-2.aws",
            "us-east-1.aws",
            "us-west-2.aws",
            "us-west-1.aws",
            "eu-west-1.aws",
            "eu-central-1.aws",
            "ap-northeast-1.aws",
            "ap-southeast-1.aws",
            "ap-southeast-2.aws",
        ],
        // Europe
        "eu-west-1.aws" => &[
            "eu-west-2.aws",
            "eu-central-1.aws",
            "eu-north-1.aws",
            "us-east-1.aws",
            "us-east-2.aws",
            "us-west-2.aws",
            "us-west-1.aws",
            "ap-south-1.aws",
            "ap-southeast-1.aws",
        ],
        "eu-west-2.aws" => &[
            "eu-west-1.aws",
            "eu-central-1.aws",
            "eu-north-1.aws",
            "us-east-1.aws",
            "us-east-2.aws",
            "us-west-2.aws",
            "us-west-1.aws",
            "ap-south-1.aws",
            "ap-southeast-1.aws",
        ],
        "eu-central-1.aws" => &[
            "eu-west-1.aws",
            "eu-west-2.aws",
            "eu-north-1.aws",
            "us-east-1.aws",
            "us-east-2.aws",
            "us-west-2.aws",
            "us-west-1.aws",
            "ap-south-1.aws",
            "ap-southeast-1.aws",
        ],
        "eu-north-1.aws" => &[
            "eu-central-1.aws",
            "eu-west-1.aws",
            "eu-west-2.aws",
            "us-east-1.aws",
            "us-east-2.aws",
            "us-west-2.aws",
            "us-west-1.aws",
            "ap-south-1.aws",
            "ap-southeast-1.aws",
        ],
        // Asia Pacific
        "ap-south-1.aws" => &[
            "ap-southeast-1.aws",
            "ap-southeast-2.aws",
            "ap-northeast-1.aws",
            "eu-west-1.aws",
            "eu-central-1.aws",
            "us-west-2.aws",
            "us-west-1.aws",
            "us-east-1.aws",
            "us-east-2.aws",
        ],
        "ap-northeast-1.aws" => &[
            "ap-southeast-1.aws",
            "ap-southeast-2.aws",
            "ap-south-1.aws",
            "us-west-2.aws",
            "us-west-1.aws",
            "us-east-1.aws",
            "us-east-2.aws",
            "eu-west-1.aws",
            "eu-central-1.aws",
        ],
        "ap-southeast-1.aws" => &[
            "ap-southeast-2.aws",
            "ap-northeast-1.aws",
            "ap-south-1.aws",
            "us-west-2.aws",
            "us-west-1.aws",
            "us-east-1.aws",
            "us-east-2.aws",
            "eu-west-1.aws",
            "eu-central-1.aws",
        ],
        "ap-southeast-2.aws" => &[
            "ap-southeast-1.aws",
            "ap-northeast-1.aws",
            "ap-south-1.aws",
            "us-west-2.aws",
            "us-west-1.aws",
            "us-east-1.aws",
            "us-east-2.aws",
            "eu-west-1.aws",
            "eu-central-1.aws",
        ],
        // Local development
        "local.dev" => &[],
        _ => return None,
    })
}

#[derive(Debug, Clone, Copy, PartialEq, Eq)]
pub enum Destination {
    /// Route to a deployment instance running in this region.
    LocalInstance,
    /// Forward to a peer frontline in another region, which redoes the full
    /// hostname -> instance chain.
    RemoteRegion,
}

/// The output of route().
///
/// For LocalInstance: local_instances carries the candidate pods in shuffled
/// order — the caller attempts them sequentially, advancing on dial
/// failures. remote_region_address, when non-empty, is a standby peer region
/// to fall through to once every local instance has dial-failed.
///
/// For RemoteRegion: only remote_region_address is populated.
#[derive(Debug, Clone)]
pub struct RouteDecision {
    pub destination: Destination,
    pub deployment_id: String,
    pub environment_id: String,
    pub workspace_id: String,
    pub project_id: String,
    pub local_instances: Vec<InstanceRow>,
    pub remote_region_address: String,
    pub upstream_protocol: crate::db::UpstreamProtocol,
}

pub struct Router {
    region_platform: String,
    db: Database,
    route_cache: Arc<SwrCache<FrontlineRouteRow>>,
    instances_cache: Arc<SwrCache<Vec<InstanceRow>>>,
}

impl Router {
    pub fn new(platform: &str, region: &str, db: Database) -> Self {
        Self {
            region_platform: format!("{region}.{platform}"),
            db,
            // frontline_route: Fresh 5s / Stale 5m / 10k entries.
            route_cache: SwrCache::new(
                "frontline_route",
                Duration::from_secs(5),
                Duration::from_secs(300),
                10_000,
            ),
            // instances_by_deployment: Fresh 10s / Stale 60s / 10k entries.
            instances_cache: SwrCache::new(
                "instances_by_deployment",
                Duration::from_secs(10),
                Duration::from_secs(60),
                10_000,
            ),
        }
    }

    /// Resolves a hostname to a forwarding decision.
    pub fn route(&self, hostname: &str) -> Result<RouteDecision, FrontlineError> {
        let start = Instant::now();
        let result = self.route_inner(hostname);
        metrics::ROUTING_DECISION_SECONDS
            .histogram_with(&[])
            .observe(start.elapsed().as_secs_f64());

        if let Ok(decision) = &result {
            match decision.destination {
                Destination::LocalInstance => metrics::ROUTING_DECISIONS_TOTAL
                    .counter_with(&["local", &self.region_platform])
                    .inc(),
                Destination::RemoteRegion => metrics::ROUTING_DECISIONS_TOTAL
                    .counter_with(&["remote", &decision.remote_region_address])
                    .inc(),
            }
        }
        result
    }

    fn route_inner(&self, hostname: &str) -> Result<RouteDecision, FrontlineError> {
        let route = self.find_route(hostname)?;
        let instances = self.get_instances(&route.deployment_id)?;
        self.select_destination(&route, instances)
    }

    /// Checks if a hostname has a configured frontline route. Part of the Go
    /// router.Service interface; kept for callers outside the request path.
    #[allow(dead_code)]
    pub fn validate_hostname(&self, hostname: &str) -> Result<(), FrontlineError> {
        self.find_route(hostname).map(|_| ())
    }

    fn find_route(&self, hostname: &str) -> Result<FrontlineRouteRow, FrontlineError> {
        let db = self.db.clone();
        let fqdn = hostname.to_string();
        let route = self
            .route_cache
            .swr(hostname, move || db.find_frontline_route_by_fqdn(&fqdn))?;

        route.ok_or_else(|| {
            FrontlineError::new(
                urn::ROUTING_CONFIG_NOT_FOUND,
                format!("no frontline route for hostname: {hostname}"),
                "Domain not configured",
            )
        })
    }

    fn get_instances(&self, deployment_id: &str) -> Result<Vec<InstanceRow>, FrontlineError> {
        let db = self.db.clone();
        let id = deployment_id.to_string();
        let instances = self.instances_cache.swr(deployment_id, move || {
            // An empty list is a real (cacheable) result, not a null.
            db.find_instances_by_deployment_id(&id).map(Some)
        })?;
        Ok(instances.unwrap_or_default())
    }

    /// Decides whether to proxy locally or forward to a peer frontline.
    /// Port of selectDestination in routing.go.
    fn select_destination(
        &self,
        route: &FrontlineRouteRow,
        instances: Vec<InstanceRow>,
    ) -> Result<RouteDecision, FrontlineError> {
        if instances.is_empty() {
            return Err(FrontlineError::new(
                urn::ROUTING_NO_RUNNING_INSTANCES,
                format!("no instances for deployment {}", route.deployment_id),
                "Service temporarily unavailable",
            ));
        }

        let mut local_running: Vec<InstanceRow> = Vec::new();
        let mut regions_with_instance: Vec<String> = Vec::new();
        for inst in instances {
            if inst.status != InstanceStatus::Running {
                continue;
            }
            let key = format!("{}.{}", inst.region_name, inst.region_platform);
            if !regions_with_instance.contains(&key) {
                regions_with_instance.push(key.clone());
            }
            if key == self.region_platform {
                local_running.push(inst);
            }
        }

        if regions_with_instance.is_empty() {
            return Err(FrontlineError::new(
                urn::ROUTING_NO_RUNNING_INSTANCES,
                format!(
                    "no running instances for deployment {}",
                    route.deployment_id
                ),
                "Service temporarily unavailable",
            ));
        }

        if !local_running.is_empty() {
            uid::shuffle(&mut local_running);
            // Pick a standby peer region for the handler to fall through to
            // if every local instance dial-fails. Empty when this is the only
            // region with running instances.
            let standby = self.find_nearest_region_platform(&regions_with_instance);
            return Ok(RouteDecision {
                destination: Destination::LocalInstance,
                deployment_id: route.deployment_id.clone(),
                environment_id: route.environment_id.clone(),
                workspace_id: local_running[0].workspace_id.clone(),
                project_id: local_running[0].project_id.clone(),
                upstream_protocol: route.upstream_protocol,
                local_instances: local_running,
                remote_region_address: standby,
            });
        }

        let nearest = self.find_nearest_region_platform(&regions_with_instance);
        if nearest.is_empty() {
            return Err(FrontlineError::new(
                urn::ROUTING_NO_RUNNING_INSTANCES,
                format!("no reachable region from {}", self.region_platform),
                "Service temporarily unavailable",
            ));
        }

        Ok(RouteDecision {
            destination: Destination::RemoteRegion,
            deployment_id: route.deployment_id.clone(),
            environment_id: route.environment_id.clone(),
            workspace_id: String::new(),
            project_id: String::new(),
            local_instances: Vec::new(),
            remote_region_address: nearest,
            upstream_protocol: route.upstream_protocol,
        })
    }

    /// The peer region (other than the local one) most likely to be
    /// reachable. The proximity table is the primary order; regions missing
    /// from it fall back to first-seen order so we return *something*
    /// reachable instead of failing closed.
    fn find_nearest_region_platform(&self, regions_with_instance: &[String]) -> String {
        if let Some(proximity) = region_proximity(&self.region_platform) {
            for region in proximity {
                if regions_with_instance.iter().any(|r| r == region) {
                    return region.to_string();
                }
            }
        }
        for region in regions_with_instance {
            if region != &self.region_platform {
                return region.clone();
            }
        }
        String::new()
    }
}

#[cfg(test)]
mod tests {
    use super::*;
    use crate::db::UpstreamProtocol;

    fn router(region: &str, platform: &str) -> Router {
        // Database handle is unused by select_destination; connect lazily by
        // building Router fields directly.
        Router {
            region_platform: format!("{region}.{platform}"),
            db: test_db(),
            route_cache: SwrCache::new("r", Duration::from_secs(1), Duration::from_secs(2), 10),
            instances_cache: SwrCache::new("i", Duration::from_secs(1), Duration::from_secs(2), 10),
        }
    }

    fn test_db() -> Database {
        // A pool that never connects (lazy); fine for pure-logic tests.
        Database::connect("mysql://unused@127.0.0.1:1/none").expect("lazy pool")
    }

    fn instance(id: &str, region: &str, platform: &str, status: InstanceStatus) -> InstanceRow {
        InstanceRow {
            id: id.into(),
            workspace_id: "ws_1".into(),
            project_id: "p_1".into(),
            address: format!("{id}.internal:8080"),
            status,
            region_name: region.into(),
            region_platform: platform.into(),
        }
    }

    fn route_row() -> FrontlineRouteRow {
        FrontlineRouteRow {
            environment_id: "env_1".into(),
            deployment_id: "dep_1".into(),
            upstream_protocol: UpstreamProtocol::Http1,
        }
    }

    #[test]
    fn no_instances_is_no_running_instances() {
        let r = router("us-east-1", "aws");
        let err = r.select_destination(&route_row(), vec![]).unwrap_err();
        assert_eq!(err.urn, urn::ROUTING_NO_RUNNING_INSTANCES);
    }

    #[test]
    fn only_non_running_instances_is_an_error() {
        let r = router("us-east-1", "aws");
        let err = r
            .select_destination(
                &route_row(),
                vec![instance("i1", "us-east-1", "aws", InstanceStatus::Pending)],
            )
            .unwrap_err();
        assert_eq!(err.urn, urn::ROUTING_NO_RUNNING_INSTANCES);
    }

    #[test]
    fn local_instances_win_and_carry_standby() {
        let r = router("us-east-1", "aws");
        let d = r
            .select_destination(
                &route_row(),
                vec![
                    instance("i1", "us-east-1", "aws", InstanceStatus::Running),
                    instance("i2", "us-east-1", "aws", InstanceStatus::Running),
                    instance("i3", "eu-west-1", "aws", InstanceStatus::Running),
                    instance("i4", "us-east-1", "aws", InstanceStatus::Failed),
                ],
            )
            .unwrap();
        assert_eq!(d.destination, Destination::LocalInstance);
        assert_eq!(d.local_instances.len(), 2);
        assert_eq!(d.workspace_id, "ws_1");
        // Standby is the nearest peer region with running instances.
        assert_eq!(d.remote_region_address, "eu-west-1.aws");
    }

    #[test]
    fn no_local_instances_routes_to_nearest_region() {
        let r = router("us-east-1", "aws");
        let d = r
            .select_destination(
                &route_row(),
                vec![
                    instance("i1", "eu-west-1", "aws", InstanceStatus::Running),
                    instance("i2", "us-east-2", "aws", InstanceStatus::Running),
                ],
            )
            .unwrap();
        assert_eq!(d.destination, Destination::RemoteRegion);
        // us-east-2 is first in us-east-1's proximity list.
        assert_eq!(d.remote_region_address, "us-east-2.aws");
        assert!(d.local_instances.is_empty());
    }

    #[test]
    fn unknown_region_falls_back_to_any_peer() {
        let r = router("mars-1", "spacex");
        let d = r
            .select_destination(
                &route_row(),
                vec![instance("i1", "eu-west-1", "aws", InstanceStatus::Running)],
            )
            .unwrap();
        assert_eq!(d.destination, Destination::RemoteRegion);
        assert_eq!(d.remote_region_address, "eu-west-1.aws");
    }

    #[test]
    fn only_region_with_instances_has_no_standby() {
        let r = router("us-east-1", "aws");
        let d = r
            .select_destination(
                &route_row(),
                vec![instance("i1", "us-east-1", "aws", InstanceStatus::Running)],
            )
            .unwrap();
        assert_eq!(d.destination, Destination::LocalInstance);
        assert_eq!(d.remote_region_address, "");
    }
}
