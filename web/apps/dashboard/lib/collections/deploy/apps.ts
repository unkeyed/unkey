import { getErrorToast, getUnkeyClient } from "@/lib/unkey-client";
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
  commitSha: z.string().nullable(),
  forkRepositoryFullName: z.string().nullable(),
  prNumber: z.number().int().nullable(),
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
      const { original } = transaction.mutations[0];

      const deleteMutation = getUnkeyClient().apps.deleteApp({
        project: original.projectId,
        app: original.id,
      });

      toast.promise(deleteMutation, {
        loading: "Deleting app...",
        success: "App deleted successfully",
        error: (err) => {
          console.error("Failed to delete app", err);
          return getErrorToast(err, "Failed to Delete App");
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
      const mutation = getUnkeyClient().apps.createApp({
        project: createInput.projectId,
        name: createInput.name,
        slug: createInput.slug,
      });

      toast.promise(mutation, {
        loading: "Creating app...",
        success: "App created successfully",
        error: (err) => {
          console.error("Failed to create app", err);
          return getErrorToast(err, "Failed to Create App");
        },
      });

      const result = await mutation;
      transaction.metadata = {
        appId: result.data.appId,
      };
    },
  }),
);
