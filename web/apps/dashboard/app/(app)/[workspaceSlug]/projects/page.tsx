"use client";

import { NewNavigationBanner } from "@/components/navigation/new-navigation-banner";
import { collection } from "@/lib/collections";
import { useLiveQuery } from "@tanstack/react-db";
import { ProjectsListControls } from "./_components/controls";
import { useDeployGate } from "./_components/hooks/use-deploy-gate";
import { ProjectsList } from "./_components/list";
import { EmptyProjects } from "./_components/list/empty-projects";
import { ProjectsListNavigation } from "./navigation";

export default function ProjectsPage() {
  const projects = useLiveQuery((q) => q.from({ project: collection.projects }));

  // No plan means no projects to search through, so hide the search controls.
  // Hook order: must run unconditionally, before the empty-state early return.
  const { gated } = useDeployGate();

  if (!projects.isLoading && projects.data.length === 0) {
    return <EmptyProjects />;
  }

  return (
    <div>
      <ProjectsListNavigation />
      {gated ? null : <ProjectsListControls />}
      <ProjectsList />
      <NewNavigationBanner />
    </div>
  );
}
