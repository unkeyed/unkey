"use client";

import { collection } from "@/lib/collections";
import type { EnvironmentSettings } from "@/lib/collections/deploy/environment-settings";
import { and, eq, useLiveQuery } from "@tanstack/react-db";
import { useMemo } from "react";
import { useAppId, useProjectData } from "../../data-provider";

type MultiEnvironmentSettings = {
  production: EnvironmentSettings;
  preview: EnvironmentSettings;
};

export function useMultiEnvironmentSettings(): MultiEnvironmentSettings | null {
  const { environments, projectId } = useProjectData();
  const appId = useAppId();

  const productionEnvId = useMemo(
    () => environments.find((e) => e.kind === "production")?.id,
    [environments],
  );
  const previewEnvId = useMemo(
    () => environments.find((e) => e.kind === "preview")?.id,
    [environments],
  );

  const { data: productionData } = useLiveQuery(
    (q) =>
      q
        .from({ s: collection.environmentSettings })
        .where(({ s }) =>
          and(
            eq(s.projectId, projectId),
            eq(s.appId, appId),
            eq(s.environmentId, productionEnvId ?? ""),
          ),
        ),
    [productionEnvId, projectId, appId],
  );

  const { data: previewData } = useLiveQuery(
    (q) =>
      q
        .from({ s: collection.environmentSettings })
        .where(({ s }) =>
          and(
            eq(s.projectId, projectId),
            eq(s.appId, appId),
            eq(s.environmentId, previewEnvId ?? ""),
          ),
        ),
    [previewEnvId, projectId, appId],
  );

  const production = productionData?.at(0);
  const preview = previewData?.at(0);

  if (!production || !preview) {
    return null;
  }

  return { production, preview };
}
