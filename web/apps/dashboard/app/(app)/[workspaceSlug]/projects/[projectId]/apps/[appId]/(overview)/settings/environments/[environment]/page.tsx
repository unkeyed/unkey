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
import { branchFor } from "../branch";
import { ComputeOverview } from "./compute-overview";
import { BranchTracking, EnvironmentDomains } from "./environment-groups";

export default function EnvironmentSettingsPage() {
  const { environment: slug } = useParams<{ environment: string }>();
  const { environments, isEnvironmentsLoading } = useProjectData();
  const { hasRepository, defaultBranch } = useBuildSource();
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
        <ComputeOverview />
        <EnvironmentDomains environment={environment} />
        {hasRepository ? (
          <BranchTracking
            environment={environment}
            branch={branchFor(environment, hasRepository, defaultBranch)}
          />
        ) : null}
      </SettingsGroups>
    </EnvironmentSettingsScope>
  );
}
