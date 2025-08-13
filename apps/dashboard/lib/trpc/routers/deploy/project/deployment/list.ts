import { deploymentsInputSchema } from "@/app/(app)/projects/[projectId]/deployments/components/table/deployments.schema";
import { ratelimit, requireUser, requireWorkspace, t, withRatelimit } from "@/lib/trpc/trpc";
import { TRPCError } from "@trpc/server";
import { z } from "zod";

const DeploymentStatus = z.enum([
  "pending",
  "downloading_docker_image",
  "building_rootfs",
  "uploading_rootfs",
  "creating_vm",
  "booting_vm",
  "assigning_domains",
  "completed",
  "failed",
]);

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

const DeploymentResponse = z.object({
  id: z.string(),
  status: DeploymentStatus,
  instances: z.number(),
  runtime: z.string().nullable(),
  size: z.string().nullable(),
  source: SourceResponse,
  createdAt: z.number(),
  author: AuthorResponse,
  description: z.string().nullable(),
  pullRequest: PullRequestResponse,
});

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
  .input(deploymentsInputSchema)
  .output(deploymentsOutputSchema)
  .query(async ({ input }) => {
    try {
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

        const statuses: z.infer<typeof DeploymentStatus>[] = [
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

        for (let i = 0; i < count; i++) {
          const author = authors[i % authors.length];
          const status = statuses[i % statuses.length];
          const prNumber = Math.floor(Math.random() * 1000) + 1;

          deployments.push({
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
          });
        }

        return deployments.sort((a, b) => b.createdAt - a.createdAt);
      };

      const hardcodedDeployments = generateDeployments(100);

      // Apply cursor-based pagination
      let filteredDeployments = hardcodedDeployments;

      if (input.cursor && typeof input.cursor === "number") {
        const cursor = input.cursor;
        filteredDeployments = hardcodedDeployments.filter(
          (deployment) => deployment.createdAt < cursor,
        );
      }

      // Sort by createdAt descending
      filteredDeployments.sort((a, b) => b.createdAt - a.createdAt);

      // Apply pagination limit
      const hasMore = filteredDeployments.length > DEPLOYMENTS_LIMIT;
      const deploymentsWithoutExtra = hasMore
        ? filteredDeployments.slice(0, DEPLOYMENTS_LIMIT)
        : filteredDeployments;

      const response: DeploymentsOutputSchema = {
        deployments: deploymentsWithoutExtra,
        hasMore,
        total: hardcodedDeployments.length,
        nextCursor:
          deploymentsWithoutExtra.length > 0
            ? deploymentsWithoutExtra[deploymentsWithoutExtra.length - 1].createdAt
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
