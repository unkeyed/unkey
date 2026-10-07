"use client";

import { useWorkspaceNavigation } from "@/hooks/use-workspace-navigation";
import { collection } from "@/lib/collections";
import type { Environment } from "@/lib/collections/deploy/environments";
import { useCollectionLoad } from "@/lib/collections/use-collection-load";
import { routes } from "@/lib/navigation/routes";
import { regionInfo } from "@/lib/regions";
import { and, eq, useLiveQuery } from "@tanstack/react-db";
import {
  IconChevronRightOutline18,
  IconCodeBranchOutline18,
  IconEarthOutline18,
} from "@unkey/icons";
import { match } from "@unkey/match";
import { ResourceListBody, ResourceListContent, ResourceListItem } from "@unkey/ui";
import { cn } from "cn";
import Link from "next/link";
import { EnvironmentLabel } from "../../../components/environment-label";
import { LoadError } from "../../../components/load-error";
import { useProjectData } from "../../data-provider";
import { environmentDomains } from "../../domains/model";
import { useBuildSource, useEnvironmentsProductionFirst } from "../hooks/use-build-source";

const COLUMNS =
  "grid grid-cols-[minmax(6.5rem,1fr)_minmax(0,1.1fr)_minmax(0,1.6fr)_minmax(0,1.2fr)_16px] items-center gap-4";
const PILL =
  "inline-block max-w-full truncate rounded-sm bg-gray-3 px-1 align-middle font-mono text-2xs leading-4 ring-1 ring-grayA-4";

export const PREVIEW_BRANCHES = "All unassigned branches";

export function EnvironmentsTable() {
  const environments = useEnvironmentsProductionFirst();
  const { projectId, appId, hasRepository, defaultBranch } = useBuildSource();
  const { domains, customDomains, isEnvironmentsLoading } = useProjectData();
  const environmentsLoad = useCollectionLoad(collection.environments.utils);
  const workspace = useWorkspaceNavigation();
  const { data: settings } = useLiveQuery(
    (q) =>
      q
        .from({ s: collection.environmentSettings })
        .where(({ s }) => and(eq(s.projectId, projectId), eq(s.appId, appId))),
    [projectId, appId],
  );

  if (environmentsLoad.failed && !isEnvironmentsLoading) {
    return <LoadError title="Could not load environments" onRetry={environmentsLoad.retry} />;
  }

  return (
    <ResourceListContent>
      <div
        className={`${COLUMNS} border-b bg-table-header px-4 py-[7px] text-xs font-medium text-gray-12`}
      >
        <span>Environment</span>
        <span>Branch</span>
        <span>Domains</span>
        <span>Regions</span>
        <span />
      </div>
      <ResourceListBody>
        {environments.map((environment) => (
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
            regions={(settings.find((s) => s.environmentId === environment.id)?.regions ?? [])
              .map((r) => regionInfo(r.name).city)
              .join(", ")}
          />
        ))}
      </ResourceListBody>
    </ResourceListContent>
  );
}

type Branch = { kind: "default"; name: string } | { kind: "unassigned" } | { kind: "none" };

export function branchFor(
  environment: Pick<Environment, "kind">,
  hasRepository: boolean,
  defaultBranch: string | null,
): Branch {
  if (!hasRepository) {
    return { kind: "none" };
  }
  if (environment.kind === "preview") {
    return { kind: "unassigned" };
  }
  return defaultBranch ? { kind: "default", name: defaultBranch } : { kind: "none" };
}

export function BranchLabel({ branch, className }: { branch: Branch; className?: string }) {
  return (
    <span className={cn("flex min-w-0 items-center gap-1.5 text-xs text-gray-11", className)}>
      {match(branch)
        .with({ kind: "default" }, ({ name }) => (
          <>
            <IconCodeBranchOutline18 className="size-3 shrink-0" />
            <span className={`${PILL} text-gray-12`}>{name}</span>
          </>
        ))
        .with({ kind: "unassigned" }, () => (
          <>
            <IconCodeBranchOutline18 className="size-3 shrink-0" />
            <span className="truncate">{PREVIEW_BRANCHES}</span>
          </>
        ))
        .with({ kind: "none" }, () => <span className="text-gray-9">—</span>)
        .exhaustive()}
    </span>
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
  regions: string;
}) {
  const [first, ...rest] = domains;
  return (
    <ResourceListItem
      className={`${COLUMNS} h-12 px-4 text-xs text-gray-12 transition-colors hover:bg-grayA-2`}
    >
      <Link
        href={href}
        replace
        scroll={false}
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
              <span className={`${PILL} shrink-0 text-gray-11`}>+{rest.length}</span>
            ) : null}
          </>
        ) : (
          <span className="text-gray-9">—</span>
        )}
      </span>
      <span className="truncate text-gray-11">
        {regions || <span className="text-gray-9">—</span>}
      </span>
      <IconChevronRightOutline18 className="size-3.5 text-gray-9" />
    </ResourceListItem>
  );
}
