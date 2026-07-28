import { DeploymentStatusBadge } from "@/app/(app)/[workspaceSlug]/projects/[projectId]/apps/[appId]/components/deployment-status-badge";
import { routes } from "@/lib/navigation/routes";
import { CodeBranch, Cube } from "@unkey/icons";
import Link from "next/link";
import type { ReactNode } from "react";
import { type DeploymentMock, deploymentsForApps, fmtTimeAgo } from "./deployments-mock";
import type { ProjectMock } from "./mock-data";
import { DotsIcon } from "./ui";

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

export function ProjectGrid({
  projects,
  workspaceSlug,
}: {
  projects: ProjectMock[];
  workspaceSlug: string;
}) {
  return (
    <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
      {projects.map((p) => (
        <ProjectCard key={p.id} project={p} workspaceSlug={workspaceSlug} />
      ))}
    </div>
  );
}

// The most recent deployment across the project's apps — the project shipped
// when any of its apps did.
function latestDeployment(project: ProjectMock): DeploymentMock | undefined {
  return deploymentsForApps(project.apps).sort((a, b) => a.timeAgoMin - b.timeAgoMin)[0];
}

function inventory(project: ProjectMock): string {
  return `${project.appCount} ${project.appCount === 1 ? "app" : "apps"}`;
}

// The latest deployment leads: it is the only thing on the card that changes on
// its own. The status badge carries deployment state, so the title needs no dot,
// and the sha is dropped — the commit message says what shipped.
function ProjectCard({
  project,
  workspaceSlug,
}: {
  project: ProjectMock;
  workspaceSlug: string;
}) {
  const deployment = latestDeployment(project);
  return (
    <div className="relative">
      <Link
        href={routes.projects.detail({ workspaceSlug, projectId: project.id })}
        // A floor, not a fixed height: a card with no deployment shouldn't hold
        // open 148px of nothing, but it still stretches to match its row.
        className="flex h-full min-h-[124px] flex-col justify-between gap-4 rounded-lg border border-grayA-4 p-5 transition-all duration-300 hover:border-grayA-7"
      >
        <div className="min-w-0 pr-8">
          <div className="truncate text-sm font-medium text-accent-12">{project.name}</div>
        </div>
        {deployment ? (
          <>
            <div className="min-w-0">
              <div className="truncate text-[13px] leading-5 text-accent-12">
                {deployment.message}
              </div>
              <div className="mt-1.5 flex min-w-0 items-center gap-2 text-xs text-gray-9">
                <span className="flex min-w-0 items-center gap-1">
                  <CodeBranch className="size-3 shrink-0" />
                  <span className="truncate font-mono">{deployment.branch}</span>
                </span>
                <span className="text-gray-7">·</span>
                <span className="shrink-0">{fmtTimeAgo(deployment.timeAgoMin)}</span>
              </div>
            </div>
            <div className="flex min-w-0 items-center gap-2 text-xs text-gray-9">
              <DeploymentStatusBadge status={deployment.status} />
              <span className="truncate">{inventory(project)}</span>
            </div>
          </>
        ) : (
          <span className="text-[13px] text-gray-9">No deployments yet</span>
        )}
      </Link>
      <button
        type="button"
        className="absolute top-5 right-5 text-gray-9 hover:text-accent-12"
        aria-label="Project actions"
      >
        <DotsIcon className="size-4" />
      </button>
    </div>
  );
}
