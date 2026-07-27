"use client";

import { GithubIcon } from "@/app/(app)/[workspaceSlug]/projects/_components/prototype/ui";
import { routes } from "@/lib/navigation/routes";
import { cn } from "@/lib/utils";
import { Button } from "@unkey/ui";
import Link from "next/link";
import { useMemo } from "react";
import { CopyAgentPromptButton } from "./agent-prompt";
import { deploymentHistoryForApp } from "./deployments-mock";
import {
  ActivityRailCard,
  DeploymentsPanel,
  KeyspacesRailCard,
  RatelimitsRailCard,
  TrafficPanel,
  UsageRailCard,
} from "./option-hero-panels";
import type { OverviewProjectData } from "./overview-data";
import { activityForProject } from "./overview-mocks";

// "Deploy hero" — Zuplo-inspired: deployments read as the project's main
// surface. A brand-new project gets the full centered hero pitching "connect
// a repo"; everyone else gets the same traffic + deployments spine with a
// resource rail alongside it, so the page doesn't reshuffle once an app shows up.
export function OptionHero({ data }: { data: OverviewProjectData }) {
  const { project, keyspaces, ratelimits, usage, deployments, workspaceSlug, scenario } = data;

  const recentDeployments = useMemo(() => {
    const merged = project.apps.flatMap((app) => deploymentHistoryForApp(app, 5));
    merged.sort((a, b) => a.timeAgoMin - b.timeAgoMin);
    return merged.slice(0, 7);
  }, [project.apps]);

  const activity = useMemo(
    () => activityForProject(project, keyspaces, ratelimits),
    [project, keyspaces, ratelimits],
  );

  const isNew = scenario === "new" && deployments.length === 0 && keyspaces.length === 0;

  return (
    <div className="flex flex-col gap-5">
      <ProjectHeader workspaceSlug={workspaceSlug} project={project} isNew={isNew} />

      {isNew ? (
        <NewProjectHero
          workspaceSlug={workspaceSlug}
          projectId={project.id}
          keyspaces={keyspaces}
          ratelimits={ratelimits}
        />
      ) : (
        <div className="flex flex-col-reverse lg:flex-row items-start gap-5">
          <div className="flex min-w-0 w-full flex-1 flex-col gap-5">
            <TrafficPanel projectId={project.id} keyspaces={keyspaces} />
            <DeploymentsPanel
              workspaceSlug={workspaceSlug}
              projectId={project.id}
              deployments={recentDeployments}
            />
          </div>
          <div className="flex w-full shrink-0 flex-col gap-5 lg:w-[340px]">
            <UsageRailCard workspaceSlug={workspaceSlug} usage={usage} />
            <KeyspacesRailCard workspaceSlug={workspaceSlug} keyspaces={keyspaces} />
            <RatelimitsRailCard workspaceSlug={workspaceSlug} ratelimits={ratelimits} />
            <ActivityRailCard activity={activity} />
          </div>
        </div>
      )}
    </div>
  );
}

function ProjectHeader({
  workspaceSlug,
  project,
  isNew,
}: {
  workspaceSlug: string;
  project: OverviewProjectData["project"];
  isNew: boolean;
}) {
  return (
    <div className="flex items-start justify-between gap-4">
      <div className="min-w-0">
        <h1 className="text-[22px] font-semibold tracking-tight leading-tight text-accent-12 truncate">
          {project.name}
        </h1>
        <div className="mt-1 flex items-center gap-1.5 text-[13px] text-gray-9">
          <span className="font-mono">{project.id}</span>
          {project.appCount > 0 && (
            <>
              <span>·</span>
              <span>
                {project.appCount} {project.appCount === 1 ? "app" : "apps"}
              </span>
            </>
          )}
        </div>
      </div>
      {!isNew && (
        <Button
          size="md"
          variant="primary"
          render={
            <Link href={routes.projects.apps.new({ workspaceSlug, projectId: project.id })} />
          }
        >
          Deploy
        </Button>
      )}
    </div>
  );
}

const GETTING_STARTED_STEPS = (data: {
  hasDeployment: boolean;
  hasKeyspace: boolean;
  hasRatelimit: boolean;
}) => [
  { label: "Deploy your app", done: data.hasDeployment },
  { label: "Create a keyspace", done: data.hasKeyspace },
  { label: "Add a ratelimit", done: data.hasRatelimit },
  { label: "Invite your team", done: false },
];

// The onboarding moment: nothing exists yet, so the whole page collapses to a
// single centered column pitching the one thing that unblocks everything
// else — connecting source control (or handing the agent the setup prompt).
function NewProjectHero({
  workspaceSlug,
  projectId,
  keyspaces,
  ratelimits,
}: {
  workspaceSlug: string;
  projectId: string;
  keyspaces: OverviewProjectData["keyspaces"];
  ratelimits: OverviewProjectData["ratelimits"];
}) {
  const steps = GETTING_STARTED_STEPS({
    hasDeployment: false,
    hasKeyspace: keyspaces.length > 0,
    hasRatelimit: ratelimits.length > 0,
  });

  return (
    <div className="flex w-full flex-col gap-5">
      <div className="flex flex-col items-center gap-4 rounded-lg border border-grayA-4 px-6 py-16 text-center">
        <span className="flex size-11 items-center justify-center rounded-lg bg-gray-3 text-gray-11">
          <GithubIcon className="size-5" />
        </span>
        <div>
          <div className="text-[15px] font-semibold text-accent-12">
            Connect source control to deploy
          </div>
          <p className="mt-1 max-w-sm text-[13px] text-gray-9">
            Push code and we&apos;ll build, deploy, and keep your app live automatically.
          </p>
        </div>
        <div className="flex items-center gap-2">
          <Button
            size="md"
            variant="primary"
            render={<Link href={routes.projects.apps.new({ workspaceSlug, projectId })} />}
          >
            Connect GitHub
          </Button>
          <CopyAgentPromptButton />
        </div>
      </div>

      <div className="rounded-lg border border-grayA-4 overflow-hidden">
        <div className="border-b border-grayA-4 px-4 py-3">
          <span className="text-sm font-medium text-accent-12">Getting started</span>
        </div>
        <div className="flex flex-col md:flex-row md:divide-x md:divide-grayA-4">
          {steps.map((step, i) => (
            <div key={step.label} className="flex flex-1 items-center gap-2.5 px-4 py-3">
              <span
                className={cn(
                  "flex size-5 shrink-0 items-center justify-center rounded-full text-[11px] font-medium",
                  step.done ? "bg-success-9 text-white" : "bg-gray-3 text-gray-9",
                )}
              >
                {step.done ? "✓" : i + 1}
              </span>
              <span
                className={cn(
                  "text-[13px]",
                  step.done ? "text-gray-9 line-through" : "text-gray-11",
                )}
              >
                {step.label}
              </span>
            </div>
          ))}
        </div>
      </div>

      <div className="flex items-center justify-center gap-3 text-[13px] text-gray-9">
        <span>Need help?</span>
        <Link href="https://unkey.com/docs" target="_blank" className="hover:text-accent-12">
          Docs
        </Link>
        <span>·</span>
        <Link href="mailto:support@unkey.com" className="hover:text-accent-12">
          Talk to us
        </Link>
        <span>·</span>
        <Link href="https://unkey.com/discord" target="_blank" className="hover:text-accent-12">
          Discord
        </Link>
      </div>
    </div>
  );
}
