"use client";

import { useWorkspaceNavigation } from "@/hooks/use-workspace-navigation";
import { collection } from "@/lib/collections";
import { ENVIRONMENT_KIND, type EnvironmentKind } from "@/lib/collections/deploy/environments";
import { withEnvironmentSlug } from "@/lib/navigation/environment-route";
import { routes } from "@/lib/navigation/routes";
import { and, eq, useLiveQuery } from "@tanstack/react-db";
import { cn } from "cn";
import { useParams, usePathname } from "next/navigation";
import { Crumb } from "./crumb";
import type { CrumbPopoverItem } from "./crumb-popover";

type EnvironmentCrumbProps = {
  projectId: string;
  appId: string;
  environmentSlug: string;
};

export function EnvironmentCrumb({ projectId, appId, environmentSlug }: EnvironmentCrumbProps) {
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
  const scope = { workspaceSlug: workspace.slug, projectId, appId };

  // A deployment belongs to one environment, so switching from its detail
  // page lands on the other environment's list instead of a 404.
  const hrefFor = (slug: string) =>
    deploymentId
      ? routes.projects.apps.deployments({ ...scope, environmentSlug: slug })
      : withEnvironmentSlug(pathname, appId, slug);

  const items: CrumbPopoverItem[] = environments.map((env) => ({
    id: env.slug,
    label: env.slug,
    href: hrefFor(env.slug),
    icon: <EnvironmentDot kind={env.kind} />,
  }));

  return (
    <Crumb
      icon={<EnvironmentDot kind={current?.kind ?? ENVIRONMENT_KIND.preview} />}
      label={environmentSlug}
      loading={environmentsQuery.isLoading}
      href={routes.projects.apps.overview({ ...scope, environmentSlug })}
      items={items}
      currentId={environmentSlug}
      emptyText="No environments"
    />
  );
}

function EnvironmentDot({ kind }: { kind: EnvironmentKind }) {
  return (
    <span
      aria-hidden
      className={cn(
        "size-2 shrink-0 rounded-full",
        kind === ENVIRONMENT_KIND.production ? "bg-warning-9" : "bg-info-9",
      )}
    />
  );
}
