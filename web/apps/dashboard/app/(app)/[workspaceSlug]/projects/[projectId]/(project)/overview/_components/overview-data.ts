"use client";

import type {
  KeyspaceStat,
  ProjectMock,
  RatelimitStat,
  Scenario,
  UsageStat,
} from "@/app/(app)/[workspaceSlug]/projects/_components/prototype/mock-data";
import { useScenario } from "@/app/(app)/[workspaceSlug]/projects/_components/prototype/scenario";
import { usePrototypeWorlds } from "@/app/(app)/[workspaceSlug]/projects/_components/prototype/store";
import { useWorkspaceNavigation } from "@/hooks/use-workspace-navigation";
import { useParams } from "next/navigation";
import { type DeploymentMock, deploymentsForApps } from "./deployments-mock";

export type OverviewProjectData = {
  workspaceSlug: string;
  scenario: Scenario;
  project: ProjectMock;
  keyspaces: KeyspaceStat[];
  ratelimits: RatelimitStat[];
  usage: UsageStat;
  deployments: DeploymentMock[];
};

// Only reachable if the "new" scenario's world (zero projects) is active while
// deep-linked straight to an overview URL — every other path always resolves a
// real project from the current scenario's world.
const FALLBACK_PROJECT: ProjectMock = {
  id: "p_new",
  name: "your-project",
  subtitle: "Created just now",
  apps: [],
  appCount: 0,
};

// Scoped to whichever project the URL names (falling back to the scenario's
// first project), so the overview page reflects the same shared scenario/world
// state as the projects list rail instead of inventing its own.
export function useOverviewProjectData(): OverviewProjectData {
  const workspace = useWorkspaceNavigation();
  const params = useParams<{ projectId: string }>();
  const { scenario } = useScenario();
  const { worlds } = usePrototypeWorlds();
  const world = worlds[scenario];
  const project =
    world.projects.find((p) => p.id === params.projectId) ?? world.projects[0] ?? FALLBACK_PROJECT;

  return {
    workspaceSlug: workspace.slug,
    scenario,
    project,
    keyspaces: world.keyspaces.filter((ks) => ks.projectId === project.id),
    ratelimits: world.ratelimits.filter((rl) => rl.projectId === project.id),
    usage: world.usage,
    deployments: deploymentsForApps(project.apps),
  };
}
