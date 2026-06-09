import { parseLoadSubsetOptions, queryCollectionOptions } from "@tanstack/query-db-collection";
import { createCollection } from "@tanstack/react-db";
import { toast } from "@unkey/ui";
import { z } from "zod";
import { queryClient, trpcClient } from "../client";

const schema = z.object({
  id: z.string(),
  projectId: z.string(),
  name: z.string(),
  slug: z.string(),
  defaultBranch: z.string(),
  currentDeploymentId: z.string().nullable(),
  isRolledBack: z.boolean(),
  repositoryFullName: z.string().nullable(),
  latestDeploymentId: z.string().nullable(),
  // Flattened current-deployment fields for the shared deployable card.
  commitTitle: z.string().nullable(),
  branch: z.string(),
  author: z.string().nullable(),
  authorAvatar: z.string().nullable(),
  commitTimestamp: z.number().int().nullable(),
  domain: z.string().nullable(),
});

export const createAppRequestSchema = z.object({
  projectId: z.string().min(1, "Project is required"),
  name: z.string().trim().min(1, "App name is required").max(256, "App name too long"),
  slug: z
    .string()
    .trim()
    .min(1, "App slug is required")
    .max(256, "App slug too long")
    .regex(/^[a-z0-9-]+$/, "App slug must contain only lowercase letters, numbers, and hyphens"),
});

export type App = z.infer<typeof schema>;
export type CreateAppRequestSchema = z.infer<typeof createAppRequestSchema>;

type ParsedFilter = { field: Array<string | number>; operator: string; value?: unknown };

function extractStringFilter(filters: ParsedFilter[], fieldName: string, operator: string) {
  const value = filters.find((f) => f.field.at(-1) === fieldName && f.operator === operator)?.value;
  return typeof value === "string" ? value : undefined;
}

/**
 * Global apps collection.
 *
 * IMPORTANT: All queries MUST filter by projectId:
 * .where(({ app }) => eq(app.projectId, projectId))
 */
export const apps = createCollection<App, string>(
  queryCollectionOptions({
    queryClient,
    queryKey: (opts) => {
      const { filters } = parseLoadSubsetOptions(opts);
      const projectId = extractStringFilter(filters, "projectId", "eq");
      return projectId ? ["apps", projectId] : ["apps"];
    },
    retry: 3,
    syncMode: "on-demand",
    refetchInterval: 5000,
    queryFn: async (ctx) => {
      const { filters } = parseLoadSubsetOptions(ctx.meta?.loadSubsetOptions);
      const projectId = extractStringFilter(filters, "projectId", "eq");

      if (!projectId) {
        throw new Error("Query must include eq(collection.projectId, projectId) constraint");
      }

      return trpcClient.deploy.app.list.query({ projectId });
    },
    getKey: (item) => item.id,
    id: "apps",
    onDelete: async ({ transaction }) => {
      const mutation = transaction.mutations[0];
      const appId = mutation.original.id;

      const deleteMutation = trpcClient.deploy.app.delete.mutate({ appId });

      toast.promise(deleteMutation, {
        loading: "Deleting app...",
        success: "App deleted successfully",
        error: (err) => {
          console.error("Failed to delete app", err);
          switch (err.data?.code) {
            case "NOT_FOUND":
              return {
                message: "App Deletion Failed",
                description: "Unable to find the app. Please refresh and try again.",
              };
            case "FORBIDDEN":
              return {
                message: "Permission Denied",
                description: "You don't have permission to delete this app.",
              };
            case "PRECONDITION_FAILED":
              return {
                message: "Delete Protection Enabled",
                description: err.message || "Disable delete protection before deleting this app.",
              };
            case "INTERNAL_SERVER_ERROR":
              return {
                message: "Server Error",
                description:
                  "We encountered an issue while deleting your app. Please try again later or contact support at support@unkey.com",
              };
            default:
              return {
                message: "Failed to Delete App",
                description: err.message || "An unexpected error occurred. Please try again later.",
              };
          }
        },
      });

      await deleteMutation;
    },
    onInsert: async ({ transaction }) => {
      const { changes } = transaction.mutations[0];

      const createInput = createAppRequestSchema.parse({
        projectId: changes.projectId,
        name: changes.name,
        slug: changes.slug,
      });
      const mutation = trpcClient.deploy.app.create.mutate(createInput);

      toast.promise(mutation, {
        loading: "Creating app...",
        success: "App created successfully",
        error: (err) => {
          console.error("Failed to create app", err);
          switch (err.data?.code) {
            case "CONFLICT":
              return {
                message: "App Already Exists",
                description: err.message || "An app with this slug already exists in this project.",
              };
            case "FORBIDDEN":
              return {
                message: "Permission Denied",
                description:
                  err.message || "You don't have permission to create apps in this project.",
              };
            case "NOT_FOUND":
              return {
                message: "App Creation Failed",
                description: "Unable to find the project. Please refresh and try again.",
              };
            case "INTERNAL_SERVER_ERROR":
              return {
                message: "Server Error",
                description:
                  "We encountered an issue while creating your app. Please try again later or contact support at support@unkey.com",
              };
            default:
              return {
                message: "Failed to Create App",
                description: err.message || "An unexpected error occurred. Please try again later.",
              };
          }
        },
      });

      const result = await mutation;
      transaction.metadata = {
        appId: result.id,
      };
    },
  }),
);
