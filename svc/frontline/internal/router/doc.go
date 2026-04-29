// Package router resolves a hostname to a deployment, parses the deployment's
// policies, and decides whether the request runs locally (engine + proxy to a
// running instance in this region) or hops to a peer frontline in another
// region.
//
// # Routing Strategy
//
//   - Look up the frontline route for the hostname (cached SWR).
//   - Load the deployment's instances (cached SWR) and the parsed policies.
//   - If a Running instance exists in the local region, route locally.
//   - Otherwise pick the nearest region that has a running instance and forward
//     to that region's peer frontline; the peer redoes the full chain.
//
// # Example Flow
//
// Request to hostname owned by a deployment running in us-east-1.aws,
// received in eu-west-1.aws which has no local instance:
//  1. eu-west-1.aws frontline receives request.
//  2. Hostname → frontline_route → deployment_id, sentinel_config, upstream_protocol.
//  3. Deployment instances live in us-east-1.aws and ap-south-1.aws.
//  4. Local region (eu-west-1.aws) has no running instance.
//  5. Pick nearest region with a running instance (us-east-1.aws).
//  6. Forward to https://frontline.us-east-1.aws.<apex>; peer runs the engine
//     and proxies to its local instance.
package router
