// Package deployments provides services for looking up and routing to tenant deployments.
//
// The deployment service is responsible for:
//   - Looking up deployment configuration by hostname
//   - Selecting the closest available deployment based on region proximity
//   - Supporting multi-region deployments with automatic failover
//
// # Multi-Region Support
//
// The deployment service supports multi-region deployments where a single hostname
// can be deployed across multiple regions. When looking up a deployment:
//
//   - All enabled deployments with running instances are considered
//   - The closest available region is selected based on:
//     1. Local region (if available)
//     2. Proximity map (nearest region)
//     3. Any available region (fallback)
//
// # Routing Strategy
//
// The deployment service uses a simple and efficient routing strategy:
//
//   - If the deployment is in the current region, route to local gateway (HTTP)
//   - If the deployment is in a different region, route directly to that region's
//     ingress (HTTPS)
//
// This direct routing approach ensures:
//   - Minimal hops (max 1 hop to reach the deployment)
//   - No intermediate regions needed
//   - Predictable latency
//   - No risk of routing loops
//
// # Example Flow
//
// Request to deployment in ap-south-1, received in us-east-1:
//  1. us-east-1 ingress receives request
//  2. Lookup shows deployment is in ap-south-1 (different region)
//  3. Request forwarded DIRECTLY to ap-south-1 ingress via HTTPS
//  4. ap-south-1 ingress routes to local gateway (same region)
package deployments
