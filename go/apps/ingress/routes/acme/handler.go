package handler

import (
	"context"
	"net/http"
	"path"

	"connectrpc.com/connect"
	"github.com/unkeyed/unkey/go/apps/ingress/services/deployments"
	"github.com/unkeyed/unkey/go/apps/ingress/services/proxy"
	ctrlv1 "github.com/unkeyed/unkey/go/gen/proto/ctrl/v1"
	"github.com/unkeyed/unkey/go/gen/proto/ctrl/v1/ctrlv1connect"
	"github.com/unkeyed/unkey/go/pkg/codes"
	"github.com/unkeyed/unkey/go/pkg/fault"
	"github.com/unkeyed/unkey/go/pkg/otel/logging"
	"github.com/unkeyed/unkey/go/pkg/zen"
)

type Handler struct {
	Logger            logging.Logger
	AcmeClient        ctrlv1connect.AcmeServiceClient
	DeploymentService deployments.Service
}

func (h *Handler) Method() string {
	return zen.CATCHALL
}

func (h *Handler) Path() string {
	return "/.well-known/acme-challenge/{token...}"
}

// Handle processes ACME HTTP-01 challenges for Let's Encrypt certificate issuance
// TODO: Implement ACME challenge handler
func (h *Handler) Handle(ctx context.Context, sess *zen.Session) error {
	req := sess.Request()
	// Look up target configuration based on the request host
	hostname := proxy.ExtractHostname(proxy.ExtractHostname(req.Host))

	_, found, err := h.DeploymentService.LookupByHostname(ctx, hostname)
	if err != nil {
		return err
	}

	if !found {
		// TODO: Correct error code.
		return fault.New("Service configuration not found", fault.Code(codes.Ingress.Routing.ConfigNotFound.URN()))
	}

	// Extract ACME token from path (last segment after /.well-known/acme-challenge/)
	token := path.Base(req.URL.Path)
	h.Logger.Info("Handling ACME challenge", "hostname", hostname, "token", token)
	createReq := connect.NewRequest(&ctrlv1.VerifyCertificateRequest{
		Domain: hostname,
		Token:  token,
	})

	resp, err := h.AcmeClient.VerifyCertificate(ctx, createReq)
	if err != nil {
		h.Logger.Error("Failed to handle certificate verification", "error", err)
		return fault.Wrap(err,
			fault.Code(codes.App.Internal.UnexpectedError.URN()),
			fault.Internal("failed to handle ACME challenge"),
			fault.Public("Failed to handle ACME challenge"),
		)
	}

	auth := resp.Msg.GetAuthorization()
	h.Logger.Info("Certificate verification handled", "response", auth)
	return sess.Plain(http.StatusOK, []byte(auth))
}
