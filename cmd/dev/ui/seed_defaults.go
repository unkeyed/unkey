package ui

import (
	"context"
	"os"
	"strings"

	devseed "github.com/unkeyed/unkey/internal/devtools/seed"
)

const defaultLocalDSN = "unkey:password@tcp(127.0.0.1:3306)/unkey?parseTime=true&interpolateParams=true"

func envOr(key, fallback string) string {
	if v := strings.TrimSpace(os.Getenv(key)); v != "" {
		return v
	}
	return fallback
}

func envBool(key string) bool {
	v := strings.ToLower(strings.TrimSpace(os.Getenv(key)))
	return v == "1" || v == "true" || v == "yes"
}

func databaseDSN() string {
	return envOr("UNKEY_DATABASE_PRIMARY", defaultLocalDSN)
}

func localSeedParams(slug string) devseed.LocalParams {
	if slug == "" {
		slug = envOr("UNKEY_SEED_SLUG", "local")
	}
	return devseed.LocalParams{
		Slug:            slug,
		OrgID:           envOr("UNKEY_ORG_ID", "org_localdefault"),
		DatabasePrimary: databaseDSN(),
		Output:          envOr("UNKEY_SEED_OUTPUT", "dev/.env.seed"),
		Portal:          envBool("UNKEY_SEED_PORTAL"),
		CtrlURL:         envOr("UNKEY_CTRL_URL", "http://localhost:7091"),
		APIKey:          envOr("UNKEY_API_KEY", "your-local-dev-key"),
	}
}

func runLocalSeed(ctx context.Context, slug string) (devseed.LocalResult, error) {
	return devseed.SeedLocal(ctx, localSeedParams(slug))
}
