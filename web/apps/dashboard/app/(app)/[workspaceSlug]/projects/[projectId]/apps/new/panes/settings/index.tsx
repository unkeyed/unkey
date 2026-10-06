"use client";

import { environmentsQueryFor } from "@/app/(app)/[workspaceSlug]/projects/[projectId]/apps/[appId]/(overview)/data-provider-queries";
import { collection } from "@/lib/collections";
import type { EnvironmentSettings } from "@/lib/collections/deploy/environment-settings";
import { ENVIRONMENT_KIND, type Environment } from "@/lib/collections/deploy/environments";
import { and, eq, useLiveQuery } from "@tanstack/react-db";
import type { SourceKind } from "../../wizard-model";
import { SettingsForm } from "./settings-form";
import { SettingsFormSkeleton } from "./skeleton";

type AppSettingsData =
  | { status: "loading" }
  | { status: "ready"; production: EnvironmentSettings; environmentIds: string[] };

function resolveSettingsData(
  environments: Environment[],
  settings: EnvironmentSettings[],
): AppSettingsData {
  const production =
    environments.find((env) => env.kind === ENVIRONMENT_KIND.production) ?? environments.at(0);
  const productionSettings = settings.find((s) => s.environmentId === production?.id);
  const allLoaded = environments.every((env) => settings.some((s) => s.environmentId === env.id));
  if (!productionSettings || !allLoaded) {
    return { status: "loading" };
  }
  return {
    status: "ready",
    production: productionSettings,
    environmentIds: environments.map((env) => env.id),
  };
}

export function useAppSettings(projectId: string, appId: string): AppSettingsData {
  const { data: environments } = useLiveQuery(environmentsQueryFor(projectId, [appId]), [
    projectId,
    appId,
  ]);
  const { data: settings } = useLiveQuery(
    (q) =>
      q
        .from({ s: collection.environmentSettings })
        .where(({ s }) => and(eq(s.projectId, projectId), eq(s.appId, appId))),
    [projectId, appId],
  );

  return resolveSettingsData(environments, settings);
}

type AppSettingsFormProps = {
  projectId: string;
  appId: string;
  source: SourceKind;
  onSaved: () => void;
};

export function AppSettingsForm({ projectId, appId, source, onSaved }: AppSettingsFormProps) {
  const data = useAppSettings(projectId, appId);
  if (data.status === "loading") {
    return <SettingsFormSkeleton />;
  }
  return (
    <SettingsForm
      appId={appId}
      source={source}
      production={data.production}
      environmentIds={data.environmentIds}
      onSaved={onSaved}
    />
  );
}
