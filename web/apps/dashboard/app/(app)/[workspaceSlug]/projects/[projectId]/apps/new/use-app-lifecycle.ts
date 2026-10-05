"use client";

import { collection } from "@/lib/collections";
import { applyDefaultSettings } from "@/lib/collections/deploy/environment-settings";
import { SERVER_PLACEHOLDER } from "@/lib/collections/deploy/utils";
import { trpc } from "@/lib/trpc/client";
import { getErrorMessage, getUnkeyClient } from "@/lib/unkey-client";
import { ConflictErrorResponse } from "@unkey/api/models/errors";
import { toast } from "@unkey/ui";
import { z } from "zod";
import { findReusablePlaceholder, provisionalAppName, uniqueAppName } from "./app-name";

type InitialSettings = Parameters<typeof applyDefaultSettings>[3];

type CreateImageAppInput = {
  baseName: string;
  imageReference: string;
  settings: InitialSettings;
};

export type CreateAppResult = { ok: true; appId: string } | { ok: false; error: string | null };

const MAX_NAME_ATTEMPTS = 3;
const REGIONS_STALE_MS = 5 * 60 * 1000;

function isConflict(error: unknown): boolean {
  return (
    error instanceof ConflictErrorResponse ||
    (error instanceof Error && error.cause instanceof ConflictErrorResponse)
  );
}

export function useAppLifecycle(projectId: string) {
  const utils = trpc.useUtils();

  // The flow's live query keeps the apps loaded; a stale list ends in a conflict retry.
  const projectApps = async (fresh: boolean) => {
    if (fresh) {
      await collection.apps.utils.refetch();
    }
    return collection.apps.toArray.filter((app) => app.projectId === projectId);
  };

  const takenSlugs = async (attempt: number) =>
    new Set((await projectApps(attempt > 0)).map((app) => app.slug));

  const applyToEnvironments = async (appId: string, settings: InitialSettings) => {
    const { data: environments } = await getUnkeyClient().environments.listEnvironments({
      project: projectId,
      app: appId,
    });
    await Promise.all(
      environments.map((environment) =>
        applyDefaultSettings(projectId, appId, environment.id, settings),
      ),
    );
  };

  // The app already exists, so a settings failure must not orphan it; the
  // settings card shows what is missing.
  const applyOrWarn = async (appId: string, settings: () => Promise<InitialSettings>) => {
    try {
      await applyToEnvironments(appId, await settings());
    } catch (error) {
      toast.error("Could not load the app settings. Refresh the page to try again.", {
        description: getErrorMessage(error),
      });
    }
  };

  const insertImageApp = async (name: string, imageReference: string): Promise<string> => {
    const transaction = collection.apps.insert({
      projectId,
      name,
      slug: name,
      sourceType: "oci",
      imageReference,
      defaultBranch: "main",
      repositoryFullName: null,
      currentDeploymentId: null,
      isRolledBack: false,
      updatedAt: null,
      id: SERVER_PLACEHOLDER,
      domain: null,
      customDomain: null,
      headlineDeployment: null,
    });
    await transaction.isPersisted.promise;
    return z.object({ appId: z.string() }).parse(transaction.metadata).appId;
  };

  // The collection already shows the create error, so a failure only reports
  // that no app exists.
  const createImageApp = async ({
    baseName,
    imageReference,
    settings,
  }: CreateImageAppInput): Promise<CreateAppResult> => {
    for (let attempt = 0; attempt < MAX_NAME_ATTEMPTS; attempt++) {
      const name = uniqueAppName(baseName, await takenSlugs(attempt));
      try {
        const appId = await insertImageApp(name, imageReference);
        await applyOrWarn(appId, async () => settings);
        return { ok: true, appId };
      } catch (error) {
        if (!isConflict(error)) {
          return { ok: false, error: null };
        }
      }
    }
    return { ok: false, error: null };
  };

  // Skips the apps collection so the create raises no "App created" toast.
  const createGitApp = async (baseName: string): Promise<CreateAppResult> => {
    const regionsQuery = utils.deploy.environmentSettings.getAvailableRegions;
    void regionsQuery.prefetch(undefined, { staleTime: REGIONS_STALE_MS });
    for (let attempt = 0; attempt < MAX_NAME_ATTEMPTS; attempt++) {
      const name = uniqueAppName(baseName, await takenSlugs(attempt));
      try {
        const { data } = await getUnkeyClient().apps.createApp({
          project: projectId,
          name,
          slug: name,
          git: {},
        });
        await Promise.all([
          applyOrWarn(data.appId, async () => {
            const regions = await regionsQuery.fetch(undefined, { staleTime: REGIONS_STALE_MS });
            return { regionNames: regions.filter((r) => r.canSchedule).map((r) => r.name) };
          }),
          collection.apps.utils.refetch(),
        ]);
        return { ok: true, appId: data.appId };
      } catch (error) {
        if (!isConflict(error)) {
          return { ok: false, error: getErrorMessage(error) };
        }
      }
    }
    return { ok: false, error: "Could not find a free app name. Try again." };
  };

  const createGitPlaceholder = async (): Promise<CreateAppResult> => {
    const reusable = findReusablePlaceholder(await projectApps(false));
    return reusable ? { ok: true, appId: reusable } : createGitApp(provisionalAppName);
  };

  const renameApp = async (appId: string, baseName: string): Promise<void> => {
    for (let attempt = 0; attempt < MAX_NAME_ATTEMPTS; attempt++) {
      const apps = await projectApps(attempt > 0);
      if (apps.find((app) => app.id === appId)?.slug === baseName) {
        return;
      }
      const name = uniqueAppName(
        baseName,
        new Set(apps.filter((app) => app.id !== appId).map((app) => app.slug)),
      );
      try {
        await getUnkeyClient().apps.updateApp({ project: projectId, app: appId, name, slug: name });
        await collection.apps.utils.refetch();
        return;
      } catch (error) {
        if (!isConflict(error)) {
          throw error;
        }
      }
    }
  };

  const updateImage = async (appId: string, image: string): Promise<void> => {
    await getUnkeyClient().apps.updateApp({ project: projectId, app: appId, oci: { image } });
    await collection.apps.utils.refetch();
  };

  return { createImageApp, createGitApp, createGitPlaceholder, renameApp, updateImage };
}
