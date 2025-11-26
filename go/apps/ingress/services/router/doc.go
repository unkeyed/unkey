// Package router handles routing lookups and gateway selection for the ingress.
//
// The router service is responsible for:
//   - Looking up ingress routes by hostname
//   - Finding available gateways for an environment
//   - Selecting the best gateway based on region proximity and health
//
// # Routing Strategy
//
// The router uses a simple and efficient strategy:
//
//   - If a healthy gateway exists in the local region, route to it directly
//   - If no local gateway, route to the nearest region's NLB
//
// # Example Flow
//
// Request to hostname in us-east-1, received in eu-west-1 (no local gateway):
//  1. eu-west-1 ingress receives request
//  2. Lookup shows environment has gateways in us-east-1, ap-south-1
//  3. eu-west-1 has no local gateway
//  4. Select nearest region with healthy gateway (us-east-1)
//  5. Forward to us-east-1 NLB
//  6. us-east-1 ingress routes to local gateway
package router
