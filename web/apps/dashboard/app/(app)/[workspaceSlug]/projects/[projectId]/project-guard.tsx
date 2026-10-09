"use client";

import { LoadError } from "@/components/load-error";
import { useProject } from "@/hooks/use-project";
import { collection } from "@/lib/collections";
import { useCollectionLoad } from "@/lib/collections/use-collection-load";
import { notFound } from "next/navigation";
import type { PropsWithChildren } from "react";

export function ProjectGuard({ children }: PropsWithChildren) {
  const { project, isLoading } = useProject();
  const projectsLoad = useCollectionLoad(collection.projects.utils);

  if (projectsLoad.failed) {
    return <LoadError title="Could not load this project" onRetry={projectsLoad.retry} />;
  }

  // The projects collection holds every project in the workspace, so once it has
  // finished loading an absent project means it does not exist (or is inaccessible).
  if (!isLoading && !project) {
    notFound();
  }

  return children;
}
