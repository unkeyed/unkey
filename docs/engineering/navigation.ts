const navigation = {
  tabs: [
    {
      label: "Development",
      path: "/contributing",
      href: "/contributing/how-to-contribute",
    },
    { label: "Architecture", path: "/architecture" },
    { label: "Company", path: "/company" },
    { label: "Infra", path: "/infra" },
  ],
  sidebar: [
    {
      label: "Development",
      root: "/contributing",
      items: [
        "/contributing",
        {
          label: "Internal workflow",
          items: [
            "/contributing/how-to-contribute",
            "/contributing/local/development",
            {
              label: "Tooling",
              items: [
                "/contributing/tooling/mise",
                "/contributing/tooling/builds",
                "/contributing/tooling/releases",
                "/contributing/tooling/feature-flags",
                "/contributing/tooling/emails",
              ],
            },
          ],
        },
        {
          label: "Quality",
          items: [
            "/contributing/quality/code-quality",
            "/contributing/quality/documentation",
            "/contributing/quality/screenshots-and-recordings",
            {
              label: "Testing",
              items: [
                "/contributing/quality/testing",
                "/contributing/quality/testing/unit-tests",
                "/contributing/quality/testing/integration-tests",
                "/contributing/quality/testing/http-handler-tests",
                "/contributing/quality/testing/fuzz-tests",
                "/contributing/quality/testing/simulation-tests",
                "/contributing/quality/testing/anti-patterns",
              ],
            },
          ],
        },
      ],
    },
    {
      label: "Architecture",
      root: "/architecture",
      items: [
        {
          label: "Overview",
          items: [
            "/architecture",
            {
              label: "Resource model",
              items: [
                "/architecture/resources/unkey-resource-names",
                "/architecture/authorization/resource-permissions",
                "/architecture/authorization/workos-permissions",
              ],
            },
            {
              label: "Rate limiting",
              items: [
                "/architecture/ratelimiting/overview",
                "/architecture/ratelimiting/request-path",
                "/architecture/ratelimiting/consistency-model",
                "/architecture/ratelimiting/global-counters",
              ],
            },
          ],
        },
        {
          label: "Services",
          items: [
            {
              label: "API",
              items: [
                "/architecture/services/api/overview",
                "/architecture/services/api/configuration",
                {
                  label: "API design",
                  items: [
                    "/architecture/services/api/api-design",
                    "/architecture/services/api/api-design/auth",
                    "/architecture/services/api/api-design/errors",
                    "/architecture/services/api/api-design/rpc",
                  ],
                },
              ],
            },
            {
              label: "Control plane",
              items: [
                {
                  label: "API",
                  items: [
                    "/architecture/services/control-plane/api/overview",
                    "/architecture/services/control-plane/api/architecture",
                    "/architecture/services/control-plane/api/configuration",
                  ],
                },
                {
                  label: "Worker",
                  items: [
                    "/architecture/services/control-plane/worker/overview",
                    "/architecture/services/control-plane/worker/configuration",
                    "/architecture/services/control-plane/worker/deployment-sync",
                    {
                      label: "Workflows",
                      items: [
                        "/architecture/services/control-plane/worker/workflows/deployments",
                        "/architecture/services/control-plane/worker/workflows/routing",
                        "/architecture/services/control-plane/worker/workflows/certificates",
                        "/architecture/services/control-plane/worker/workflows/custom-domains",
                        "/architecture/services/control-plane/worker/workflows/github-app",
                        "/architecture/services/control-plane/worker/workflows/key-last-used-sync",
                        "/architecture/services/control-plane/worker/workflows/deploy-billing",
                        "/architecture/services/control-plane/worker/workflows/deploy-spend-cap",
                      ],
                    },
                  ],
                },
              ],
            },
            {
              label: "Frontline",
              items: [
                "/architecture/services/frontline/overview",
                "/architecture/services/frontline/ingress",
                "/architecture/services/frontline/routing",
                "/architecture/services/frontline/request-flow",
                "/architecture/services/frontline/configuration",
                "/architecture/services/frontline/failure-modes",
                {
                  label: "Policies",
                  items: [
                    "/architecture/services/frontline/policies",
                    "/architecture/services/frontline/policies/policy",
                    "/architecture/services/frontline/policies/match-expressions",
                    "/architecture/services/frontline/policies/principal",
                    "/architecture/services/frontline/policies/keyauth",
                    "/architecture/services/frontline/policies/jwtauth",
                    "/architecture/services/frontline/policies/ratelimit",
                    "/architecture/services/frontline/policies/firewall",
                    "/architecture/services/frontline/policies/openapi",
                  ],
                },
              ],
            },
            {
              label: "Krane",
              items: [
                "/architecture/services/krane/overview",
                "/architecture/services/krane/deployment",
                "/architecture/services/krane/secrets",
                "/architecture/services/krane/configuration",
              ],
            },
            {
              label: "Vault",
              items: [
                "/architecture/services/vault/overview",
                "/architecture/services/vault/auth",
                "/architecture/services/vault/configuration",
              ],
            },
          ],
        },
        {
          label: "RFCs",
          items: [
            "/architecture/rfcs",
            "/architecture/rfcs/template",
            "/architecture/rfcs/rbac",
            "/architecture/rfcs/github-secret-scanning",
            "/architecture/rfcs/key-shape",
            "/architecture/rfcs/coss-starter",
            "/architecture/rfcs/analytics-api",
            "/architecture/rfcs/auth-migration",
            "/architecture/rfcs/client-file-structure",
            "/architecture/rfcs/dataplane",
            "/architecture/rfcs/pricing-updates",
            "/architecture/rfcs/split-monos",
            "/architecture/rfcs/unkey-resource-names",
            "/architecture/rfcs/stricter-linter",
            "/architecture/rfcs/custom-domains",
            "/architecture/rfcs/frontline-middleware",
            "/architecture/rfcs/ratelimit-cross-region-counts",
            "/architecture/rfcs/vault-s3-storage",
          ],
        },
      ],
    },
    {
      label: "Company",
      root: "/company",
      items: ["/company", "/company/meetings"],
    },
    {
      label: "Infra",
      root: "/infra",
      items: [
        {
          label: "Infra",
          items: [
            "/infra",
            "/infra/planetscale/query-insights-tags",
            "/infra/planetscale/schema-changes",
          ],
        },
        {
          label: "Runbooks",
          items: [
            "/infra/runbooks/key-migration",
            "/infra/runbooks/legacy-billing",
          ],
        },
        {
          label: "ClickHouse",
          items: [
            "/infra/clickhouse",
            {
              label: "Roles",
              items: [
                "/infra/clickhouse/roles/grafana-readonly",
                "/infra/clickhouse/roles/insertonly-role",
                "/infra/clickhouse/roles/readonly-role",
              ],
            },
            {
              label: "Users",
              items: [
                "/infra/clickhouse/users/apiv2",
                "/infra/clickhouse/users/eve",
                "/infra/clickhouse/users/github",
                "/infra/clickhouse/users/grafana",
                "/infra/clickhouse/users/frontline",
                "/infra/clickhouse/users/unkey-admin",
                "/infra/clickhouse/users/vector",
                "/infra/clickhouse/users/vercel-dashboard",
              ],
            },
          ],
        },
      ],
    },
  ],
};

export default navigation;
