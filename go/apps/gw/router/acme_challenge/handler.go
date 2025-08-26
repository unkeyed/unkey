package acme_challenge

import (
	"context"
	"net"
	"net/http"
	"path"

	"connectrpc.com/connect"
	"github.com/unkeyed/unkey/go/apps/gw/server"
	"github.com/unkeyed/unkey/go/apps/gw/services/routing"
	ctrlv1 "github.com/unkeyed/unkey/go/gen/proto/ctrl/v1"
	"github.com/unkeyed/unkey/go/gen/proto/ctrl/v1/ctrlv1connect"
	"github.com/unkeyed/unkey/go/pkg/codes"
	"github.com/unkeyed/unkey/go/pkg/fault"
	"github.com/unkeyed/unkey/go/pkg/otel/logging"
)

// Handler implements the main gateway proxy functionality.
// It handles all incoming requests and forwards them to backend services.
type Handler struct {
	Logger         logging.Logger
	RoutingService routing.Service
	AcmeClient     ctrlv1connect.AcmeServiceClient
}

// Handle sends all ACME Let's Encrypt challenges to the control plane.
func (h *Handler) Handle(ctx context.Context, s *server.Session) error {
	req := s.Request()

	// Look up target configuration based on the request host
	// Strip port from hostname for database lookup (Host header may include port)
	hostname, _, err := net.SplitHostPort(req.Host)
	if err != nil {
		// If SplitHostPort fails, req.Host doesn't contain a port, use it as-is
		hostname = req.Host
	}

	_, err = h.RoutingService.GetConfig(ctx, hostname)
	if err != nil {
		return fault.Wrap(err,
			fault.Code(codes.Gateway.Routing.ConfigNotFound.URN()),
			fault.Internal("failed to lookup target configuration"),
			fault.Public("Service configuration not found"),
		)
	}

	// Extract ACME token from path (last segment after /.well-known/acme-challenge/)
	token := path.Base(req.URL.Path)

	h.Logger.Info("Handling ACME challenge", "hostname", hostname, "token", token)

	createReq := connect.NewRequest(&ctrlv1.HandleCertificateVerificationRequest{
		Domain: hostname,
		Token:  token,
	})
	// createReq.Header().Set("Authorization", "Bearer "+c.opts.AuthToken)

	resp, err := h.AcmeClient.HandleCertificateVerification(ctx, createReq)
	if err != nil {
		h.Logger.Error("Failed to handle certificate verification", "error", err)

		return fault.Wrap(err,
			fault.Code(codes.App.Internal.UnexpectedError.URN()),
			fault.Internal("failed to handle ACME challenge"),
			fault.Public("Failed to handle ACME challenge"),
		)
	}

	h.Logger.Info("Certificate verification handled", "response", resp.Msg.GetToken())
	s.Plain(http.StatusOK, []byte(resp.Msg.GetToken()))

	return nil
}
