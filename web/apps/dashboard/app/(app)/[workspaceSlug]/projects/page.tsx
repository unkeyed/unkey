"use server";
import { LoadingState } from "@/components/loading-state";
import { NewNavigationBanner } from "@/components/navigation/new-navigation-banner";
import { Suspense } from "react";
import { ProjectsClient } from "./projects-client";

export default async function ProjectsPage() {
  return (
    <Suspense fallback={<LoadingState message="Loading projects..." />}>
      <ProjectsClient />
      <NewNavigationBanner />
    </Suspense>
  );
}
