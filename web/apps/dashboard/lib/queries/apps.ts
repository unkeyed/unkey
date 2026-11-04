"use client";
import {
  type DeploymentStatus,
  isDeploymentInFlight,
} from "@/lib/collections/deploy/deployment-status";
import { trackSave } from "@/lib/collections/deploy/pending-redeploy";
import { sanitizeImageRef, validateImageRef } from "@/lib/docker-image-ref";
import { DEPLOY_DATA_STALE_MS } from "@/lib/queries/stale-time";
import { queryKeys } from "@/lib/query-keys";
import { trpcClient } from "@/lib/trpc/client";
import { getErrorToast, getUnkeyClient } from "@/lib/unkey-client";
import { type QueryClient, useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { NotFoundErrorResponse } from "@unkey/api/models/errors";
import { toast } from "@unkey/ui";
import { z } from "zod";

export type App = {
  id: string;
  projectId: string;
  name: string;
  slug: string;
  sourceType: "unknown" | "git" | "oci";
  imageReference: string | null;
  defaultBranch: string;
  currentDeploymentId: string | null;
  isRolledBack: boolean;
  updatedAt: number | null;
  repositoryFullName: string | null;
  domain: string | null;
  customDomain: string | null;
  headlineDeployment: {
    id: string;
    status: DeploymentStatus;
    deployedAt: number;
    commitMessage: string | null;
    commitSha: string | null;
    branch: string | null;
    prNumber: number | null;
    forkRepositoryFullName: string | null;
  } | null;
};

export type ProjectApp = Pick<App, "id" | "name" | "customDomain" | "headlineDeployment">;

export const ociImageReferenceSchema = z
  .string()
  .transform(sanitizeImageRef)
  .superRefine((ref, ctx) => {
    const validation = validateImageRef(ref);
    if (!validation.ok) {
      ctx.addIssue({
        code: "custom",
        message: ref ? validation.error : "Enter an image, like ghcr.io/unkeyed/deploy-demo:latest",
      });
    }
  });

export const APP_SLUG_LENGTH = { min: 3, max: 255 };

export const createAppRequestSchema = z.object({
  projectId: z.string().min(1, "Project is required"),
  name: z.string().trim().min(1, "App name is required").max(256, "App name too long"),
  slug: z
    .string()
    .trim()
    .min(1, "Enter a name")
    .min(APP_SLUG_LENGTH.min, `Use at least ${APP_SLUG_LENGTH.min} characters`)
    .max(APP_SLUG_LENGTH.max, `Use ${APP_SLUG_LENGTH.max} characters or fewer`)
    .regex(/^[a-z0-9-]+$/, "App slug must contain only lowercase letters, numbers, and hyphens"),
});

export type AppDeploymentRefs = {
  headlineDeployment: { id: string } | null;
  currentDeploymentId: string | null;
};

export function latestDeploymentId(app: AppDeploymentRefs): string | null {
  return app.headlineDeployment?.id ?? app.currentDeploymentId;
}

const IN_FLIGHT_POLL_MS = 5_000;
const IDLE_POLL_MS = 60_000;

export function appsPollInterval(apps: readonly App[] | undefined): number {
  const inFlight = (apps ?? []).some(
    (app) => app.headlineDeployment !== null && isDeploymentInFlight(app.headlineDeployment.status),
  );
  return inFlight ? IN_FLIGHT_POLL_MS : IDLE_POLL_MS;
}

export function useProjectApps(
  projectId: string,
  {
    enabled = true,
    refetchInterval,
  }: {
    enabled?: boolean;
    refetchInterval?: number | ((apps: App[] | undefined) => number | false);
  } = {},
) {
  return useQuery({
    ...projectAppsOptions(projectId),
    enabled,
    refetchInterval,
  });
}

export function refetchProjectApps(
  queryClient: QueryClient,
  projectId: string,
  { throwOnError = false } = {},
) {
  return queryClient.refetchQueries(
    { queryKey: queryKeys.apps.list(projectId), type: "active" },
    { throwOnError },
  );
}

export function getProjectApps(queryClient: QueryClient, projectId: string): App[] {
  return queryClient.getQueryData<App[]>(queryKeys.apps.list(projectId)) ?? [];
}

export function getApp(
  queryClient: QueryClient,
  projectId: string,
  appId: string,
): App | undefined {
  return getProjectApps(queryClient, projectId).find((app) => app.id === appId);
}

export function removeApp(queryClient: QueryClient, projectId: string, appId: string) {
  queryClient.setQueryData<App[]>(queryKeys.apps.list(projectId), (apps) =>
    apps?.filter((app) => app.id !== appId),
  );
}

type AppChanges = Partial<Pick<App, "name" | "defaultBranch" | "imageReference">>;

// Writes the saved fields into the cache instead of refetching, so a save does not wait on the list
export function useUpdateApp(projectId: string) {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: async ({ appId, changes }: { appId: string; changes: AppChanges }) => {
      const request = getUnkeyClient().apps.updateApp({
        project: projectId,
        app: appId,
        name: changes.name,
        oci: changes.imageReference ? { image: changes.imageReference } : undefined,
        git: changes.defaultBranch ? { defaultBranch: changes.defaultBranch } : undefined,
      });
      toast.promise(request, {
        loading: "Saving app...",
        success: "App updated",
        error: (err) => getErrorToast(err, "Failed to update app"),
      });
      await (changes.imageReference ? trackSave(request, { kind: "app", appId }) : request);
    },
    onSuccess: (_data, { appId, changes }) => {
      queryClient.setQueryData<App[]>(queryKeys.apps.list(projectId), (apps) =>
        apps?.map((app) => (app.id === appId ? { ...app, ...changes } : app)),
      );
    },
  });
}

