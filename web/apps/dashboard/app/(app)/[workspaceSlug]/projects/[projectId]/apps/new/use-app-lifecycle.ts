"use client";

import {
  getAppData,
  getProjectAppsData,
  refetchProjectApps,
  removeAppData,
  useEnvironmentSettingsMutations,
} from "@/lib/deploy-data";
import { trpc } from "@/lib/trpc/client";
import { getErrorMessage, getErrorToast, getUnkeyClient } from "@/lib/unkey-client";
import { TRPCClientError } from "@trpc/client";
import type { AppGitCreateInput } from "@unkey/api/models/components";
import { ConflictErrorResponse, NotFoundErrorResponse } from "@unkey/api/models/errors";
import { toast } from "@unkey/ui";
import {
  APP_NAME_TAKEN,
  findReusablePlaceholder,
  provisionalAppName,
  uniqueAppName,
} from "./app-name";

type AppSource = { git: AppGitCreateInput } | { oci: { image: string } };

const MAX_NAME_ATTEMPTS = 3;
const REGIONS_STALE_MS = 5 * 60 * 1000;
const DEFAULT_REGION = "us-east-1";

class NoFreeAppNameError extends Error {
  constructor() {
    super("Could not find a free app name. Try again.");
  }
}

export class AppNameTakenError extends Error {
  constructor() {
    super(APP_NAME_TAKEN);
  }
}

export function newAppErrorMessage(cause: unknown): string {
  return cause instanceof NoFreeAppNameError || cause instanceof TRPCClientError
    ? cause.message
    : getErrorMessage(cause);
}

function isApiError(
  cause: unknown,
  type: typeof ConflictErrorResponse | typeof NotFoundErrorResponse,
): boolean {
  return cause instanceof type || (cause instanceof Error && cause.cause instanceof type);
}

export function useAppLifecycle(projectId: string) {
  const utils = trpc.useUtils();
  const { applyDefaultSettings } = useEnvironmentSettingsMutations();

  const takenSlugs = (excludeAppId: string | null): ReadonlySet<string> =>
    new Set(
      getProjectAppsData(projectId)
        .filter((app) => app.id !== excludeAppId)
        .map((app) => app.slug),
    );

  const withFreeSlug = async <T>(
    baseName: string,
    excludeAppId: string | null,
    write: (slug: string) => Promise<T>,
  ): Promise<T> => {
    for (let attempt = 0; attempt < MAX_NAME_ATTEMPTS; attempt++) {
      if (attempt > 0) {
        await refetchProjectApps(projectId);
      }
      try {
        return await write(uniqueAppName(baseName, takenSlugs(excludeAppId)));
      } catch (error) {
        if (!isApiError(error, ConflictErrorResponse)) {
          throw error;
        }
      }
    }
    throw new NoFreeAppNameError();
  };

  const applyDefaults = (appId: string) =>
    applyDefaultSettings(projectId, appId, async () => {
      const regions = await utils.deploy.environmentSettings.getAvailableRegions.fetch(undefined, {
        staleTime: REGIONS_STALE_MS,
      });
      return regions
        .filter((region) => region.canSchedule && region.name === DEFAULT_REGION)
        .map((region) => region.name);
    }).catch((error: unknown) => {
      if (!isApiError(error, NotFoundErrorResponse)) {
        const { message, description } = getErrorToast(error, "Could not apply default settings");
        toast.error(message, { description });
      }
    });

  const setAppName = async (appId: string, slug: string): Promise<void> => {
    if (getAppData(appId)?.slug === slug) {
      return;
    }
    try {
      await getUnkeyClient().apps.updateApp({ project: projectId, app: appId, name: slug, slug });
    } catch (error) {
      throw isApiError(error, ConflictErrorResponse) ? new AppNameTakenError() : error;
    }
    await refetchProjectApps(projectId);
  };

  const deleteApp = async (appId: string): Promise<void> => {
    try {
      await getUnkeyClient().apps.deleteApp({ project: projectId, app: appId });
    } catch (error) {
      if (!isApiError(error, NotFoundErrorResponse)) {
        throw error;
      }
    }
    removeAppData(appId);
  };

  const createApp = async (baseName: string, source: AppSource): Promise<string> => {
    const client = getUnkeyClient();
    const appId = await withFreeSlug(baseName, null, async (slug) => {
      const { data } = await client.apps.createApp({
        project: projectId,
        name: slug,
        slug,
        ...source,
      });
      return data.appId;
    });
    void applyDefaults(appId);
    try {
      await refetchProjectApps(projectId, { throwOnError: true });
    } catch (error) {
      await deleteApp(appId).catch(() => undefined);
      await refetchProjectApps(projectId).catch(() => undefined);
      throw error;
    }
    return appId;
  };

  const createGitApp = (baseName: string, git: AppGitCreateInput = {}) =>
    createApp(baseName, { git });

  const createImageApp = (baseName: string, image: string) =>
    createApp(baseName, { oci: { image } });

  const createGitPlaceholder = async (): Promise<string> =>
    findReusablePlaceholder(getProjectAppsData(projectId)) ?? createGitApp(provisionalAppName);

  const renameApp = async (appId: string, baseName: string): Promise<void> => {
    if (getAppData(appId)?.slug === baseName) {
      return;
    }
    await withFreeSlug(baseName, appId, (slug) =>
      getUnkeyClient().apps.updateApp({ project: projectId, app: appId, name: slug, slug }),
    );
    await refetchProjectApps(projectId);
  };

  return {
    createGitApp,
    createImageApp,
    createGitPlaceholder,
    renameApp,
    setAppName,
    takenSlugs,
    deleteApp,
  };
}
