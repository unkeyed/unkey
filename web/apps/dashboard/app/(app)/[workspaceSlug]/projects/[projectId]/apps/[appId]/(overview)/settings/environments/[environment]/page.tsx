"use client";

import { LoadError } from "@/components/load-error";
import { useEnvironments } from "@/lib/deploy-data";
import { useFlag } from "@/lib/flags/provider";
import { SettingsGroups } from "@unkey/ui";
import { notFound, useParams } from "next/navigation";
import { EnvironmentSettingsScope } from "../../../../../_components/settings/environment-provider";
import { useBuildSource } from "../../../../../_components/settings/hooks/use-build-source";
import { SettingsSkeleton } from "../../../../../_components/settings/settings-skeleton";
import { BranchTracking, EnvironmentDomains } from "./environment-groups";
import { RuntimeOverview as LegacyRuntimeOverview } from "./legacy/runtime-overview";
import { RuntimeOverview } from "./runtime-overview";

export default function EnvironmentSettingsPage() {
  const { environment: slug } = useParams<{ environment: string }>();
  const {
    data: environments,
    isLoading: isEnvironmentsLoading,
    isError,
    refetch,
  } = useEnvironments();
  const { hasRepository } = useBuildSource();
  const appV2 = useFlag("appV2");
  const environment = environments.find((e) => e.slug === slug);

  if (!environment) {
    if (isEnvironmentsLoading) {
      return <SettingsSkeleton />;
    }
    if (isError) {
      return <LoadError title="Could not load environments" onRetry={refetch} />;
    }
    notFound();
  }

  return (
    <EnvironmentSettingsScope environmentId={environment.id}>
      <SettingsGroups>
        {appV2 ? <RuntimeOverview /> : <LegacyRuntimeOverview />}
        <EnvironmentDomains environment={environment} />
        {hasRepository ? <BranchTracking environment={environment} /> : null}
      </SettingsGroups>
    </EnvironmentSettingsScope>
  );
}
