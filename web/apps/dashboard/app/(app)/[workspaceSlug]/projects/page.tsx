"use server";
import { LoadingState } from "@/components/loading-state";
import { Suspense } from "react";
import { ProjectsClient } from "./projects-client";

export default async function ProjectsPage() {
  return (
    <Suspense fallback={<LoadingState message="Loading projects..." />}>
      <ProjectsClient />
    </Suspense>
  );
}
