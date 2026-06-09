"use client";

import { NewNavigationBanner } from "@/components/navigation/new-navigation-banner";
import { ProjectsListControls } from "./_components/controls";
import { useDeployGate } from "./_components/hooks/use-deploy-gate";
import { ProjectsList } from "./_components/list";
import { ProjectsListNavigation } from "./navigation";

export default function ProjectsPage() {
  // No plan means no projects to search through, so hide the search controls.
  const { gated } = useDeployGate();
  return (
    <div>
      <ProjectsListNavigation />
      {gated ? null : <ProjectsListControls />}
      <ProjectsList />
      <NewNavigationBanner />
    </div>
  );
}
