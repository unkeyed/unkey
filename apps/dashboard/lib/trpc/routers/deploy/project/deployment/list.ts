import { deploymentListInputSchema } from "@/app/(app)/projects/[projectId]/deployments/components/table/deployments.schema";
import {
  DEPLOYMENT_STATUSES,
  type DeploymentStatus,
} from "@/app/(app)/projects/[projectId]/deployments/filters.schema";
import { ratelimit, requireUser, requireWorkspace, t, withRatelimit } from "@/lib/trpc/trpc";
import { TRPCError } from "@trpc/server";
import { getTimestampFromRelative } from "@unkey/ui/src/lib/utils";
import { z } from "zod";

const Status = z.enum(DEPLOYMENT_STATUSES);

const AuthorResponse = z.object({
  name: z.string(),
  image: z.string(),
});

const SourceResponse = z.object({
  branch: z.string(),
  gitSha: z.string(),
});

const PullRequestResponse = z.object({
  number: z.number(),
  title: z.string(),
  url: z.string(),
});

// Base deployment fields
const BaseDeploymentResponse = z.object({
  id: z.string(),
  status: Status,
  instances: z.number(),
  runtime: z.string().nullable(),
  size: z.string().nullable(),
  source: SourceResponse,
  createdAt: z.number(),
  author: AuthorResponse,
  description: z.string().nullable(),
  pullRequest: PullRequestResponse,
});

// Discriminated union for environment-specific deployments
const ProductionDeploymentResponse = BaseDeploymentResponse.extend({
  environment: z.literal("production"),
  active: z.boolean(), // Only one production deployment can be active
});

const PreviewDeploymentResponse = BaseDeploymentResponse.extend({
  environment: z.literal("preview"),
});

const DeploymentResponse = z.discriminatedUnion("environment", [
  ProductionDeploymentResponse,
  PreviewDeploymentResponse,
]);

const deploymentsOutputSchema = z.object({
  deployments: z.array(DeploymentResponse),
  hasMore: z.boolean(),
  total: z.number(),
  nextCursor: z.number().int().nullish(),
});

type DeploymentsOutputSchema = z.infer<typeof deploymentsOutputSchema>;
export type Deployment = z.infer<typeof DeploymentResponse>;

export const DEPLOYMENTS_LIMIT = 50;

