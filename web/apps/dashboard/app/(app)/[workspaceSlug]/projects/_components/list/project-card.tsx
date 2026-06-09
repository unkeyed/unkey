import { useWorkspaceNavigation } from "@/hooks/use-workspace-navigation";
import { Github, Terminal } from "@unkey/icons";
import { HoverCard, HoverCardContent, HoverCardTrigger, InfoTooltip } from "@unkey/ui";
import Link from "next/link";
import type { ReactNode } from "react";

export type ProjectCardApp = {
  id: string;
  name: string;
  source: "github" | "code";
  repository: string | null;
};

type ProjectCardProps = {
  name: string;
  projectId: string;
  /** Total number of apps in the project (may exceed `apps.length`). */
  appCount: number;
  /** Apps to render in the stack, already ordered. Only the first few are shown. */
  apps: ProjectCardApp[];
  actions: ReactNode;
};

const MAX_VISIBLE_APPS = 4;

function AppSourceIcon({
  source,
  className,
}: { source: ProjectCardApp["source"]; className: string }) {
  return source === "github" ? (
    <Github className={className} />
  ) : (
    <Terminal className={className} />
  );
}

function AppLabel({ app, className }: { app: ProjectCardApp; className?: string }) {
  return (
    <span className={className}>
      {app.name}
      {app.repository ? <span className="text-gray-10"> ({app.repository})</span> : null}
    </span>
  );
}

function AppTooltipRow({ app }: { app: ProjectCardApp }) {
  return (
    <div className="flex items-center gap-2 whitespace-nowrap">
      <AppSourceIcon source={app.source} className="size-3.5 shrink-0 text-gray-11" />
      <AppLabel app={app} className="text-xs text-gray-12" />
    </div>
  );
}

export function ProjectCard({ name, projectId, appCount, apps, actions }: ProjectCardProps) {
  const workspace = useWorkspaceNavigation();
  const projectPath = `/${workspace.slug}/projects/${projectId}`;

  return (
    <div className="relative p-5 flex flex-col justify-between border border-grayA-4 hover:border-grayA-7 rounded-2xl w-full h-full gap-6 group transition-all duration-300 [&_a]:z-10 [&_button]:z-10">
      {/* Invisible base clickable layer - covers entire card */}
      <Link
        href={projectPath}
        className="absolute inset-0 z-0"
        aria-label={`View ${name} project`}
      />

      {/*Top Section > Project name + actions*/}
      <div className="flex gap-4 items-start justify-between min-h-5">
        <InfoTooltip content={name} asChild position={{ align: "start", side: "top" }}>
          <Link
            href={projectPath}
            className="font-medium text-sm leading-5 text-accent-12 truncate"
          >
            {name}
          </Link>
        </InfoTooltip>
        <div className="relative shrink-0">{actions}</div>
      </div>

      {/*Bottom Section > App stack + count*/}
      {appCount === 0 ? (
        <span className="text-xs text-gray-9">No apps yet</span>
      ) : (
        <div className="relative z-10 w-fit">
          <AppIconStack apps={apps} appCount={appCount} projectPath={projectPath} />
        </div>
      )}
    </div>
  );
}

function AppIconStack({
  apps,
  appCount,
  projectPath,
}: { apps: ProjectCardApp[]; appCount: number; projectPath: string }) {
  const visible = apps.slice(0, MAX_VISIBLE_APPS);
  const overflow = appCount - visible.length;

  return (
    <div className="flex items-center">
      {visible.map((app) => (
        <InfoTooltip
          key={app.id}
          content={<AppTooltipRow app={app} />}
          asChild
          delayDuration={0}
          position={{ side: "top" }}
        >
          <Link
            href={`${projectPath}/apps/${app.id}/deployments`}
            aria-label={`View ${app.name}`}
            className="size-7 rounded-full bg-gray-3 ring-2 ring-gray-1 flex items-center justify-center -ml-2 first:ml-0 text-gray-12 hover:bg-gray-4 transition-colors"
          >
            <AppSourceIcon source={app.source} className="size-3.5 shrink-0" />
          </Link>
        </InfoTooltip>
      ))}
      {overflow > 0 ? (
        <HoverCard openDelay={0} closeDelay={100}>
          <HoverCardTrigger asChild>
            <Link
              href={projectPath}
              aria-label={`View all ${appCount} apps`}
              className="size-7 rounded-full bg-gray-3 ring-2 ring-gray-1 flex items-center justify-center -ml-2 text-[10px] font-medium text-gray-11 hover:bg-gray-4 transition-colors"
            >
              +{overflow}
            </Link>
          </HoverCardTrigger>
          <HoverCardContent align="start" className="w-56 p-1">
            <div className="flex flex-col">
              {apps.map((app) => (
                <Link
                  key={app.id}
                  href={`${projectPath}/apps/${app.id}/deployments`}
                  className="flex items-center gap-2 rounded-md px-2 py-1.5 text-xs text-gray-12 hover:bg-grayA-3 transition-colors"
                >
                  <AppSourceIcon source={app.source} className="size-3.5 shrink-0 text-gray-11" />
                  <AppLabel app={app} className="truncate" />
                </Link>
              ))}
            </div>
          </HoverCardContent>
        </HoverCard>
      ) : null}
    </div>
  );
}
