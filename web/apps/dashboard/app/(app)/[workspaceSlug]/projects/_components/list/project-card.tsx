import { useAppHomeHref } from "@/hooks/use-app-home-href";
import { useMeasuredWidth } from "@/hooks/use-measured-width";
import { useWorkspaceNavigation } from "@/hooks/use-workspace-navigation";
import type { ProjectApp } from "@/lib/collections/deploy/projects";
import { routes } from "@/lib/navigation/routes";
import { IconCubeOutline18 } from "@unkey/icons";
import { HoverCard, HoverCardContent, HoverCardTrigger, InfoTooltip } from "@unkey/ui";
import Link from "next/link";
import type { ReactNode } from "react";
import { AppDetailTooltip, AppRow } from "../apps/app-row";

type ProjectCardProps = {
  name: string;
  projectId: string;
  apps: ProjectApp[];
  actions: ReactNode;
};

const MAX_VISIBLE_APPS = 2;

function byRecency(apps: ProjectApp[]): ProjectApp[] {
  return apps.toSorted(
    (a, b) => (b.headlineDeployment?.deployedAt ?? 0) - (a.headlineDeployment?.deployedAt ?? 0),
  );
}

export function ProjectCard({ name, projectId, apps, actions }: ProjectCardProps) {
  const workspace = useWorkspaceNavigation();
  const appHomeHref = useAppHomeHref();
  const projectPath = routes.projects.detail({
    workspaceSlug: workspace.slug,
    projectId,
  });
  const hrefFor = (app: ProjectApp) =>
    appHomeHref({ workspaceSlug: workspace.slug, projectId, appId: app.id });

  const ordered = byRecency(apps);
  const visible = ordered.slice(0, MAX_VISIBLE_APPS);
  const rest = ordered.slice(MAX_VISIBLE_APPS);
  const restLabel = `+${rest.length} more ${rest.length === 1 ? "app" : "apps"}`;

  const { width, measureRef } = useMeasuredWidth<HTMLDivElement>();

  return (
    <div className="relative p-5 flex flex-col border border-grayA-4 hover:border-grayA-7 rounded-lg w-full h-full min-h-[124px] gap-4 group transition-all duration-300 [&_a]:z-10 [&_button]:z-10">
      <Link
        href={projectPath}
        className="absolute inset-0 z-0"
        aria-label={`View ${name} project`}
      />

      <div className="flex min-h-5 items-center justify-between gap-2.5">
        <span className="flex size-7 shrink-0 items-center justify-center rounded-lg border border-grayA-4 bg-white dark:bg-black">
          <IconCubeOutline18 className="size-3.5 text-gray-12" />
        </span>
        <InfoTooltip content={name} asChild position={{ align: "start", side: "top" }}>
          <Link
            href={projectPath}
            className="min-w-0 flex-1 truncate text-sm font-medium leading-5 text-accent-12"
          >
            {name}
          </Link>
        </InfoTooltip>
        <div className="relative shrink-0">{actions}</div>
      </div>

      {apps.length === 0 ? (
        <span className="text-[13px] text-gray-9">No apps yet</span>
      ) : (
        <div ref={measureRef} className="relative z-10 flex flex-col gap-1.5">
          {visible.map((app) => (
            <AppDetailTooltip key={app.id} app={app} width={width}>
              <AppRow app={app} href={hrefFor(app)} className="-mx-2 px-2 py-0.5" />
            </AppDetailTooltip>
          ))}
          {rest.length > 0 ? (
            <HoverCard>
              <HoverCardTrigger
                delay={100}
                closeDelay={100}
                render={
                  <Link
                    href={projectPath}
                    aria-label={`${restLabel}, view all ${apps.length}`}
                    className="-mx-2 w-fit rounded-md px-2 py-0.5 text-xs text-gray-9 transition-colors hover:bg-grayA-3 hover:text-gray-11"
                  >
                    {restLabel}
                  </Link>
                }
              />
              <HoverCardContent side="right" align="center" style={{ width }} className="p-1">
                <div className="flex max-h-64 flex-col overflow-y-auto">
                  {rest.map((app) => (
                    <AppRow key={app.id} app={app} href={hrefFor(app)} className="px-2 py-1" />
                  ))}
                </div>
              </HoverCardContent>
            </HoverCard>
          ) : null}
        </div>
      )}
    </div>
  );
}
