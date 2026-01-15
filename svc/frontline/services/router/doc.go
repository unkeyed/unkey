// Package router handles routing lookups and sentinel selection for the frontline.
//
// The router service is responsible for:
//   - Looking up frontline routes by hostname
//   - Finding available sentinels for an environment
//   - Selecting the best sentinel based on region proximity and health
//
// # Routing Strategy
//
// The router uses a simple and efficient strategy:
//
//   - If a healthy sentinel exists in the local region, route to it directly
//   - If no local sentinel, route to the nearest region's NLB
//
// # Example Flow
//
// Request to hostname in us-east-1.aws, received in eu-west-1.aws (no local sentinel):
//  1. eu-west-1.aws frontline receives request
//  2. Lookup shows environment has sentinels in us-east-1.aws, ap-south-1.aws
//  3. eu-west-1.aws has no local sentinel
//  4. Select nearest region with healthy sentinel (us-east-1.aws)
//  5. Forward to us-east-1.aws NLB
//  6. us-east-1.aws frontline routes to local sentinel
package router