export const queryDeployments = t.procedure
  .use(requireUser)
  .use(requireWorkspace)
  .use(withRatelimit(ratelimit.read))
  .input(deploymentListInputSchema)
  .output(deploymentsOutputSchema)
  .query(async ({ input }) => {
    try {
      const hardcodedDeployments = generateDeployments(10_000);

      // Apply filters to deployments
      let filteredDeployments = hardcodedDeployments;

      // Time range filters
      let startTime: number | null | undefined = null;
      let endTime: number | null | undefined = null;

      if (input.since !== null && input.since !== undefined) {
        try {
          startTime = getTimestampFromRelative(input.since);
          endTime = Date.now();
        } catch {
          throw new TRPCError({
            code: "BAD_REQUEST",
            message: `Invalid since format: ${input.since}. Expected format like "1h", "2d", "30m", "1w".`,
          });
        }
      } else {
        startTime = input.startTime;
        endTime = input.endTime;
      }

      // Apply time filters
      if (startTime !== null && startTime !== undefined) {
        filteredDeployments = filteredDeployments.filter(
          (deployment) => deployment.createdAt >= startTime,
        );
      }

      if (endTime !== null && endTime !== undefined) {
        filteredDeployments = filteredDeployments.filter(
          (deployment) => deployment.createdAt <= endTime,
        );
      }

      // Status filter - expand grouped statuses to actual statuses
      if (input.status && input.status.length > 0) {
        const expandedStatusValues: DeploymentStatus[] = [];

        input.status.forEach((filter) => {
          if (filter.operator !== "is") {
            throw new TRPCError({
              code: "BAD_REQUEST",
              message: `Unsupported status operator: ${filter.operator}`,
            });
          }

          // Expand grouped status to actual statuses
          switch (filter.value) {
            case "pending":
              expandedStatusValues.push("pending");
              break;
            case "building":
              expandedStatusValues.push(
                "downloading_docker_image",
                "building_rootfs",
                "uploading_rootfs",
                "creating_vm",
                "booting_vm",
                "assigning_domains",
              );
              break;
            case "completed":
              expandedStatusValues.push("completed");
              break;
            case "failed":
              expandedStatusValues.push("failed");
              break;
            default:
              throw new TRPCError({
                code: "BAD_REQUEST",
                message: `Unknown grouped status: ${filter.value}`,
              });
          }
        });

        filteredDeployments = filteredDeployments.filter((deployment) =>
          expandedStatusValues.includes(deployment.status),
        );
      }

      // Environment filter
      if (input.environment && input.environment.length > 0) {
        const environmentValues = input.environment.map((filter) => {
          if (filter.operator !== "is") {
            throw new TRPCError({
              code: "BAD_REQUEST",
              message: `Unsupported environment operator: ${filter.operator}`,
            });
          }
          return filter.value;
        });

        filteredDeployments = filteredDeployments.filter((deployment) =>
          environmentValues.includes(deployment.environment),
        );
      }

      // Branch filter
      if (input.branch && input.branch.length > 0) {
        filteredDeployments = filteredDeployments.filter((deployment) => {
          return input.branch?.some((filter) => {
            if (filter.operator === "is") {
              return deployment.source.branch === filter.value;
            }
            if (filter.operator === "contains") {
              return deployment.source.branch.toLowerCase().includes(filter.value.toLowerCase());
            }
            throw new TRPCError({
              code: "BAD_REQUEST",
              message: `Unsupported branch operator: ${filter.operator}`,
            });
          });
        });
      }

      // Apply cursor-based pagination after filtering
      if (input.cursor && typeof input.cursor === "number") {
        const cursor = input.cursor;
        filteredDeployments = filteredDeployments.filter(
          (deployment) => deployment.createdAt < cursor,
        );
      }

      //Separate active deployment before sorting/pagination
      const activeProductionDeployment = filteredDeployments.find(
        (deployment): deployment is z.infer<typeof ProductionDeploymentResponse> =>
          deployment.environment === "production" && deployment.active === true,
      );

      // Remove active deployment from main list to avoid duplicates
      const nonActiveDeployments = filteredDeployments.filter(
        (deployment) => !(deployment.environment === "production" && deployment.active === true),
      );

      // Sort non-active deployments by createdAt descending
      nonActiveDeployments.sort((a, b) => b.createdAt - a.createdAt);

      // Get total count before pagination
      const totalCount = filteredDeployments.length;

      // Apply pagination limit, accounting for active deployment
      const remainingSlots = activeProductionDeployment ? DEPLOYMENTS_LIMIT - 1 : DEPLOYMENTS_LIMIT;
      const paginatedDeployments = nonActiveDeployments.slice(0, remainingSlots);

      // Combine results with active deployment always first
      const finalDeployments: Deployment[] = [];
      if (activeProductionDeployment) {
        finalDeployments.push(activeProductionDeployment);
      }
      finalDeployments.push(...paginatedDeployments);

      const hasMore = nonActiveDeployments.length > remainingSlots;

      const response: DeploymentsOutputSchema = {
        deployments: finalDeployments,
        hasMore,
        total: totalCount,
        nextCursor:
          paginatedDeployments.length > 0
            ? paginatedDeployments[paginatedDeployments.length - 1].createdAt
            : null,
      };

      return response;
    } catch (error) {
      console.error("Error querying deployments:", error);
      throw new TRPCError({
        code: "INTERNAL_SERVER_ERROR",
        message:
          "Failed to retrieve deployments due to an error. If this issue persists, please contact support.",
      });
    }
  });

