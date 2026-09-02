"use client";

import { and, eq, useLiveQuery } from "@tanstack/react-db";
import { collection } from "@/lib/collections";
import type { EnvironmentSettings } from "@/lib/collections/deploy/environment-settings";
import { ENVIRONMENT_KIND } from "@/lib/collections/deploy/environments";
import { useAppId, useProjectData } from "../../data-provider";

type MultiEnvironmentSettings = {
  production: EnvironmentSettings;
  preview: EnvironmentSettings;
};

export function useMultiEnvironmentSettings(): MultiEnvironmentSettings | null {
  const { environments, projectId } = useProjectData();
  const appId = useAppId();

  const { data } = useLiveQuery(
    (q) =>
      q
        .from({ s: collection.environmentSettings })
        .where(({ s }) => and(eq(s.projectId, projectId), eq(s.appId, appId))),
    [projectId, appId],
  );

  const productionEnvId = environments.find((e) => e.kind === ENVIRONMENT_KIND.production)?.id;
  const previewEnvId = environments.find((e) => e.kind === ENVIRONMENT_KIND.preview)?.id;

  const production = data?.find((s) => s.environmentId === productionEnvId);
  const preview = data?.find((s) => s.environmentId === previewEnvId);

  if (!production || !preview) {
    return null;
  }

  return { production, preview };
}
