import { ratelimit, requireUser, requireWorkspace, t, withRatelimit } from "@/lib/trpc/trpc";
import { z } from "zod";

const deploymentDetailsOutputSchema = z.object({
  // Active deployment
  repository: z.object({
    owner: z.string(),
    name: z.string(),
  }),
  branch: z.string(),
  commit: z.string(),
  description: z.string(),
  image: z.string(),
  author: z.object({
    name: z.string(),
    avatar: z.string(),
  }),
  createdAt: z.number(),

  // Runtime settings
  instances: z.number(),
  regions: z.array(z.string()),
  cpu: z.number(),
  memory: z.number(),
  storage: z.number(),
  healthcheck: z.object({
    method: z.string(),
    path: z.string(),
    interval: z.number(),
  }),
  scaling: z.object({
    min: z.number(),
    max: z.number(),
    threshold: z.number(),
  }),

  // Build info
  imageSize: z.number(),
  buildTime: z.number(),
  buildStatus: z.enum(["success", "failed", "pending"]),
  baseImage: z.string(),
  builtAt: z.number(),
});

type DeploymentDetailsOutputSchema = z.infer<typeof deploymentDetailsOutputSchema>;
export type DeploymentDetails = z.infer<typeof deploymentDetailsOutputSchema>;

export const getDeploymentDetails = t.procedure
  .use(requireUser)
  .use(requireWorkspace)
  .use(withRatelimit(ratelimit.read))
  .input(
    z.object({
      projectId: z.string(),
    }),
  )
  .output(deploymentDetailsOutputSchema)
  .query(() => {
    //TODO: This should make a db look-up find the "active" and "latest" and "prod" deployment
    const details: DeploymentDetailsOutputSchema = {
      repository: {
        owner: "acme",
        name: "acme",
      },
      branch: "main",
      commit: "e5f6a7b",
      description: "Add auth routes + logging",
      image: "unkey:latest",
      author: {
        name: "Oz",
        avatar: "https://avatars.githubusercontent.com/u/138932600?s=48&v=4",
      },
      createdAt: Date.now(),

      instances: 4,
      regions: ["eu-west-2", "us-east-1", "ap-southeast-1"],
      cpu: 32,
      memory: 512,
      storage: 1024,
      healthcheck: {
        method: "GET",
        path: "/health",
        interval: 30,
      },
      scaling: {
        min: 0,
        max: 5,
        threshold: 80,
      },

      imageSize: 210,
      buildTime: 45,
      buildStatus: "success",
      baseImage: "node:18-alpine",
      builtAt: Date.now() - 300000,
    };

    return details;
  });
