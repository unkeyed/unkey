import { ratelimit, requireUser, requireWorkspace, t, withRatelimit } from "@/lib/trpc/trpc";
import { z } from "zod";

const LogLevel = z.enum(["info", "warning", "error"]);

const LogEntry = z.object({
  id: z.string(),
  timestamp: z.number(),
  level: LogLevel.optional(),
  message: z.string(),
});

const deploymentLogsInputSchema = z.object({
  deploymentId: z.string(),
});

const deploymentLogsOutputSchema = z.object({
  logs: z.array(LogEntry),
});

const MOCK_LOGS: DeploymentLog[] = [
  {
    id: "log_1",
    timestamp: Date.now() - 360000,
    message: "Running build in us-east-1 (Washington, D.C.) — iad1",
  },
  {
    id: "log_2",
    timestamp: Date.now() - 359000,
    message: "Cloning github.com/acme/api (Branch: main, Commit: e5f6a7b)",
    level: "error",
  },
  {
    id: "log_3",
    timestamp: Date.now() - 358000,
    message: "Build cache not found for this project",
  },
  {
    id: "log_4",
    timestamp: Date.now() - 357000,
    message: "Clone completed in 307ms",
    level: "warning",
  },
  {
    id: "log_5",
    timestamp: Date.now() - 356000,
    message: "Running `unkey build`",
  },
  {
    id: "log_6",
    timestamp: Date.now() - 355000,
    message: "Unkey CLI 0.42.1",
  },
  {
    id: "log_7",
    timestamp: Date.now() - 354000,
    message: "Validating config files...",
  },
  {
    id: "log_8",
    timestamp: Date.now() - 353000,
    message: "✓ env-vars.json validated",
  },
  {
    id: "log_9",
    timestamp: Date.now() - 352000,
    message: "✓ runtime.json validated",
  },
  {
    id: "log_10",
    timestamp: Date.now() - 351000,
    message: "✓ secrets.json decrypted successfully",
  },
  {
    id: "log_11",
    timestamp: Date.now() - 350000,
    message: "✓ openapi.yaml parsed — 13 endpoints detected",
  },
  {
    id: "log_12",
    timestamp: Date.now() - 349000,
    message: '⚠️  Warning: Environment variable "STRIPE_SECRET" is not set. Using fallback value',
    level: "warning",
  },
  {
    id: "log_13",
    timestamp: Date.now() - 348000,
    message: "Setting up runtime environment",
  },
  {
    id: "log_14",
    timestamp: Date.now() - 347000,
    message: "Target image: unkey:latest",
  },
  {
    id: "log_15",
    timestamp: Date.now() - 346000,
    message: "Build environment: nodejs18.x | Linux (x64)",
  },
  {
    id: "log_16",
    timestamp: Date.now() - 345000,
    message: "Installing dependencies...",
  },
  {
    id: "log_17",
    timestamp: Date.now() - 344000,
    message: "✓  Dependencies installed in 1.3s",
  },
  {
    id: "log_18",
    timestamp: Date.now() - 343000,
    message: "Compiling project...",
  },
  {
    id: "log_19",
    timestamp: Date.now() - 342000,
    message: "✓ Build successful in 331ms",
  },
  {
    id: "log_20",
    timestamp: Date.now() - 341000,
    message: "Registering healthcheck: GET /health every 30s",
  },
  {
    id: "log_21",
    timestamp: Date.now() - 340000,
    message: "Checking availability in selected regions...",
  },
  {
    id: "log_22",
    timestamp: Date.now() - 339000,
    message: "✓ us-east-1 available (2 slots)",
  },
  {
    id: "log_23",
    timestamp: Date.now() - 338000,
    message: "✓ eu-west-1 available (1 slot)",
  },
  {
    id: "log_24",
    timestamp: Date.now() - 337000,
    message: "✓ ap-south-1 available (1 slot)",
  },
  {
    id: "log_25",
    timestamp: Date.now() - 336000,
    message: "Creating deployment image...",
  },
  {
    id: "log_26",
    timestamp: Date.now() - 335000,
    message:
      "❌ Error: Failed to optimize image layer for region eu-west-1. Using fallback strategy",
    level: "error",
  },
  {
    id: "log_27",
    timestamp: Date.now() - 334000,
    message: "✓ Image built: 210mb",
  },
  {
    id: "log_28",
    timestamp: Date.now() - 333000,
    message: "Launching 4 VM instances",
  },
  {
    id: "log_29",
    timestamp: Date.now() - 332000,
    message: "✓ Scaling enabled: 0–5 instances at 80% CPU",
  },
  {
    id: "log_30",
    timestamp: Date.now() - 331000,
    message: "Deploying to:",
  },
  {
    id: "log_31",
    timestamp: Date.now() - 330000,
    message: "  - api.gateway.com (https)",
  },
  {
    id: "log_32",
    timestamp: Date.now() - 329000,
    message: "  - internal.api.gateway.com (http)",
  },
  {
    id: "log_33",
    timestamp: Date.now() - 328000,
    message: "  - dashboard:3000, 8080, 5792",
  },
  {
    id: "log_34",
    timestamp: Date.now() - 327000,
    message: "Activating deployment: v_alpha001",
  },
  {
    id: "log_35",
    timestamp: Date.now() - 326000,
    message: "✓ Deployment active",
  },
  {
    id: "log_36",
    timestamp: Date.now() - 325000,
    message: "View logs at /dashboard/logs/alpha001",
  },
  {
    id: "log_37",
    timestamp: Date.now() - 324000,
    message: "Deployment completed in 5.7s",
  },
];

export type DeploymentLog = z.infer<typeof LogEntry>;
export type DeploymentLogsInput = z.infer<typeof deploymentLogsInputSchema>;

export const getDeploymentBuildLogs = t.procedure
  .use(requireUser)
  .use(requireWorkspace)
  .use(withRatelimit(ratelimit.read))
  .input(deploymentLogsInputSchema)
  .output(deploymentLogsOutputSchema)
  .query(async () => {
    // In real implementation: fetch from database/logging service by deploymentId
    // If not sorted, sort by timestamp asc for chronological build order
    const sortedLogs = MOCK_LOGS.sort((a, b) => a.timestamp - b.timestamp);
    return {
      logs: sortedLogs,
    };
  });