// Generator function for hardcoded data
const generateDeployments = (count: number): Deployment[] => {
  const authors = [
    {
      name: "imeyer",
      image: "https://avatars.githubusercontent.com/u/78000?v=4",
    },
    {
      name: "Flo4604",
      image: "https://avatars.githubusercontent.com/u/53355483?v=4",
    },
    {
      name: "ogzhanolguncu",
      image:
        "https://avatars.githubusercontent.com/u/21091016?s=400&u=788774b6cbffaa93e2b8eadcd10ef32e1c6ecf58&v=4",
    },
    {
      name: "perkinsjr",
      image: "https://avatars.githubusercontent.com/u/45409975?v=4",
    },
    {
      name: "mcstepp",
      image: "https://avatars.githubusercontent.com/u/7390124?v=4",
    },
    {
      name: "chronark",
      image: "https://avatars.githubusercontent.com/u/18246773?v=4",
    },
    {
      name: "MichaelUnkey",
      image: "https://avatars.githubusercontent.com/u/148160799?v=4",
    },
  ];

  const statuses: z.infer<typeof Status>[] = [
    "pending",
    "downloading_docker_image",
    "building_rootfs",
    "uploading_rootfs",
    "creating_vm",
    "booting_vm",
    "assigning_domains",
    "completed",
    "failed",
  ];

  const environments: Array<"production" | "preview"> = ["production", "preview"];
  const branches = ["main", "dev", "feature/auth", "hotfix/security", "staging"];
  const runtimes = ["58", "12", "43", "22", "38", "200", "400", "1000", "35", "362"];
  const sizes = ["512", "1024", "256", "2048", "4096", "8192"];
  const descriptions = [
    "Add auth routes + logging",
    "Patch: revert error state",
    "Major refactor prep",
    "Added rate limit env vars",
    "Boot up optimization",
    "Initial staging cut",
    "Clean up unused modules",
    "Old stable fallback",
    "Aborted config test",
    "Failing on init timeout",
  ];

  const prTitles = [
    "Fix authentication flow and add logging",
    "Revert error handling changes",
    "Prepare codebase for major refactor",
    "Add environment variables for rate limiting",
    "Optimize application boot sequence",
    "Initial deployment to staging environment",
    "Remove unused module dependencies",
    "Implement stable fallback mechanism",
    "Update configuration settings",
    "Fix initialization timeout issues",
  ];

  const deployments: Deployment[] = [];
  const baseTime = Date.now();
  let hasActiveProduction = false;

  for (let i = 0; i < count; i++) {
    const author = authors[i % authors.length];
    const status = statuses[i % statuses.length];
    const environment = environments[i % environments.length];
    const prNumber = Math.floor(Math.random() * 1000) + 1;

    const baseDeployment = {
      id: `deployment_${Math.random().toString(36).substr(2, 16)}`,
      status,
      instances: Math.floor(Math.random() * 5) + 1,
      runtime: status === "completed" ? runtimes[i % runtimes.length] : null,
      size: sizes[i % sizes.length],
      source: {
        branch: branches[i % branches.length],
        gitSha: Math.random().toString(36).substr(2, 7),
      },
      createdAt: baseTime - i * 1000 * 60 * Math.floor(Math.random() * 60),
      author,
      description: descriptions[i % descriptions.length],
      pullRequest: {
        number: prNumber,
        title: prTitles[i % prTitles.length],
        url: `https://github.com/unkeyed/unkey/pull/${prNumber}`,
      },
    };

    if (environment === "production") {
      // Only the first (most recent) production deployment is active
      const isActive = !hasActiveProduction;
      if (isActive) {
        hasActiveProduction = true;
      }

      deployments.push({
        ...baseDeployment,
        environment: "production" as const,
        active: isActive,
        status: "completed",
      });
    } else {
      deployments.push({
        ...baseDeployment,
        environment: "preview" as const,
      });
    }
  }

  return deployments.sort((a, b) => b.createdAt - a.createdAt);
};
