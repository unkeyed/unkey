"use client";

import { useSearchParams } from "next/navigation";
import { Onboarding } from "./(onboarding)";
import { ProjectsListControls } from "./_components/controls";
import { ProjectsList } from "./_components/list";
import { ProjectsListNavigation } from "./navigation";

export function ProjectsClient() {
  const searchParams = useSearchParams();
  const hasOnboarding = searchParams.get("onboarding");

  return hasOnboarding ? (
    <Onboarding />
  ) : (
    <div>
      <ProjectsListNavigation />
      <ProjectsListControls />
      <ProjectsList />
    </div>
  );
}
