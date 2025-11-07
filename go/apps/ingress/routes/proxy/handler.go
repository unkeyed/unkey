package handler

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httputil"
	"net/url"

	"github.com/unkeyed/unkey/go/apps/ingress/services/deployments"
	partitionv1 "github.com/unkeyed/unkey/go/gen/proto/partition/v1"
	"github.com/unkeyed/unkey/go/pkg/codes"
	"github.com/unkeyed/unkey/go/pkg/fault"
	"github.com/unkeyed/unkey/go/pkg/otel/logging"
	"github.com/unkeyed/unkey/go/pkg/zen"
)

type Handler struct {
	Logger            logging.Logger
	DeploymentService *deployments.Service
	CurrentRegion     string
	BaseDomain        string
}

func (h *Handler) Method() string {
	return zen.CATCHALL
}

func (h *Handler) Path() string {
	return "/{path...}"
}

// Handle routes incoming requests to either:
// 1. Local gateway (if deployment is in current region) - forwards to HTTP
// 2. Remote ingress (if deployment is elsewhere) - forwards to HTTPS
func (h *Handler) Handle(ctx context.Context, sess *zen.Session) error {
	hostname := sess.Request().Host

	// Lookup deployment by hostname
	deployment, found, err := h.DeploymentService.LookupByHostname(ctx, hostname)
	if err != nil {
		h.Logger.Error("failed to lookup deployment", "hostname", hostname, "error", err)
		return fault.Wrap(err,
			fault.Code(codes.Gateway.Internal.InternalServerError.URN()),
			fault.Internal("deployment lookup failed"),
			fault.Public("Unable to process request"),
		)
	}

	// Deployment not found
	if !found {
		h.Logger.Warn("deployment not found", "hostname", hostname)
		return fault.New("deployment not found",
			fault.Code(codes.Gateway.Routing.ConfigNotFound.URN()),
			fault.Internal(fmt.Sprintf("no deployment found for hostname: %s", hostname)),
			fault.Public("Domain not configured"),
		)
	}

	// Determine target based on region
	var targetAddr string
	var targetScheme string
	var isLocal bool

	if deployment.Region == h.CurrentRegion {
		// Local gateway - use HTTP (TLS already terminated by ingress)
		targetAddr = deployment.K8SServiceName
		targetScheme = "http"
		isLocal = true

		h.Logger.Info("routing to local gateway",
			"hostname", hostname,
			"region", deployment.Region,
			"gateway", targetAddr,
		)
	} else {
		// Remote ingress - use HTTPS (will re-encrypt)
		// Find the closest region to route through
		closestRegions := h.DeploymentService.GetClosestRegions(h.CurrentRegion)

		// Determine which region to route to:
		// 1. If deployment region is in our closest regions list, route there directly
		// 2. Otherwise, route to the first region in our proximity list (closest hop)
		targetRegion := deployment.Region

		// Check if we should route through a closer intermediate region
		foundInProximity := false
		for _, region := range closestRegions {
			if region == deployment.Region {
				foundInProximity = true
				break
			}
		}

		// If deployment region is not in our proximity list or is far down the list,
		// route to the closest region as an intermediate hop
		if !foundInProximity && len(closestRegions) > 0 {
			targetRegion = closestRegions[0]
			h.Logger.Info("routing through intermediate region",
				"hostname", hostname,
				"currentRegion", h.CurrentRegion,
				"deploymentRegion", deployment.Region,
				"intermediateRegion", targetRegion,
			)
		}

		targetAddr = fmt.Sprintf("%s.%s", targetRegion, h.BaseDomain)
		targetScheme = "https"
		isLocal = false

		h.Logger.Info("routing to remote region",
			"hostname", hostname,
			"currentRegion", h.CurrentRegion,
			"targetRegion", targetRegion,
			"deploymentRegion", deployment.Region,
			"targetAddr", targetAddr,
		)
	}

	return h.forward(ctx, sess, targetScheme, targetAddr, deployment, isLocal)
}

// forward handles proxying to either local gateway or remote ingress
func (h *Handler) forward(ctx context.Context, sess *zen.Session, scheme, addr string, deployment *partitionv1.Deployment, isLocal bool) error {
	// Build target URL
	targetURL, err := url.Parse(fmt.Sprintf("%s://%s", scheme, addr))
	if err != nil {
		return fault.Wrap(err,
			fault.Code(codes.Gateway.Internal.InternalServerError.URN()),
			fault.Internal("failed to parse target URL"),
		)
	}

	// Create reverse proxy
	proxy := httputil.NewSingleHostReverseProxy(targetURL)

	// Customize director
	originalDirector := proxy.Director
	proxy.Director = func(req *http.Request) {
		originalDirector(req)

		if isLocal {
			// Local gateway - add ingress metadata headers
			req.Header.Set("X-Forwarded-Proto", "https")
			req.Header.Set("X-Ingress-Region", h.CurrentRegion)
			req.Header.Set("X-Deployment-Id", deployment.Id)
		} else {
			// Remote ingress - preserve original Host header for routing
			req.Host = sess.Request().Host
			req.Header.Set("Host", sess.Request().Host)

			// Add hop tracking to prevent infinite loops
			hopCount := req.Header.Get("X-Ingress-Hops")
			if hopCount == "" {
				req.Header.Set("X-Ingress-Hops", "1")
			} else {
				// TODO: parse and increment, fail if > 3
				req.Header.Set("X-Ingress-Hops", "2")
			}
		}
	}

	// Custom error handler
	proxy.ErrorHandler = func(w http.ResponseWriter, r *http.Request, err error) {
		h.Logger.Error("proxy error",
			"error", err,
			"target", targetURL.String(),
			"isLocal", isLocal,
		)
		// Write basic error response
		w.WriteHeader(http.StatusBadGateway)
		w.Write([]byte(`Service temporarily unavailable`))
	}

	h.Logger.Info("proxying request",
		"target", targetURL.String(),
		"path", sess.Request().URL.Path,
		"isLocal", isLocal,
	)

	// Proxy the request
	proxy.ServeHTTP(sess.ResponseWriter(), sess.Request())

	return nil
}
