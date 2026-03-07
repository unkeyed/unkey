# Why Deployments over ReplicaSets for customer workloads

## The problem with ReplicaSets

ReplicaSets don't roll pods when the pod template changes. They only ensure the desired replica count is met — if the template changes on an existing RS, running pods keep the old spec indefinitely.

This meant that infra-level changes — env var overrides, resource defaults, security settings, runtime class updates — would **not take effect** on existing customer workloads until the customer triggered a new deploy. Each user redeploy created a new deployment ID (and therefore a new RS), so the issue was masked during normal deploys. But any change we made to the pod template from our side was silently ignored for all running workloads.

## Why Deployments fix this

A Kubernetes Deployment owns ReplicaSets and performs rolling updates automatically when the pod template changes. When we update the Deployment spec (e.g. change a resource limit, add an env var override, update the runtime class), the Deployment controller:

1. Creates a new RS with the updated template
2. Scales up the new RS while scaling down the old one
3. Respects the rolling update strategy (`MaxUnavailable: 0, MaxSurge: 1`) for zero-downtime transitions

This gives us the ability to push infra-level changes to all running workloads without requiring customers to redeploy.

## Rolling update strategy

We use `MaxUnavailable: 0, MaxSurge: 1`:

- **MaxUnavailable: 0** — never take down a running pod before its replacement is ready. This ensures zero downtime during rolls.
- **MaxSurge: 1** — allow at most 1 extra pod above the desired count during the roll. This keeps resource usage bounded while still making progress.

## Migration path

Existing ReplicaSets from the previous controller are cleaned up automatically:

- The resync loop identifies orphan RS (those with krane labels but no ownerReferences, meaning they weren't created by a Deployment) and deletes them.
- New Deployments are created with the same name and labels, so the transition is seamless from the control plane's perspective.

## Sentinels already use this pattern

The sentinel controller already uses Deployments. This change aligns customer workloads with the same pattern, reducing the number of K8s resource types we need to reason about.