/**
 * Optimistic, and not refetched after: the API only queues the delete, so a
 * refetch right after it would still list the app
 */
export function useDeleteApp(projectId: string) {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: async (appId: string) => {
      const request = getUnkeyClient().apps.deleteApp({ project: projectId, app: appId });
      toast.promise(request, {
        loading: "Deleting app...",
        success: "App deleted successfully",
        error: (err) => {
          console.error("Failed to delete app", err);
          return getErrorToast(err, "Failed to Delete App");
        },
      });
      await request;
    },
    onMutate: async (appId) => {
      const key = queryKeys.apps.list(projectId);
      await queryClient.cancelQueries({ queryKey: key });
      const previous = queryClient.getQueryData<App[]>(key);
      removeApp(queryClient, projectId, appId);
      return { previous };
    },
    onError: (_error, _appId, context) => {
      queryClient.setQueryData(queryKeys.apps.list(projectId), context?.previous);
    },
  });
}

// One query per project, so every screen listing a project shares its cache
export function projectAppsOptions(projectId: string) {
  return {
    queryKey: queryKeys.apps.list(projectId),
    staleTime: DEPLOY_DATA_STALE_MS,
    retry: 3,
    queryFn: ({ signal }: { signal?: AbortSignal }) => loadProjectApps(projectId, signal),
  };
}

// A project deleted under the page lists no apps rather than failing
export async function loadProjectApps(
  projectId: string,
  signal: AbortSignal | undefined,
): Promise<App[]> {
  try {
    const [pages, headlines, displayDomains] = await Promise.all([
      getUnkeyClient()
        .apps.listApps({ project: projectId, limit: 100 }, { signal })
        .then((firstPage) => Array.fromAsync(firstPage)),
      trpcClient.deploy.deployment.listHeadlines.query({ projectId }, { signal }),
      trpcClient.deploy.domain.listDisplayDomains.query({ projectId }, { signal }),
    ]);
    const headlineByApp = new Map(headlines.map(({ appId, ...headline }) => [appId, headline]));
    const domainsByApp = new Map(displayDomains.map((d) => [d.appId, d]));

    const apps = pages.flatMap((page) =>
      page.result.data.map((app): App => {
        const currentDeploymentId = app.currentDeploymentId ?? null;
        return {
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
          domain: currentDeploymentId ? (domainsByApp.get(app.id)?.domain ?? null) : null,
          customDomain: currentDeploymentId
            ? (domainsByApp.get(app.id)?.customDomain ?? null)
            : null,
          headlineDeployment: headlineByApp.get(app.id) ?? null,
        };
      }),
    );
    return apps.toSorted(byLatestUpdate);
  } catch (error) {
    if (error instanceof NotFoundErrorResponse) {
      return [];
    }
    throw error;
  }
}

export function byLatestUpdate(
  a: Pick<App, "id" | "updatedAt">,
  b: Pick<App, "id" | "updatedAt">,
): number {
  return (b.updatedAt ?? -1) - (a.updatedAt ?? -1) || (b.id < a.id ? -1 : b.id > a.id ? 1 : 0);
}
