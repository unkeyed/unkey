"use client";

import { NewNavigationBanner } from "@/components/navigation/new-navigation-banner";
import { ProjectsListControls } from "./_components/controls";
import { ProjectsList } from "./_components/list";
import { ProjectsListNavigation } from "./navigation";

export default function ProjectsPage() {
  return (
    <div>
      <ProjectsListNavigation />
      <ProjectsListControls />
      <ProjectsList />
      <NewNavigationBanner />
    </div>
  );
}
