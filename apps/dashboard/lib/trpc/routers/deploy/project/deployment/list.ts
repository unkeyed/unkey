import { deploymentsQueryPayload as deploymentsInputSchema } from "@/app/(app)/deployments/_components/list/deployments-list.schema";
import {
  ratelimit,
  requireUser,
  requireWorkspace,
  t,
  withRatelimit,
} from "@/lib/trpc/trpc";
import { TRPCError } from "@trpc/server";
import { z } from "zod";

const DeploymentStatus = z.enum([
  "pending",
  "building",
  "deploying",
  "active",
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
  pullRequest: PullRequestResponse.nullable(),
});

const deploymentsOutputSchema = z.object({
  deployments: z.array(DeploymentResponse),
  hasMore: z.boolean(),
  total: z.number(),
  nextCursor: z.number().int().nullish(),
});

type DeploymentsOutputSchema = z.infer<typeof deploymentsOutputSchema>;
export type Deployment = z.infer<typeof DeploymentResponse>;

export const DEPLOYMENTS_LIMIT = 10;

export const queryDeployments = t.procedure
  .use(requireUser)
  .use(requireWorkspace)
  .use(withRatelimit(ratelimit.read))
  .output(deploymentsOutputSchema)
  .query(async ({ input }) => {
    try {
      // Generator function for hardcoded data
      const generateDeployments = (count: number): Deployment[] => {
        const authors = [
          {
            name: "Ian",
            image: "https://avatars.githubusercontent.com/u/78000?v=4",
          },
          {
            name: "Flo",
            image: "https://avatars.githubusercontent.com/u/53355483?v=4",
          },
          {
            name: "Oz",
            image:
              "https://avatars.githubusercontent.com/u/21091016?s=400&u=788774b6cbffaa93e2b8eadcd10ef32e1c6ecf58&v=4",
          },
          {
            name: "James",
            image: "https://avatars.githubusercontent.com/u/45409975?v=4",
          },
          {
            name: "Meg",
            image: "https://avatars.githubusercontent.com/u/7390124?v=4",
          },
          {
            name: "Chronark",
            image: "https://avatars.githubusercontent.com/u/18246773?v=4",
          },
          {
            name: "Mike",
            image: "https://avatars.githubusercontent.com/u/148160799?v=4",
          },
        ];

        const statuses: z.infer<typeof DeploymentStatus>[] = [
          "pending",
          "building",
          "deploying",
          "active",
          "failed",
        ];
        const branches = [
          "main",
          "dev",
          "feature/auth",
          "hotfix/security",
          "staging",
        ];
        const runtimes = ["58s", "12s", "43s", "22s", "38s"];
        const sizes = [
          "310mb",
          "305mb",
          "312mb",
          "316mb",
          "301mb",
          "298mb",
          "300mb",
        ];
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

        const deployments: Deployment[] = [];
        const baseTime = Date.now();

        for (let i = 0; i < count; i++) {
          const author = authors[i % authors.length];
          const status = statuses[i % statuses.length];
          const hasPR = Math.random() > 0.3; // 70% chance of having a PR

          deployments.push({
            id: `v_${Math.random().toString(36).substr(2, 8)}${String(
              i
            ).padStart(3, "0")}`,
            status,
            instances: Math.floor(Math.random() * 5) + 1,
            runtime: status === "active" ? runtimes[i % runtimes.length] : null,
            size:
              status === "active" || status === "failed"
                ? sizes[i % sizes.length]
                : null,
            source: {
              branch: branches[i % branches.length],
              gitSha: Math.random().toString(36).substr(2, 7),
            },
            createdAt:
              baseTime - i * 1000 * 60 * Math.floor(Math.random() * 60), // Random time in past
            author,
            description: descriptions[i % descriptions.length],
            pullRequest: hasPR
              ? {
                  number: Math.floor(Math.random() * 1000) + 1,
                  title: `Fix: ${descriptions[i % descriptions.length]}`,
                  url: `https://github.com/unkeyed/unkey/pull/${
                    Math.floor(Math.random() * 1000) + 1
                  }`,
                }
              : null,
          });
        }

        return deployments.sort((a, b) => b.createdAt - a.createdAt);
      };

      const hardcodedDeployments = generateDeployments(25);

      // Apply cursor-based pagination
      let filteredDeployments = hardcodedDeployments;

      if (input.cursor && typeof input.cursor === "number") {
        filteredDeployments = hardcodedDeployments.filter(
          (deployment) => deployment.createdAt < input.cursor
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
        total: hardcodedDeployments.length, // Total count before filtering
        nextCursor:
          deploymentsWithoutExtra.length > 0
            ? deploymentsWithoutExtra[deploymentsWithoutExtra.length - 1]
                .createdAt
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
