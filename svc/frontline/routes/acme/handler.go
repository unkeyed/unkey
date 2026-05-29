package handler

import (
	"context"
	"net/http"
	"path"

	ctrlv1 "github.com/unkeyed/unkey/gen/proto/ctrl/v1"
	"github.com/unkeyed/unkey/gen/rpc/ctrl"
	"github.com/unkeyed/unkey/pkg/codes"
	"github.com/unkeyed/unkey/pkg/fault"
	"github.com/unkeyed/unkey/pkg/logger"
	"github.com/unkeyed/unkey/pkg/mysql"
	"github.com/unkeyed/unkey/pkg/zen"
	"github.com/unkeyed/unkey/svc/frontline/internal/proxy"
)

// domainLookup is the minimal slice of the frontline db.Querier the ACME
// handler needs. ACME HTTP-01 only validates ownership of a custom domain,
// not routing configuration, so the handler bypasses the router entirely and
// consults custom_domains directly.
type domainLookup interface {
	FindCustomDomainIDByDomain(ctx context.Context, domain string) (string, error)
}

type Handler struct {
	AcmeClient ctrl.AcmeServiceClient
	DB         domainLookup
}

func (h *Handler) Method() string {
	return zen.CATCHALL
}

func (h *Handler) Path() string {
	return "/.well-known/acme-challenge/{token...}"
}

// Handle processes ACME HTTP-01 challenges for Let's Encrypt certificate issuance
func (h *Handler) Handle(ctx context.Context, sess *zen.Session) error {
	req := sess.Request()
	// Look up target configuration based on the request host
	hostname := proxy.ExtractHostname(req.Host)

	_, err := h.DB.FindCustomDomainIDByDomain(ctx, hostname)
	if err != nil {
		if mysql.IsNotFound(err) {
			return fault.New("no custom domain registered for hostname: "+hostname,
				fault.Code(codes.Frontline.Routing.ConfigNotFound.URN()),
				fault.Public("Domain not configured"),
			)
		}
		return fault.Wrap(err,
			fault.Code(codes.Frontline.Internal.ConfigLoadFailed.URN()),
			fault.Internal("error loading custom domain"),
			fault.Public("Failed to load route configuration"),
		)
	}

	// Extract ACME token from path (last segment after /.well-known/acme-challenge/)
	token := path.Base(req.URL.Path)
	logger.Info("Handling ACME challenge", "hostname", hostname, "token", token)
	resp, err := h.AcmeClient.VerifyCertificate(ctx, &ctrlv1.VerifyCertificateRequest{
		Domain: hostname,
		Token:  token,
	})
	if err != nil {
		logger.Error("Failed to handle certificate verification", "error", err)
		return fault.Wrap(err,
			fault.Code(codes.App.Internal.UnexpectedError.URN()),
			fault.Internal("failed to handle ACME challenge"),
			fault.Public("Failed to handle ACME challenge"),
		)
	}

	auth := resp.GetAuthorization()
	logger.Info("Certificate verification handled", "response", auth)
	return sess.Plain(http.StatusOK, []byte(auth))
}
