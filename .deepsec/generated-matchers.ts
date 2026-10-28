import { compileDeclarativeMatchers, type DeepsecPlugin } from "deepsec/config";

const specs = [
	{
		version: 1,
		slug: "unkey-zen-route-registration",
		description:
			"Detects Unkey public API route registration, Zen handlers, and authentication middleware.",
		noiseTier: "precise",
		filePatterns: [
			"svc/api/run.go",
			"svc/api/routes/register.go",
			"svc/api/routes/*/handler.go",
			"svc/api/internal/middleware/authentication.go",
		],
		patterns: [
			{
				source: "^\\s*routes\\.Register\\(srv,\\s*&routes\\.Services\\{",
				flags: "m",
				label: "API route collection registration",
			},
			{
				source: "^\\s*srv\\.RegisterRoute\\(",
				flags: "m",
				label: "Zen route registration",
			},
			{
				source:
					"^func \\(h \\*Handler\\) Handle\\(ctx context\\.Context, (?:s|sess) \\*zen\\.Session\\) error \\{",
				flags: "m",
				label: "Zen HTTP handler",
			},
			{
				source:
					"^func WithAuthentication\\(config AuthenticationConfig\\) zen\\.Middleware \\{",
				flags: "m",
				label: "API authentication middleware",
			},
		],
		examples: [
			"routes.Register(srv, &routes.Services{",
			"srv.RegisterRoute(",
			"func (h *Handler) Handle(ctx context.Context, s *zen.Session) error {",
			"func WithAuthentication(config AuthenticationConfig) zen.Middleware {",
		],
		closesSurfaceIds: ["public-api-http"],
	},
	{
		version: 1,
		slug: "frontline-zen-policy-surface",
		description:
			"Detects Frontline Zen gateway routes, policy executors, and reserved-header middleware.",
		noiseTier: "precise",
		filePatterns: [
			"svc/frontline/run.go",
			"svc/frontline/routes/register.go",
			"svc/frontline/routes/*/handler.go",
			"svc/frontline/internal/policies/engine.go",
			"svc/frontline/internal/policies/*/executor.go",
			"svc/frontline/middleware/sanitize_headers.go",
		],
		patterns: [
			{
				source: "^\\s*routes\\.Register(?:HTTPServer)?\\(",
				flags: "m",
				label: "Frontline route collection registration",
			},
			{
				source: "^\\s*srv\\.RegisterRoute\\(",
				flags: "m",
				label: "Frontline Zen route registration",
			},
			{
				source:
					"^func \\(h \\*Handler\\) Handle\\(ctx context\\.Context, (?:s|sess) \\*zen\\.Session\\) error \\{",
				flags: "m",
				label: "Frontline Zen handler",
			},
			{
				source: "^func \\(e \\*Engine\\) Evaluate\\(",
				flags: "m",
				label: "Gateway policy engine",
			},
			{
				source: "^func \\(e \\*Executor\\) Execute\\(",
				flags: "m",
				label: "Gateway policy executor",
			},
			{
				source: "^func WithReservedHeaderStrip\\(\\) zen\\.Middleware \\{",
				flags: "m",
				label: "Reserved-header boundary middleware",
			},
		],
		examples: [
			"routes.RegisterHTTPServer(httpSrv, svcs)",
			"srv.RegisterRoute(",
			"func (h *Handler) Handle(ctx context.Context, sess *zen.Session) error {",
			"func (e *Engine) Evaluate(",
			"func (e *Executor) Execute(",
			"func WithReservedHeaderStrip() zen.Middleware {",
		],
		closesSurfaceIds: ["frontline-gateway-http"],
	},
	{
		version: 1,
		slug: "dashboard-trpc-procedure-registration",
		description:
			"Detects the dashboard tRPC fetch adapter, context factory, routers, and authenticated procedure declarations.",
		noiseTier: "precise",
		filePatterns: [
			"web/apps/dashboard/app/**/api/trpc/**/route.ts",
			"web/apps/dashboard/lib/trpc/context.ts",
			"web/apps/dashboard/lib/trpc/trpc.ts",
			"web/apps/dashboard/lib/trpc/routers/**/*.ts",
		],
		patterns: [
			{
				source: "^\\s*return fetchRequestHandler\\(\\{",
				flags: "m",
				label: "tRPC HTTP adapter",
			},
			{
				source:
					"^export async function createContext\\(\\{ req \\}: FetchCreateContextFnOptions\\) \\{",
				flags: "m",
				label: "tRPC request context",
			},
			{
				source:
					"^export const [A-Za-z][A-Za-z0-9_]* = (?:publicProcedure|protectedProcedure|workspaceProcedure)\\b",
				flags: "m",
				label: "tRPC procedure declaration",
			},
			{
				source: "^export const router = t\\.router\\(\\{",
				flags: "m",
				label: "tRPC root router",
			},
		],
		excludeFilePatterns: ["**/*.test.ts", "**/*.test.tsx"],
		examples: [
			"return fetchRequestHandler({",
			"export async function createContext({ req }: FetchCreateContextFnOptions) {",
			"export const getOrg = workspaceProcedure.input(",
			"export const router = t.router({",
		],
		closesSurfaceIds: ["dashboard-trpc-rpc"],
	},
	{
		version: 1,
		slug: "dashboard-server-action-directive",
		description:
			"Detects dashboard modules registered as callable Next.js Server Actions.",
		noiseTier: "precise",
		filePatterns: [
			"web/apps/dashboard/app/actions.ts",
			"web/apps/dashboard/app/**/actions.ts",
		],
		patterns: [
			{
				source: "^\\s*[\"']use server[\"'];?\\s*$",
				flags: "m",
				label: "Next.js Server Action module",
			},
		],
		examples: ['"use server";'],
		closesSurfaceIds: ["dashboard-next-http"],
	},
	{
		version: 1,
		slug: "portal-tanstack-server-surface",
		description:
			"Detects portal TanStack server functions and file-route registrations.",
		noiseTier: "precise",
		filePatterns: [
			"web/apps/portal/src/lib/session.ts",
			"web/apps/portal/src/lib/portal-api.ts",
			"web/apps/portal/src/routes/**/*.tsx",
		],
		patterns: [
			{
				source:
					'^(?:export )?const [A-Za-z][A-Za-z0-9_]* = createServerFn\\(\\{ method: "(?:GET|POST)" \\}\\)',
				flags: "m",
				label: "TanStack server function",
			},
			{
				source: '^export const Route = createFileRoute\\("[^"]+"\\)\\(\\{',
				flags: "m",
				label: "TanStack file route",
			},
			{
				source:
					"^export const Route = createRootRouteWithContext<[^>]+>\\(\\)\\(\\{",
				flags: "m",
				label: "TanStack root route",
			},
		],
		examples: [
			'export const exchangeCode = createServerFn({ method: "POST" })',
			'const deleteSessionCookie = createServerFn({ method: "POST" })',
			'export const Route = createFileRoute("/_portal")({',
			"export const Route = createRootRouteWithContext<{ queryClient: QueryClient }>()({",
		],
		closesSurfaceIds: ["portal-server-functions"],
	},
	{
		version: 1,
		slug: "unkey-provider-webhook-receiver",
		description:
			"Detects Unkey webhook receiver contracts, provider verification handlers, event registration, and Next.js webhook endpoints.",
		noiseTier: "precise",
		filePatterns: [
			"pkg/webhook/**/*.go",
			"svc/ctrl/api/webhooks/**/*.go",
			"web/apps/dashboard/app/api/webhooks/*/route.ts",
		],
		patterns: [
			{
				source:
					"^type Verifier interface \\{\\s*Verify\\(r \\*http\\.Request\\) \\(Event, error\\)",
				flags: "m",
				label: "Webhook verifier contract",
			},
			{
				source:
					"^func New\\(provider string, verifier Verifier, opts \\.\\.\\.Option\\) \\*Receiver \\{",
				flags: "m",
				label: "Webhook receiver constructor",
			},
			{
				source: "^func \\(rec \\*Receiver\\) (?:OnRaw|Default|Use)\\(",
				flags: "m",
				label: "Webhook event registration",
			},
			{
				source: "^func \\(rec \\*Receiver\\) On(?:Response)?\\[",
				flags: "m",
				label: "Typed webhook event registration",
			},
			{
				source:
					"^func \\(v \\*Verifier\\) Verify\\(r \\*http\\.Request\\) \\(webhook\\.Event, error\\) \\{",
				flags: "m",
				label: "Provider signature verifier",
			},
			{
				source: '^\\s*return webhook\\.New\\("[a-z_]+",',
				flags: "m",
				label: "Provider webhook endpoint construction",
			},
			{
				source: "^export (?:async function POST\\(|const POST = async \\()",
				flags: "m",
				label: "Next.js webhook POST endpoint",
			},
		],
		excludeFilePatterns: ["**/*_test.go", "**/*.test.ts"],
		examples: [
			"type Verifier interface {\n\tVerify(r *http.Request) (Event, error)",
			"func New(provider string, verifier Verifier, opts ...Option) *Receiver {",
			"func (rec *Receiver) OnRaw(eventTypes []string, fn HandlerFunc) *Receiver {",
			"func (rec *Receiver) On[T any](",
			"func (v *Verifier) Verify(r *http.Request) (webhook.Event, error) {",
			'return webhook.New("github", githubverifier.New(webhookSecret)).',
			"export const POST = async (",
			"export async function POST(",
		],
		closesSurfaceIds: ["provider-webhooks"],
	},
	{
		version: 1,
		slug: "unkey-connect-service-binding",
		description:
			"Detects control-plane Connect RPC handler bindings and generated service-handler implementations.",
		noiseTier: "precise",
		filePatterns: ["svc/ctrl/api/run.go", "svc/ctrl/services/**/*.go"],
		patterns: [
			{
				source:
					"^\\s*mux\\.Handle\\(ctrlv1connect\\.New[A-Za-z0-9]+ServiceHandler\\(",
				flags: "m",
				label: "Connect service handler registration",
			},
			{
				source:
					"^\\s*ctrlv1connect\\.Unimplemented[A-Za-z0-9]+ServiceHandler\\s*$",
				flags: "m",
				label: "Connect service implementation",
			},
		],
		excludeFilePatterns: ["**/*_test.go"],
		examples: [
			"mux.Handle(ctrlv1connect.NewCtrlServiceHandler(",
			"ctrlv1connect.UnimplementedClusterServiceHandler",
		],
		closesSurfaceIds: ["control-plane-connect-rpc"],
	},
	{
		version: 1,
		slug: "unkey-restate-handler-binding",
		description:
			"Detects Restate service bindings, generated service implementations, and handlers through their Restate context types.",
		noiseTier: "precise",
		filePatterns: ["svc/ctrl/worker/run.go", "svc/ctrl/worker/**/*.go"],
		patterns: [
			{
				source:
					"^\\s*restateSrv\\.Bind\\((?:hydrav1\\.New[A-Za-z0-9]+(?:Service|Workflow)Server|restate\\.NewObject)\\(",
				flags: "m",
				label: "Restate service binding",
			},
			{
				source:
					"^\\s*hydrav1\\.Unimplemented[A-Za-z0-9]+(?:Service|Workflow)Server\\s*$",
				flags: "m",
				label: "Generated Restate service implementation",
			},
			{
				source:
					"^func \\([^)]*\\) [A-Z][A-Za-z0-9_]*\\(\\s*ctx restate\\.(?:Context|ObjectContext|WorkflowContext|WorkflowSharedContext),",
				flags: "m",
				label: "Restate service handler",
			},
			{
				source:
					"^\\s*return hydrav1\\.New[A-Za-z0-9]+ServiceServer\\(s, opts\\.\\.\\.\\)",
				flags: "m",
				label: "Configured Restate service definition",
			},
		],
		excludeFilePatterns: ["svc/ctrl/worker/cron/**", "**/*_test.go"],
		examples: [
			"restateSrv.Bind(hydrav1.NewDeployWorkflowServer(",
			"hydrav1.UnimplementedGitHubWebhookServiceServer",
			"func (s *Service) HandlePush(ctx restate.ObjectContext,",
			"return hydrav1.NewKeyLeakServiceServer(s, opts...)",
		],
		closesSurfaceIds: [
			"control-worker-restate-rpc",
			"durable-restate-event-consumers",
		],
	},
	{
		version: 1,
		slug: "unkey-restate-cron-binding",
		description:
			"Detects Restate cron handlers, handler retry bindings, Kubernetes CronJobs, invocation paths, and the CronService RPC declaration.",
		noiseTier: "precise",
		filePatterns: [
			"svc/ctrl/worker/run.go",
			"svc/ctrl/worker/cron/cron.go",
			"svc/ctrl/worker/cron/*/handler.go",
			"svc/ctrl/proto/hydra/v1/cron.proto",
			"dev/k8s/charts/restate-cronjobs/values.yaml",
			"dev/k8s/charts/restate-cronjobs/templates/cronjobs.yaml",
		],
		patterns: [
			{
				source:
					"^func \\(h \\*Handler\\) Handle\\(\\s*ctx restate\\.ObjectContext,",
				flags: "m",
				label: "Restate cron task handler",
			},
			{
				source:
					"^func \\(s \\*Service\\) (?:Run|Close|Enforce|Throttle)[A-Za-z0-9_]*\\(\\s*ctx restate\\.ObjectContext,",
				flags: "m",
				label: "Restate CronService handler",
			},
			{
				source:
					'^\\s*ConfigureHandler\\("(?:Run|Close|Enforce|Throttle)[A-Za-z0-9_]+"',
				flags: "m",
				label: "Cron retry-policy binding",
			},
			{
				source: "^kind:\\s+CronJob\\s*$",
				flags: "m",
				label: "Kubernetes CronJob",
			},
			{
				source: '^\\s*urlPath:\\s+"hydra\\.v1\\.CronService/',
				flags: "m",
				label: "Restate cron invocation path",
			},
			{
				source: "^service CronService \\{",
				flags: "m",
				label: "Restate cron RPC service",
			},
		],
		excludeFilePatterns: ["**/*_test.go"],
		examples: [
			"func (h *Handler) Handle(\n\tctx restate.ObjectContext,",
			"func (s *Service) RunQuotaCheck(\n\tctx restate.ObjectContext,",
			'ConfigureHandler("RunQuotaCheck", cronQuotaCheckRetry)',
			"kind: CronJob",
			'urlPath: "hydra.v1.CronService/$(date -u +%Y-%m-%d)/RunKeyRefill/send"',
			"service CronService {",
		],
		closesSurfaceIds: ["restate-cron-jobs"],
	},
	{
		version: 1,
		slug: "unkey-cli-command-registration",
		description:
			"Detects Unkey CLI command declarations, the root command, command runtime, and authenticated SDK client construction.",
		noiseTier: "precise",
		filePatterns: [
			"build/cli/*.go",
			"cmd/api/**/*.go",
			"cmd/auth/*.go",
			"cmd/deploy/**/*.go",
			"cmd/healthcheck/*.go",
			"pkg/cli/*.go",
		],
		patterns: [
			{
				source: "^func [A-Za-z][A-Za-z0-9_]*\\([^)]*\\) \\*cli\\.Command \\{",
				flags: "m",
				label: "CLI command factory",
			},
			{
				source: "^var (?:Cmd|[A-Za-z][A-Za-z0-9_]*Cmd) = &cli\\.Command\\{",
				flags: "m",
				label: "CLI command declaration",
			},
			{
				source: "^\\s*app := &cli\\.Command\\{",
				flags: "m",
				label: "Root CLI command",
			},
			{
				source: "^\\s*unkey\\.WithSecurity\\(",
				flags: "m",
				label: "Authenticated CLI SDK client",
			},
			{
				source: "^type Command struct \\{",
				flags: "m",
				label: "CLI command framework",
			},
			{
				source: "^func \\(c \\*Command\\) Run\\(",
				flags: "m",
				label: "CLI command runtime",
			},
		],
		excludeFilePatterns: ["**/*_test.go"],
		examples: [
			"func Cmd() *cli.Command {",
			"var Cmd = &cli.Command{",
			"app := &cli.Command{",
			"unkey.WithSecurity(key),",
			"type Command struct {",
			"func (c *Command) Run(",
		],
		closesSurfaceIds: ["unkey-cli"],
	},
	{
		version: 1,
		slug: "unkey-restate-cron-handler-implementation",
		description:
			"Identifies Unkey Restate cron task implementations and their generated CronService RPC adapters.",
		noiseTier: "precise",
		filePatterns: [
			"svc/ctrl/worker/cron/cron.go",
			"svc/ctrl/worker/cron/*/handler.go",
		],
		requires: {
			tech: ["go"],
			sentinelFiles: ["svc/ctrl/proto/hydra/v1/cron.proto"],
		},
		patterns: [
			{
				source:
					"^func \\(h \\*Handler\\) Handle\\(\\s+ctx restate\\.ObjectContext,\\s+_ \\*hydrav1\\.Run[A-Za-z0-9]+Request,\\s+\\) \\(\\*hydrav1\\.Run[A-Za-z0-9]+Response, error\\) \\{",
				flags: "m",
				label: "Restate cron task implementation",
			},
			{
				source:
					"^func \\(s \\*Service\\) (?:Run|Close|Enforce|Throttle)[A-Za-z0-9]+\\(\\s+ctx restate\\.ObjectContext,\\s+req \\*hydrav1\\.[A-Za-z0-9]+Request,\\s+\\) \\(\\*hydrav1\\.[A-Za-z0-9]+Response, error\\) \\{",
				flags: "m",
				label: "CronService RPC adapter",
			},
		],
		examples: [
			"func (h *Handler) Handle(\n\tctx restate.ObjectContext,\n\t_ *hydrav1.RunKeyRefillRequest,\n) (*hydrav1.RunKeyRefillResponse, error) {",
			"func (s *Service) RunQuotaCheck(\n\tctx restate.ObjectContext,\n\treq *hydrav1.RunQuotaCheckRequest,\n) (*hydrav1.RunQuotaCheckResponse, error) {",
		],
		closesSurfaceIds: ["restate-cron-jobs"],
	},
	{
		version: 1,
		slug: "unkey-cli-dispatch-runtime",
		description:
			"Identifies the custom Unkey CLI framework entrypoint and recursive command dispatcher.",
		noiseTier: "precise",
		filePatterns: ["pkg/cli/command.go", "pkg/cli/parser.go"],
		requires: {
			tech: ["go"],
			sentinelFiles: ["pkg/cli/command.go"],
		},
		patterns: [
			{
				source:
					"^func \\(c \\*Command\\) (?:Run|parse)\\(ctx context\\.Context, args \\[\\]string\\) error \\{",
				flags: "m",
				label: "Unkey CLI command dispatcher",
			},
		],
		examples: [
			"func (c *Command) Run(ctx context.Context, args []string) error {",
			"func (c *Command) parse(ctx context.Context, args []string) error {",
		],
		closesSurfaceIds: ["unkey-cli"],
	},
	{
		version: 1,
		slug: "unkey-cron-object-context-entrypoint",
		description:
			"Detects Unkey cron entrypoints through the Restate ObjectContext parameter used by CronService adapters and task handlers.",
		noiseTier: "precise",
		filePatterns: [
			"svc/ctrl/worker/cron/cron.go",
			"svc/ctrl/worker/cron/*/handler.go",
		],
		requires: {
			tech: ["go"],
			sentinelFiles: ["svc/ctrl/proto/hydra/v1/cron.proto"],
		},
		patterns: [
			{
				source: "^\\s*ctx\\s+restate\\.ObjectContext,\\s*$",
				flags: "m",
				label: "Restate cron object handler context",
			},
		],
		excludeFilePatterns: ["**/*_test.go"],
		examples: ["\tctx restate.ObjectContext,"],
		closesSurfaceIds: ["restate-cron-jobs"],
	},
];

export const generatedMatchersPlugin: DeepsecPlugin = {
	name: "deepsec-generated-matchers",
	matchers: compileDeclarativeMatchers(specs),
};
