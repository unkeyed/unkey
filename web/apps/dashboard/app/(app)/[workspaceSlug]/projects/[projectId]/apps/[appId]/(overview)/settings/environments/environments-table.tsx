"use client";

import { LoadError } from "@/components/load-error";
import { RegionFlag } from "@/components/region-flag";
import { useWorkspaceNavigation } from "@/hooks/use-workspace-navigation";
import { collection } from "@/lib/collections";
import { type Environment, productionFirst } from "@/lib/collections/deploy/environments";
import { useCollectionLoad } from "@/lib/collections/use-collection-load";
import { routes } from "@/lib/navigation/routes";
import { type RegionInfo, regionInfo } from "@/lib/regions";
import { IconChevronRightOutline18, IconEarthOutline18 } from "@unkey/icons";
import {
  Badge,
  InfoTooltip,
  ResourceListBody,
  ResourceListContent,
  ResourceListItem,
} from "@unkey/ui";
import Link from "next/link";
import { EnvironmentLabel } from "../../../../_components/environment-label";
import { useAppEnvironmentSettings } from "../../../../_components/settings/environment-provider";
import { useBuildSource } from "../../../../_components/settings/hooks/use-build-source";
import { useProjectData } from "../../data-provider";
import { type Branch, branchFor } from "./branch";
import { BranchLabel } from "./branch-label";
import { environmentDomains } from "./model";

const MAX_REGION_FLAGS = 3;

export function EnvironmentsTable() {
  const { projectId, appId, hasRepository, defaultBranch } = useBuildSource();
  const { environments, domains, customDomains, isEnvironmentsLoading } = useProjectData();
  const environmentsLoad = useCollectionLoad(collection.environments.utils);
  const workspace = useWorkspaceNavigation();
  const settings = useAppEnvironmentSettings();

  if (environmentsLoad.failed && !isEnvironmentsLoading) {
    return <LoadError title="Could not load environments" onRetry={environmentsLoad.retry} />;
  }

  return (
    <ResourceListContent className="[--env-columns:minmax(6.5rem,1fr)_minmax(0,1.1fr)_minmax(0,1.6fr)_minmax(0,1.2fr)_16px]">
      <div className="grid grid-cols-(--env-columns) items-center gap-4 border-b bg-table-header px-4 py-[7px] text-xs font-medium text-gray-12">
        <span>Environment</span>
        <span>Branch</span>
        <span>Domains</span>
        <span>Regions</span>
        <span />
      </div>
      <ResourceListBody>
        {productionFirst(environments).map((environment) => (
          <EnvironmentRow
            key={environment.id}
            environment={environment}
            href={routes.projects.apps.settings({
              workspaceSlug: workspace.slug,
              projectId,
              appId,
              page: "environments",
              environment: environment.slug,
            })}
            branch={branchFor(environment, hasRepository, defaultBranch)}
            domains={environmentDomains(environment, domains, customDomains).map((d) => d.hostname)}
            regions={(settings.find((s) => s.environmentId === environment.id)?.regions ?? []).map(
              (r) => regionInfo(r.name),
            )}
          />
        ))}
      </ResourceListBody>
    </ResourceListContent>
  );
}

function EnvironmentRow({
  environment,
  href,
  branch,
  domains,
  regions,
}: {
  environment: Environment;
  href: ReturnType<typeof routes.projects.apps.settings>;
  branch: Branch;
  domains: string[];
  regions: RegionInfo[];
}) {
  const [first, ...rest] = domains;
  const shownRegions = regions.slice(0, MAX_REGION_FLAGS);
  const hiddenRegions = regions.slice(MAX_REGION_FLAGS);
  return (
    <ResourceListItem className="grid h-12 grid-cols-(--env-columns) items-center gap-4 px-4 text-xs text-gray-12 transition-colors hover:bg-grayA-2">
      <Link
        href={href}
        className="absolute inset-0 z-0"
        aria-label={`Open ${environment.slug} environment`}
      />
      <EnvironmentLabel environment={environment} className="text-gray-12" />
      <BranchLabel branch={branch} />
      <span className="flex min-w-0 items-center gap-1.5 text-gray-11">
        {first ? (
          <>
            <IconEarthOutline18 className="size-3 shrink-0" />
            <span className="truncate font-medium text-gray-12">{first}</span>
            {rest.length > 0 ? (
              <Badge variant="code" className="shrink-0 text-gray-11">
                +{rest.length}
              </Badge>
            ) : null}
          </>
        ) : (
          <span className="text-gray-9">—</span>
        )}
      </span>
      <span className="flex items-center gap-1">
        {regions.length > 0 ? (
          <>
            {shownRegions.map((region) => (
              <InfoTooltip
                key={region.name}
                content={<RegionLine region={region} />}
                position={{ side: "top" }}
                triggerClassName="relative z-10"
              >
                <RegionFlag region={region.name} shape="rect" size="sm" />
                <span className="sr-only">
                  {region.city} ({region.name})
                </span>
              </InfoTooltip>
            ))}
            {hiddenRegions.length > 0 ? (
              <InfoTooltip
                content={
                  <span className="flex flex-col gap-0.5">
                    {hiddenRegions.map((region) => (
                      <RegionLine key={region.name} region={region} />
                    ))}
                  </span>
                }
                position={{ side: "top" }}
                triggerClassName="relative z-10"
              >
                <Badge variant="code" className="shrink-0 text-gray-11" aria-hidden>
                  +{hiddenRegions.length}
                </Badge>
                {hiddenRegions.map((region) => (
                  <span key={region.name} className="sr-only">
                    {region.city} ({region.name})
                  </span>
                ))}
              </InfoTooltip>
            ) : null}
          </>
        ) : (
          <span className="text-gray-9">—</span>
        )}
      </span>
      <IconChevronRightOutline18 className="size-3.5 text-gray-9" />
    </ResourceListItem>
  );
}

function RegionLine({ region }: { region: RegionInfo }) {
  return (
    <span>
      {region.city} <span className="text-gray-8">({region.name})</span>
    </span>
  );
}
