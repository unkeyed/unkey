"use client";

import { NewNavigationBanner } from "@/components/navigation/new-navigation-banner";
import { collection } from "@/lib/collections";
import { useLiveQuery } from "@tanstack/react-db";
import { ProjectsListControls } from "./_components/controls";
import { ProjectsList } from "./_components/list";
import { EmptyProjects } from "./_components/list/empty-projects";
import { ProjectsListNavigation } from "./navigation";

export default function ProjectsPage() {
  const projects = useLiveQuery((q) => q.from({ project: collection.projects }));

  // A workspace with no projects yet gets a special full-bleed empty state: no list header,
  // controls, or navigation banner, so the "create your first project" moment stands alone.
  if (!projects.isLoading && projects.data.length === 0) {
    return <EmptyProjects />;
  }

  return (
    <div>
      <ProjectsListNavigation />
      <ProjectsListControls />
      <ProjectsList />
      <NewNavigationBanner />
    </div>
  );
}
