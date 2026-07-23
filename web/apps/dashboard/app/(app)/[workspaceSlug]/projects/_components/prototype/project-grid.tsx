import { cn } from "@/lib/utils";
import { Cube } from "@unkey/icons";
import type { ReactNode } from "react";
import type { AppMock, ProjectMock } from "./mock-data";
import { DotsIcon, GithubIcon, TerminalIcon } from "./ui";

export function ProjectsEmptyCard({ children }: { children?: ReactNode }) {
  return (
    <div className="flex w-full flex-col items-center gap-3 rounded-lg border border-dashed border-grayA-4 bg-background px-4 py-10 text-center">
      <Cube className="size-6 text-gray-9" />
      <div>
        <h3 className="text-[15px] font-semibold leading-6 text-accent-12">Create a project</h3>
        <p className="text-[13px] leading-5 text-gray-11">
          Build, deploy and scale your API inside Unkey.
        </p>
      </div>
      {children}
    </div>
  );
}

export function ProjectGrid({ projects }: { projects: ProjectMock[] }) {
  return (
    <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
      {projects.map((p) => (
        <ProjectCard key={p.id} project={p} />
      ))}
    </div>
  );
}

function ProjectCard({ project }: { project: ProjectMock }) {
  const overflow = project.appCount - project.apps.length;
  return (
    <div className="relative p-5 flex flex-col justify-between border border-grayA-4 hover:border-grayA-7 rounded-lg gap-6 h-[132px] transition-all duration-300">
      <div className="flex items-start justify-between gap-4">
        <div className="min-w-0">
          <div className="font-medium text-sm text-accent-12 truncate">{project.name}</div>
          <div className="text-xs text-gray-9 mt-0.5 truncate">{project.subtitle}</div>
        </div>
        <button type="button" className="text-gray-9 hover:text-accent-12 shrink-0" aria-label="Project actions">
          <DotsIcon className="size-4" />
        </button>
      </div>
      {project.apps.length === 0 ? (
        <span className="text-xs text-gray-11">No apps yet</span>
      ) : (
        <div className="flex items-center">
          {project.apps.map((app, i) => (
            <AppBlob key={app.id} app={app} first={i === 0} />
          ))}
          {overflow > 0 && (
            <span className="size-7 rounded-full bg-gray-3 ring-2 ring-gray-1 flex items-center justify-center text-[10px] font-medium text-gray-11 -ml-2">
              +{overflow}
            </span>
          )}
        </div>
      )}
    </div>
  );
}

function AppBlob({ app, first }: { app: AppMock; first: boolean }) {
  const Icon = app.source === "github" ? GithubIcon : TerminalIcon;
  return (
    <span
      className={cn(
        "size-7 rounded-full bg-gray-3 ring-2 ring-gray-1 flex items-center justify-center text-gray-12",
        first ? "ml-0" : "-ml-2",
      )}
    >
      <Icon className="size-3.5" />
    </span>
  );
}

