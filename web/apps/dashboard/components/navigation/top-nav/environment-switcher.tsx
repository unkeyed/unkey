"use client";

import { useWorkspaceNavigation } from "@/hooks/use-workspace-navigation";
import { collection } from "@/lib/collections";
import { ENVIRONMENT_KIND, type EnvironmentKind } from "@/lib/collections/deploy/environments";
import { withEnvironmentSlug } from "@/lib/navigation/environment-route";
import { routes } from "@/lib/navigation/routes";
import { and, eq, useLiveQuery } from "@tanstack/react-db";
import { cn } from "cn";
import { useParams, usePathname } from "next/navigation";
import { CrumbPopover, type CrumbPopoverItem } from "./crumb-popover";

type EnvironmentSwitcherProps = {
  projectId: string;
  appId: string;
  environmentSlug: string;
};

/**
 * The environment pill inside the app crumb. Clicking it lists the app's
 * environments; picking one keeps the current sub-page.
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
  const environments = environmentsQuery.data ?? [];
  const current = environments.find((env) => env.slug === environmentSlug);

  // A deployment belongs to one environment, so switching from its detail
  // page lands on the other environment's list instead of a 404.
  const hrefFor = (slug: string) =>
    deploymentId
      ? routes.projects.apps.deployments({
          workspaceSlug: workspace.slug,
          projectId,
          appId,
          environmentSlug: slug,
        })
      : withEnvironmentSlug(pathname, appId, slug);

  const items: CrumbPopoverItem[] = environments.map((env) => ({
    id: env.slug,
    label: env.slug,
    href: hrefFor(env.slug),
  }));

  return (
    <CrumbPopover items={items} currentId={environmentSlug} emptyText="No environments">
      <button
        type="button"
        aria-label={`Switch environment (${environmentSlug})`}
        className={environmentBadgeClass(current?.kind ?? ENVIRONMENT_KIND.preview)}
      >
        {environmentSlug}
      </button>
    </CrumbPopover>
  );
}

const BADGE_CLASS =
  "inline-flex items-center justify-center gap-1 whitespace-nowrap rounded-full border px-[5.5px] py-[3px] font-medium text-[9px] uppercase leading-none tracking-[0.07em] transition-colors";

function environmentBadgeClass(kind: EnvironmentKind): string {
  return cn(
    BADGE_CLASS,
    kind === ENVIRONMENT_KIND.production
      ? "border-warningA-6 bg-warningA-3 text-warningA-11 hover:bg-warningA-4"
      : "border-successA-6 bg-successA-3 text-successA-11 hover:bg-successA-4",
  );
}
