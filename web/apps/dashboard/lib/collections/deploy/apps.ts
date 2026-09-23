import { getErrorToast, getUnkeyClient } from "@/lib/unkey-client";
import { parseLoadSubsetOptions, queryCollectionOptions } from "@tanstack/query-db-collection";
import { createCollection } from "@tanstack/react-db";
import { NotFoundErrorResponse } from "@unkey/api/models/errors";
import { toast } from "@unkey/ui";
import { z } from "zod";
import { queryClient, trpcClient } from "../client";
import { DEPLOYMENT_STATUSES } from "./deployment-status";
import { extractStringValues } from "./utils";

const schema = z.object({
  id: z.string(),
  projectId: z.string(),
  name: z.string(),
  slug: z.string(),
  sourceType: z.enum(["unknown", "git", "oci"]),
  imageReference: z.string().nullable(),
  defaultBranch: z.string(),
  currentDeploymentId: z.string().nullable(),
  isRolledBack: z.boolean(),
  updatedAt: z.number().nullable(),
  repositoryFullName: z.string().nullable(),
  domain: z.string().nullable(),
  headlineDeployment: z
    .object({
      id: z.string(),
      status: z.enum(DEPLOYMENT_STATUSES),
      deployedAt: z.number().int(),
      commitMessage: z.string().nullable(),
      commitSha: z.string().nullable(),
      branch: z.string().nullable(),
      prNumber: z.number().int().nullable(),
      forkRepositoryFullName: z.string().nullable(),
    })
    .nullable(),
});

export const ociImageReferenceSchema = z
  .string()
  .trim()
  .min(1, "Image reference is required")
  .max(512, "Image reference too long");

export const appCreationSourceSchema = z.discriminatedUnion("kind", [
  z.object({ kind: z.literal("git") }),
  z.object({
    kind: z.literal("oci"),
    imageReference: ociImageReferenceSchema,
  }),
]);

export const createAppRequestSchema = z.object({
  projectId: z.string().min(1, "Project is required"),
  name: z.string().trim().min(1, "App name is required").max(256, "App name too long"),
  slug: z
    .string()
    .trim()
    .min(1, "App slug is required")
    .max(256, "App slug too long")
    .regex(/^[a-z0-9-]+$/, "App slug must contain only lowercase letters, numbers, and hyphens"),
  source: appCreationSourceSchema,
});

export type App = z.infer<typeof schema>;
export type CreateAppRequestSchema = z.infer<typeof createAppRequestSchema>;

/**
 * Global apps collection.
 *
 * IMPORTANT: All queries MUST filter by projectId with eq or inArray:
 * .where(({ app }) => eq(app.projectId, projectId))
 */
export const apps = createCollection<App, string>(
  queryCollectionOptions({
    queryClient,
    queryKey: (opts) => {
      const { filters } = parseLoadSubsetOptions(opts);
      return ["apps", ...extractStringValues(filters, "projectId")];
    },
    retry: 3,
    syncMode: "on-demand",
    queryFn: async (ctx) => {
      const { filters } = parseLoadSubsetOptions(ctx.meta?.loadSubsetOptions);
      const projectIds = extractStringValues(filters, "projectId");

      if (projectIds.length === 0) {
        throw new Error("Query must include an eq or inArray constraint on collection.projectId");
      }

      const perProject = await Promise.all(projectIds.map(listProjectApps));
      return perProject.flat();
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
        source:
          changes.sourceType === "oci"
            ? { kind: "oci", imageReference: changes.imageReference }
            : { kind: changes.sourceType },
      });
      const mutation = getUnkeyClient().apps.createApp(
        createInput.source.kind === "git"
          ? {
              project: createInput.projectId,
              name: createInput.name,
              slug: createInput.slug,
              git: {},
            }
          : {
              project: createInput.projectId,
              name: createInput.name,
              slug: createInput.slug,
              oci: { image: createInput.source.imageReference },
            },
      );

      toast.promise(mutation, {
        loading: "Creating app...",
        success: "App created successfully",
        error: (err) => {
          console.error("Failed to create app", err);
          return getErrorToast(err, "Failed to Create App");
        },
      });

      const result = await mutation;
      transaction.metadata = { appId: result.data.appId };
    },
  }),
);

async function listProjectApps(projectId: string): Promise<App[]> {
  try {
    const [pages, headlines, displayDomains] = await Promise.all([
      getUnkeyClient().apps.listApps({ project: projectId, limit: 100 }),
      trpcClient.deploy.deployment.listHeadlines.query({ projectId }),
      trpcClient.deploy.domain.listDisplayDomains.query({ projectId }),
    ]);
    const headlineByApp = new Map(headlines.map(({ appId, ...headline }) => [appId, headline]));
    const domainByApp = new Map(displayDomains.map((d) => [d.appId, d.domain]));

    const apps: App[] = [];
    for await (const page of pages) {
      for (const app of page.result.data) {
        const currentDeploymentId = app.currentDeploymentId ?? null;
        apps.push({
          id: app.id,
          projectId,
          name: app.name,
          slug: app.slug,
          sourceType: app.sourceType ?? "unknown",
          imageReference: app.oci?.image ?? null,
          defaultBranch: app.git?.defaultBranch || "main",
          currentDeploymentId,
          isRolledBack: app.isRolledBack,
          updatedAt: app.updatedAt ?? null,
          repositoryFullName: app.git?.repository ?? null,
          domain: currentDeploymentId ? (domainByApp.get(app.id) ?? null) : null,
          headlineDeployment: headlineByApp.get(app.id) ?? null,
        });
      }
    }
    return apps;
  } catch (error) {
    if (error instanceof NotFoundErrorResponse) {
      return [];
    }
    throw error;
  }
}
