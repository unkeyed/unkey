import { queryCollectionOptions } from "@tanstack/query-db-collection";
import { createCollection } from "@tanstack/react-db";
import { toast } from "@unkey/ui";
import { z } from "zod";
import { queryClient, trpcClient } from "../client";

const schema = z.object({
  id: z.string(),
  name: z.string(),
  slug: z.string(),
  // Apps inside the project, newest first, for the card's app stack.
  appCount: z.number().int(),
  apps: z.array(
    z.object({
      id: z.string(),
      name: z.string(),
      source: z.enum(["github", "code"]),
      repository: z.string().nullable(),
    }),
  ),
  repositoryFullName: z.string().nullable(),
  latestDeploymentId: z.string().nullable(),
  currentDeploymentId: z.string().nullable(),
  isRolledBack: z.boolean(),
  // Flattened deployment fields for UI
  commitTitle: z.string().nullable(),
  commitSha: z.string().nullable(),
  forkRepositoryFullName: z.string().nullable(),
  prNumber: z.number().int().nullable(),
  branch: z.string(),
  author: z.string().nullable(),
  authorAvatar: z.string().nullable(),
  commitTimestamp: z.number().int().nullable(),
  // Domain field
  domain: z.string().nullable(),
});

export const createProjectRequestSchema = z.object({
  name: z.string().trim().min(1, "Project name is required").max(256, "Project name too long"),
  slug: z
    .string()
    .trim()
    .min(3, "Project slug must be at least 3 characters")
    .max(256, "Project slug too long")
    .regex(
      /^[a-z0-9]+([_-][a-z0-9]+)*$/,
      "Project slug must contain only lowercase letters, numbers, hyphens, and underscores",
    ),
});

export type Project = z.infer<typeof schema>;
export type CreateProjectRequestSchema = z.infer<typeof createProjectRequestSchema>;

export const projects = createCollection<Project, string>(
  queryCollectionOptions({
    queryClient,
    queryKey: ["projects"],
    retry: 3,
    refetchInterval: 5000,
    queryFn: async () => {
      return await trpcClient.deploy.project.list.query();
    },
    getKey: (item) => item.id,
    onDelete: async ({ transaction }) => {
      const mutation = transaction.mutations[0];
      const projectId = mutation.original.id;

      const deleteMutation = trpcClient.deploy.project.delete.mutate({
        projectId,
      });

      toast.promise(deleteMutation, {
        loading: "Deleting project...",
        success: "Project deleted successfully",
        error: (err) => {
          console.error("Failed to delete project", err);

          switch (err.data?.code) {
            case "NOT_FOUND":
              return {
                message: "Project Deletion Failed",
                description: "Unable to find the project. Please refresh and try again.",
              };
            case "FORBIDDEN":
              return {
                message: "Permission Denied",
                description: "You don't have permission to delete this project.",
              };
            case "INTERNAL_SERVER_ERROR":
              return {
                message: "Server Error",
                description:
                  "We encountered an issue while deleting your project. Please try again later or contact support at support@unkey.com",
              };
            default:
              return {
                message: "Failed to Delete Project",
                description: err.message || "An unexpected error occurred. Please try again later.",
              };
          }
        },
      });

      await deleteMutation;
      // Automatically refetches query after delete
    },
    onInsert: async ({ transaction }) => {
      const { changes } = transaction.mutations[0];

      const createInput = createProjectRequestSchema.parse({
        name: changes.name,
        slug: changes.slug,
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
                  "We encountered an issue while creating your project. Please try again later or contact support at support@unkey.com",
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
      const result = await mutation;
      transaction.metadata = {
        projectId: result.id,
      };
    },
    onUpdate: async ({ transaction }) => {
      const { original, changes } = transaction.mutations[0];

      const updateInput = {
        projectId: original.id,
        ...createProjectRequestSchema.pick({ name: true }).parse({
          name: changes.name ?? original.name,
        }),
      };
      const mutation = trpcClient.deploy.project.update.mutate(updateInput);

      toast.promise(mutation, {
        loading: "Updating project...",
        success: "Project updated successfully",
        error: (err) => {
          console.error("Failed to update project", err);

          switch (err.data?.code) {
            case "FORBIDDEN":
              return {
                message: "Permission Denied",
                description: err.message || "You don't have permission to update this project.",
              };
            case "NOT_FOUND":
              return {
                message: "Project Update Failed",
                description: "Unable to find the project. Please refresh and try again.",
              };
            case "INTERNAL_SERVER_ERROR":
              return {
                message: "Server Error",
                description:
                  "We encountered an issue while updating your project. Please try again later or contact support at support@unkey.com",
              };
            default:
              return {
                message: "Failed to Update Project",
                description: err.message || "An unexpected error occurred. Please try again later.",
              };
          }
        },
      });

      await mutation;
    },
  }),
);
