package deploy

// instancesReadyAwakeableKey is the Restate virtual object state key under
// which a Deploy handler stashes the awakeable it's parked on while waiting
// for ReportDeploymentStatus to call NotifyInstancesReady.
//
// DeployService is keyed by deployment_id, so each VO instance owns the
// awakeable for exactly one deployment — no cross-deployment guard needed.
const instancesReadyAwakeableKey = "instances_ready_awakeable"
