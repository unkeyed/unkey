// Package deployments provides services for looking up and routing to tenant deployments.
//
// The deployment service is responsible for:
//   - Looking up deployment configuration by hostname
//   - Determining if a deployment is local (in the current region) or remote
//   - Computing optimal routing paths between regions based on proximity
//
// # Routing Strategy
//
// The deployment service uses a simple and efficient routing strategy:
//
//   - If the deployment is in the current region, route to local gateway (HTTP)
//   - If the deployment is in a different region, route directly to that region's
//     ingress (HTTPS), regardless of distance
//
// This direct routing approach ensures:
//   - Minimal hops (max 1 hop to reach the deployment)
//   - No intermediate regions needed
//   - Predictable latency
//   - No risk of routing loops
//
// The proximity map is kept for future optimization (like finding the closest
// region with an active replica), but currently unused.
//
// # Example Flow
//
// Request to deployment in ap-south-1, received in us-east-1:
//  1. us-east-1 ingress receives request
//  2. Lookup shows deployment is in ap-south-1 (IsLocal returns false)
//  3. Request forwarded DIRECTLY to ap-south-1 ingress via HTTPS
//  4. ap-south-1 ingress routes to local gateway (IsLocal returns true)
package deployments
