import { useAppHomeHref } from "@/hooks/use-app-home-href";
import { useMeasuredWidth } from "@/hooks/use-measured-width";
import { useWorkspaceNavigation } from "@/hooks/use-workspace-navigation";
import type { ProjectApp } from "@/lib/collections/deploy/project-cards";
import { routes } from "@/lib/navigation/routes";
import { IconCubeOutline18 } from "@unkey/icons";
import { HoverCard, HoverCardContent, HoverCardTrigger, InfoTooltip, Skeleton } from "@unkey/ui";
import Link from "next/link";
import type { Ref } from "react";
import {
  type WarmHandlers,
  useWarmAppPage,
} from "../../[projectId]/apps/[appId]/(overview)/hooks/use-warm-app-page";
import { AppDetailHoverCard, AppRow } from "../apps/app-row";
import { ProjectActionsButton } from "./project-actions";
import { appNoun, projectSummaryOf } from "./project-summary";

type ProjectCardProps = {
  ref: Ref<HTMLDivElement>;
  warm: WarmHandlers;
  name: string;
  projectId: string;
  apps: ProjectApp[];
  isLoading: boolean;
};

const MAX_VISIBLE_APPS = 2;

export function ProjectCard({ ref, warm, name, projectId, apps, isLoading }: ProjectCardProps) {
  const workspace = useWorkspaceNavigation();
  const appHomeHref = useAppHomeHref();
  const { warmApp, cancelWarm } = useWarmAppPage();
  const projectPath = routes.projects.detail({ workspaceSlug: workspace.slug, projectId });
  const hrefFor = (app: ProjectApp) =>
    appHomeHref({ workspaceSlug: workspace.slug, projectId, appId: app.id });

  const { appsByRecency, appsLabel } = projectSummaryOf(apps);
  const visible = appsByRecency.slice(0, MAX_VISIBLE_APPS);
  const rest = appsByRecency.slice(MAX_VISIBLE_APPS);
  const restLabel = `+${rest.length} more ${appNoun(rest.length)}`;

  const { width, measureRef } = useMeasuredWidth<HTMLDivElement>();

  return (
    <div
      ref={ref}
      className="relative p-5 flex flex-col border hover:border-strong bg-raised shadow-xs rounded-lg w-full h-full min-h-[124px] gap-4 group transition-all duration-300 [&_a]:z-10 [&_button]:z-10"
      {...warm}
    >
      <Link
        href={projectPath}
        className="absolute inset-0 z-0"
        aria-label={`View ${name} project`}
      />

      <div className="flex min-h-5 items-center justify-between gap-2.5">
        <span className="flex size-7 shrink-0 items-center justify-center rounded-lg border bg-raised">
          <IconCubeOutline18 className="size-3.5 text-gray-12" />
        </span>
        <InfoTooltip content={name} asChild position={{ align: "start", side: "top" }}>
          <Link
            href={projectPath}
            className="min-w-0 flex-1 truncate text-sm font-medium leading-5 text-gray-12"
          >
            {name}
          </Link>
        </InfoTooltip>
        <div className="relative shrink-0">
          <ProjectActionsButton projectId={projectId} />
        </div>
      </div>

      {isLoading ? (
        <div className="flex flex-col gap-1.5">
          <div className="flex h-6 items-center justify-between gap-2">
            <Skeleton className="h-3.5 w-20" />
            <Skeleton className="h-3.5 w-24" />
          </div>
          <div className="flex h-6 items-center justify-between gap-2">
            <Skeleton className="h-3.5 w-28" />
            <Skeleton className="h-3.5 w-20" />
          </div>
        </div>
      ) : apps.length === 0 ? (
        <span className="text-sm text-gray-9">{appsLabel}</span>
      ) : (
        <div ref={measureRef} className="relative z-10 flex flex-col gap-1.5">
          {visible.map((app) => (
            <AppDetailHoverCard key={app.id} app={app} width={width}>
              <AppRow
                app={app}
                href={hrefFor(app)}
                className="-mx-2 px-2 py-0.5"
                onPointerEnter={() => warmApp(projectId, app.id)}
                onPointerLeave={cancelWarm}
                onFocus={() => warmApp(projectId, app.id)}
              />
            </AppDetailHoverCard>
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
                    <AppRow
                      key={app.id}
                      app={app}
                      href={hrefFor(app)}
                      className="px-2 py-1"
                      onPointerEnter={() => warmApp(projectId, app.id)}
                      onPointerLeave={cancelWarm}
                      onFocus={() => warmApp(projectId, app.id)}
                    />
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
