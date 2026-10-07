"use client";

import { useProject } from "@/hooks/use-project";
import { collection } from "@/lib/collections";
import { useCollectionLoad } from "@/lib/collections/use-collection-load";
import { IconCubeOutline18 } from "@unkey/icons";
import {
  Button,
  EmptyState,
  EmptyStateActions,
  EmptyStateDescription,
  EmptyStateHeader,
  EmptyStateIcon,
  EmptyStateTitle,
} from "@unkey/ui";
import { notFound } from "next/navigation";
import type { PropsWithChildren } from "react";

export function ProjectGuard({ children }: PropsWithChildren) {
  const { project, isLoading } = useProject();
  const projectsLoad = useCollectionLoad(collection.projects.utils);

  if (projectsLoad.failed) {
    return (
      <EmptyState>
        <EmptyStateIcon>
          <IconCubeOutline18 />
        </EmptyStateIcon>
        <EmptyStateHeader>
          <EmptyStateTitle>Could not load this project</EmptyStateTitle>
          <EmptyStateDescription>The project list failed to load. Try again.</EmptyStateDescription>
        </EmptyStateHeader>
        <EmptyStateActions>
          <Button variant="primary" size="md" onClick={projectsLoad.retry}>
            Retry
          </Button>
        </EmptyStateActions>
      </EmptyState>
    );
  }

  // The projects collection holds every project in the workspace, so once it has
  // finished loading an absent project means it does not exist (or is inaccessible).
  if (!isLoading && !project) {
    notFound();
  }

  return children;
}
