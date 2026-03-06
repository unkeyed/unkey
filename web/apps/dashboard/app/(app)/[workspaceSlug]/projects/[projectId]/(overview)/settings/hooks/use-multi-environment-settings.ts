"use client";

import { collection } from "@/lib/collections";
import type { EnvironmentSettings } from "@/lib/collections/deploy/environment-settings";
import { eq, useLiveQuery } from "@tanstack/react-db";
import { useMemo } from "react";
import { useProjectData } from "../../data-provider";

type MultiEnvironmentSettings = {
  production: EnvironmentSettings;
  preview: EnvironmentSettings;
};

export function useMultiEnvironmentSettings(): MultiEnvironmentSettings | null {
  const { environments } = useProjectData();

  const productionEnvId = useMemo(
    () => environments.find((e) => e.slug === "production")?.id,
    [environments],
  );
  const previewEnvId = useMemo(
    () => environments.find((e) => e.slug === "preview")?.id,
    [environments],
  );

  const { data: productionData } = useLiveQuery(
    (q) =>
      q
        .from({ s: collection.environmentSettings })
        .where(({ s }) => eq(s.environmentId, productionEnvId ?? "")),
    [productionEnvId],
  );

  const { data: previewData } = useLiveQuery(
    (q) =>
      q
        .from({ s: collection.environmentSettings })
        .where(({ s }) => eq(s.environmentId, previewEnvId ?? "")),
    [previewEnvId],
  );

  const production = productionData?.at(0);
  const preview = previewData?.at(0);

  if (!production || !preview) {
    return null;
  }

  return { production, preview };
}
