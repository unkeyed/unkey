"use client";

import {
  ENVIRONMENT_BADGE_CLASS,
  environmentPillClass,
} from "@/app/(app)/[workspaceSlug]/projects/[projectId]/apps/[appId]/components/environment-badge";
import { useWorkspaceNavigation } from "@/hooks/use-workspace-navigation";
import { collection } from "@/lib/collections";
import { ENVIRONMENT_KIND, type EnvironmentKind } from "@/lib/collections/deploy/environments";
import { withEnvironmentSlug } from "@/lib/navigation/environment-route";
import { routes } from "@/lib/navigation/routes";
import { and, eq, useLiveQuery } from "@tanstack/react-db";
import { IconChevronExpandYOutline12, IconGearOutline18 } from "@unkey/icons";
import { cn } from "cn";
import { useParams, usePathname } from "next/navigation";
import { CRUMB_TRIGGER_CLASS } from "./crumb";
import { CrumbPopover, type CrumbPopoverItem } from "./crumb-popover";

const KIND_ORDER: Record<EnvironmentKind, number> = {
  [ENVIRONMENT_KIND.production]: 0,
  [ENVIRONMENT_KIND.preview]: 1,
};

type EnvironmentSwitcherProps = {
  projectId: string;
  appId: string;
  environmentSlug: string;
};

/**
 * The environment pill and its switcher inside the app crumb. Picking another
 * environment keeps the current sub-page.
 */
export function EnvironmentSwitcher({
  projectId,
  appId,
  environmentSlug,
}: EnvironmentSwitcherProps) {
  const workspace = useWorkspaceNavigation();
  const pathname = usePathname();
  const { deploymentId } = useParams<{ deploymentId?: string }>();
  const environmentsQuery = useLiveQuery(
    (q) =>
      q
        .from({ env: collection.environments })
        .where(({ env }) => and(eq(env.projectId, projectId), eq(env.appId, appId))),
    [projectId, appId],
  );
  const environments = [...(environmentsQuery.data ?? [])].sort(
    (a, b) => KIND_ORDER[a.kind] - KIND_ORDER[b.kind],
  );
  const kind = environments.find((env) => env.slug === environmentSlug)?.kind;
  const scope = { workspaceSlug: workspace.slug, projectId, appId, environmentSlug };

  // A deployment belongs to one environment, so switching from its detail
  // page lands on the other environment's list instead of a 404.
  const hrefFor = (slug: string) =>
    deploymentId
      ? routes.projects.apps.deployments({ ...scope, environmentSlug: slug })
      : withEnvironmentSlug(pathname, appId, slug);

  const items: CrumbPopoverItem[] = environments.map((env) => ({
    id: env.slug,
    label: capitalize(env.slug),
    href: hrefFor(env.slug),
  }));

  return (
    <>
      <span className={cn(ENVIRONMENT_BADGE_CLASS, "h-5 capitalize", environmentPillClass(kind))}>
        {environmentSlug}
      </span>
      <CrumbPopover
        items={items}
        currentId={environmentSlug}
        emptyText="No environments"
        footer={{
          icon: IconGearOutline18,
          label: "Environment settings",
          href: routes.projects.apps.settings(scope),
        }}
      >
        <button type="button" className={CRUMB_TRIGGER_CLASS} aria-label="Switch environment">
          <IconChevronExpandYOutline12 />
        </button>
      </CrumbPopover>
    </>
  );
}

function capitalize(slug: string): string {
  return slug.charAt(0).toUpperCase() + slug.slice(1);
}
