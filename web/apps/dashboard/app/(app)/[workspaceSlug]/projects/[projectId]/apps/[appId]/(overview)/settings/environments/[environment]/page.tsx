"use client";

import { LoadError } from "@/components/load-error";
import { collection } from "@/lib/collections";
import { useCollectionLoad } from "@/lib/collections/use-collection-load";
import { SettingsGroups } from "@unkey/ui";
import { notFound, useParams } from "next/navigation";
import { EnvironmentSettingsScope } from "../../../../../_components/settings/environment-provider";
import { useBuildSource } from "../../../../../_components/settings/hooks/use-build-source";
import { SettingsSkeleton } from "../../../../../_components/settings/settings-skeleton";
import { useProjectData } from "../../../data-provider";
import { BranchTracking, EnvironmentDomains } from "./environment-groups";
import { RuntimeOverview } from "./runtime-overview";

export default function EnvironmentSettingsPage() {
  const { environment: slug } = useParams<{ environment: string }>();
  const { environments, isEnvironmentsLoading } = useProjectData();
  const { hasRepository } = useBuildSource();
  const environmentsLoad = useCollectionLoad(collection.environments.utils);
  const environment = environments.find((e) => e.slug === slug);

  if (!environment) {
    if (isEnvironmentsLoading) {
      return <SettingsSkeleton />;
    }
    if (environmentsLoad.failed) {
      return <LoadError title="Could not load environments" onRetry={environmentsLoad.retry} />;
    }
    notFound();
  }

  return (
    <EnvironmentSettingsScope environmentId={environment.id}>
      <SettingsGroups>
        <RuntimeOverview />
        <EnvironmentDomains environment={environment} />
        {hasRepository ? <BranchTracking environment={environment} /> : null}
      </SettingsGroups>
    </EnvironmentSettingsScope>
  );
}
