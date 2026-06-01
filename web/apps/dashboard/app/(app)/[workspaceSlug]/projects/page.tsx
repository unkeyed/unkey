"use client"

import { ProjectsListControls } from "./_components/controls"
import { ProjectsList } from "./_components/list"
import { ProjectsListNavigation } from "./navigation"
import { NewNavigationBanner } from "@/components/navigation/new-navigation-banner";

export default function ProjectsPage() {
  return <div>
    <ProjectsListNavigation />
    <ProjectsListControls />
    <ProjectsList />
      <NewNavigationBanner />
  </div>
}
