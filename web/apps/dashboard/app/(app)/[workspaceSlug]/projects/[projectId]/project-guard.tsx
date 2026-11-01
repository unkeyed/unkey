"use client";

import { LoadError } from "@/components/load-error";
import { useProject } from "@/lib/deploy-data";
import { notFound, useParams } from "next/navigation";
import type { PropsWithChildren } from "react";

export function ProjectGuard({ children }: PropsWithChildren) {
  const { projectId } = useParams<{ projectId?: string }>();
  const { data: project, isLoading, isError, refetch } = useProject(projectId);

  if (isError) {
    return <LoadError title="Could not load this project" onRetry={refetch} />;
  }

  // The projects collection holds every project in the workspace, so once it has
  // finished loading an absent project means it does not exist (or is inaccessible).
  if (!isLoading && !project) {
    notFound();
  }

  return children;
}
