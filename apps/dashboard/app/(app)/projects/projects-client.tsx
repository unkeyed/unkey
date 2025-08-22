"use client";

import { ProjectsListControls } from "./_components/controls";
import { ProjectsList } from "./_components/list";
import { ProjectsListNavigation } from "./navigation";

export function ProjectsClient() {
  return (
    <div>
      <ProjectsListNavigation />
      <ProjectsListControls />
      <ProjectsList />
    </div>
  );
}
