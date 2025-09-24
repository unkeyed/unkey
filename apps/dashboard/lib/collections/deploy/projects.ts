import { queryCollectionOptions } from "@tanstack/query-db-collection";
import { createCollection } from "@tanstack/react-db";
import { toast } from "@unkey/ui";
import { z } from "zod";
import { queryClient, trpcClient } from "../client";

const schema = z.object({
  id: z.string(),
  name: z.string(),
  slug: z.string(),
  gitRepositoryUrl: z.string().nullable(),
  updatedAt: z.number().int().nullable(),
  liveDeploymentId: z.string().nullable(),
  rolledBackDeploymentId: z.string().nullable(),
  // Flattened deployment fields for UI
  commitTitle: z.string(),
  branch: z.string(),
  author: z.string(),
  commitTimestamp: z.number().int().nullable(),
  regions: z.array(z.string()),
  // Domain field
  domain: z.string(),
});

export const createProjectRequestSchema = z.object({
  name: z.string().trim().min(1, "Project name is required").max(256, "Project name too long"),
  slug: z
    .string()
    .trim()
    .min(1, "Project slug is required")
    .max(256, "Project slug too long")
    .regex(
      /^[a-z0-9-]+$/,
      "Project slug must contain only lowercase letters, numbers, and hyphens",
    ),
  gitRepositoryUrl: z.string().trim().url("Must be a valid URL").nullable().or(z.literal("")),
});

export type Project = z.infer<typeof schema>;
export type CreateProjectRequestSchema = z.infer<typeof createProjectRequestSchema>;

export const projects = createCollection<Project>(
  queryCollectionOptions({
    queryClient,
    queryKey: ["projects"],
    retry: 3,
    refetchInterval: 5000,
    queryFn: async () => {
      return await trpcClient.deploy.project.list.query();
    },
    getKey: (item) => item.id,
    onInsert: async ({ transaction }) => {
      const { changes } = transaction.mutations[0];

      const createInput = createProjectRequestSchema.parse({
        name: changes.name,
        slug: changes.slug,
        gitRepositoryUrl: changes.gitRepositoryUrl,
      });
      const mutation = trpcClient.deploy.project.create.mutate(createInput);

      toast.promise(mutation, {
        loading: "Creating project...",
        success: "Project created successfully",
        error: (err) => {
          console.error("Failed to create project", err);

          switch (err.data?.code) {
            case "CONFLICT":
              return {
                message: "Project Already Exists",
                description:
                  err.message || "A project with this slug already exists in your workspace.",
              };
            case "FORBIDDEN":
              return {
                message: "Permission Denied",
                description:
                  err.message || "You don't have permission to create projects in this workspace.",
              };
            case "BAD_REQUEST":
              return {
                message: "Invalid Configuration",
                description: `Please check your project settings. ${err.message || ""}`,
              };
            case "INTERNAL_SERVER_ERROR":
              return {
                message: "Server Error",
                description:
                  "We encountered an issue while creating your project. Please try again later or contact support at support@unkey.dev",
              };
            case "NOT_FOUND":
              return {
                message: "Project Creation Failed",
                description: "Unable to find the workspace. Please refresh and try again.",
              };
            default:
              return {
                message: "Failed to Create Project",
                description: err.message || "An unexpected error occurred. Please try again later.",
              };
          }
        },
      });
      await mutation;
    },
  }),
);
