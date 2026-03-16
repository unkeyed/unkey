package openapi

import (
	"database/sql"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	restate "github.com/restatedev/sdk-go"
	hydrav1 "github.com/unkeyed/unkey/gen/proto/hydra/v1"
	"github.com/unkeyed/unkey/pkg/db"
	"github.com/unkeyed/unkey/pkg/fault"
	"github.com/unkeyed/unkey/pkg/logger"
	"github.com/unkeyed/unkey/pkg/uid"
)

// ScrapeSpec fetches the OpenAPI spec from a running deployment and persists it
// in the database. It uses the deployment's runtime settings to locate the spec
// path, and a deployment-sticky frontline route to resolve the target host.
//
// Returns success (no error) when scraping is not configured
// (openapi_spec_path is null/empty), when no suitable route or instance exists,
// or when the endpoint returns 404 — not all deployments expose an OpenAPI spec.
func (s *Service) ScrapeSpec(ctx restate.Context, req *hydrav1.ScrapeSpecRequest) (*hydrav1.ScrapeSpecResponse, error) {
	deploymentID := req.GetDeploymentId()

	deployment, err := restate.Run(ctx, func(runCtx restate.RunContext) (db.Deployment, error) {
		return db.Query.FindDeploymentById(runCtx, s.db.RO(), deploymentID)
	}, restate.WithName("find deployment"))
	if err != nil {
		logger.Info("failed to query deployment, skipping openapi scrape",
			"deployment_id", deploymentID,
			"error", err,
		)
		return &hydrav1.ScrapeSpecResponse{}, nil
	}

	settings, err := restate.Run(ctx, func(runCtx restate.RunContext) (db.FindAppRuntimeSettingsByAppAndEnvRow, error) {
		return db.Query.FindAppRuntimeSettingsByAppAndEnv(runCtx, s.db.RO(), db.FindAppRuntimeSettingsByAppAndEnvParams{
			AppID:         deployment.AppID,
			EnvironmentID: deployment.EnvironmentID,
		})
	}, restate.WithName("find runtime settings"))
	if err != nil {
		logger.Info("failed to query runtime settings, skipping openapi scrape",
			"deployment_id", deploymentID,
			"error", err,
		)
		return &hydrav1.ScrapeSpecResponse{}, nil
	}

	if !settings.AppRuntimeSetting.OpenApiSpecPath.Valid || settings.AppRuntimeSetting.OpenApiSpecPath.String == "" {
		logger.Info("openapi_spec_path not configured, skipping scrape", "deployment_id", deploymentID)
		return &hydrav1.ScrapeSpecResponse{}, nil
	}
	specPath := settings.AppRuntimeSetting.OpenApiSpecPath.String

	routes, err := restate.Run(ctx, func(runCtx restate.RunContext) ([]db.FrontlineRoute, error) {
		return db.Query.FindFrontlineRoutesByDeploymentID(runCtx, s.db.RO(), deploymentID)
	}, restate.WithName("find frontline routes"))
	if err != nil {
		logger.Info("failed to query frontline routes, skipping openapi scrape",
			"deployment_id", deploymentID,
			"error", err,
		)
		return &hydrav1.ScrapeSpecResponse{}, nil
	}

	var fqdn string
	for _, route := range routes {
		if route.Sticky == db.FrontlineRoutesStickyDeployment {
			fqdn = route.FullyQualifiedDomainName
			break
		}
	}
	if fqdn == "" {
		logger.Info("no deployment-sticky route found, skipping openapi scrape", "deployment_id", deploymentID)
		return &hydrav1.ScrapeSpecResponse{}, nil
	}

	// Local dev: *.unkey.local FQDNs are not publicly routable and the mkcert CA
	// is not trusted inside the cluster, so reach the pod directly via plain HTTP.
	// Production: use HTTPS with the public FQDN.
	isLocal := strings.HasSuffix(fqdn, ".unkey.local")
	var specURL string
	if isLocal {
		instances, instErr := restate.Run(ctx, func(runCtx restate.RunContext) ([]db.Instance, error) {
			return db.Query.FindInstancesByDeploymentId(runCtx, s.db.RO(), deploymentID)
		}, restate.WithName("find instances"))
		if instErr != nil {
			logger.Info("failed to query instances, skipping openapi scrape",
				"deployment_id", deploymentID,
				"error", instErr,
			)
			return &hydrav1.ScrapeSpecResponse{}, nil
		}

		var instanceAddr string
		for _, inst := range instances {
			if inst.Status == db.InstancesStatusRunning {
				instanceAddr = inst.Address
				break
			}
		}
		if instanceAddr == "" {
			logger.Info("no running instance found, skipping openapi scrape", "deployment_id", deploymentID)
			return &hydrav1.ScrapeSpecResponse{}, nil
		}

		specURL = fmt.Sprintf("http://%s%s", instanceAddr, specPath)
	} else {
		specURL = fmt.Sprintf("https://%s%s", fqdn, specPath)
	}

	logger.Info("scraping openapi spec", "deployment_id", deploymentID, "fqdn", fqdn, "path", specPath, "url", specURL)

	specBody, err := restate.Run(ctx, func(_ restate.RunContext) ([]byte, error) {
		httpReq, reqErr := http.NewRequest(http.MethodGet, specURL, nil)
		if reqErr != nil {
			return nil, fmt.Errorf("creating request: %w", reqErr)
		}
		if isLocal {
			httpReq.Header.Set("X-Deployment-Id", deploymentID)
		}

		resp, doErr := s.httpClient.Do(httpReq)
		if doErr != nil {
			return nil, fmt.Errorf("fetching openapi spec: %w", doErr)
		}

		body, readErr := io.ReadAll(resp.Body)
		closeErr := resp.Body.Close()
		if readErr != nil {
			return nil, fmt.Errorf("reading openapi response body: %w", readErr)
		}
		if closeErr != nil {
			return nil, fmt.Errorf("closing openapi response body: %w", closeErr)
		}

		if resp.StatusCode == http.StatusNotFound {
			return nil, nil
		}
		if resp.StatusCode != http.StatusOK {
			return nil, fmt.Errorf("unexpected status %d from %q endpoint", resp.StatusCode, specURL)
		}

		return body, nil
	}, restate.WithName("fetch openapi spec"))
	if err != nil {
		logger.Info("failed to fetch openapi spec, skipping",
			"deployment_id", deploymentID,
			"error", err,
		)
		return &hydrav1.ScrapeSpecResponse{}, nil
	}

	if len(specBody) == 0 {
		logger.Info("no openapi spec found", "deployment_id", deploymentID, "path", specPath)
		return &hydrav1.ScrapeSpecResponse{}, nil
	}

	err = restate.RunVoid(ctx, func(runCtx restate.RunContext) error {
		return db.Query.UpsertOpenApiSpec(runCtx, s.db.RW(), db.UpsertOpenApiSpecParams{
			ID:           uid.New(uid.OpenApiSpecPrefix),
			WorkspaceID:  deployment.WorkspaceID,
			ProjectID:    sql.NullString{Valid: true, String: deployment.ProjectID},
			DeploymentID: sql.NullString{Valid: true, String: deploymentID},
			Spec:         specBody,
			CreatedAt:    time.Now().UnixMilli(),
			UpdatedAt:    sql.NullInt64{Valid: true, Int64: time.Now().UnixMilli()},
		})
	}, restate.WithName("persist openapi spec"))
	if err != nil {
		return nil, fault.Wrap(err, fault.Public("Failed to persist OpenAPI spec."))
	}

	logger.Info("openapi spec scraped and persisted", "deployment_id", deploymentID)
	return &hydrav1.ScrapeSpecResponse{}, nil
}
