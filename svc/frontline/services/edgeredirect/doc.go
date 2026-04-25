// Package edgeredirect evaluates ordered redirect rules against incoming
// HTTP requests at the frontline edge. The engine is a pure function over
// (rules, *http.Request) returning either nil (no redirect) or a *Result
// describing the target Location and status code; callers are responsible
// for writing the response.
//
// Rules are protobuf messages (frontline.edgeredirect.v1.Rule) so the same
// shape stores in MySQL (frontline_routes.edge_redirect_config), travels
// over the wire to the dashboard, and feeds the engine without conversion.
//
// Two consumers exist today:
//
//   - The frontline HTTP listener bakes a single RequireHTTPS rule into a
//     catchall handler so every plain-HTTP request gets a 308 to its
//     https:// equivalent. ACME paths are more specific and still win.
//
//   - The frontline HTTPS proxy handler evaluates the per-FQDN rule set
//     (parsed once at router cache fill time) before forwarding to the
//     sentinel.
package edgeredirect
