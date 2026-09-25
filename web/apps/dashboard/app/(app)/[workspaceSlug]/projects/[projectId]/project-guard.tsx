"use client";

import { useProject } from "@/hooks/use-project";
import { collection } from "@/lib/collections";
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

  // A failed fetch leaves the query collection ready with no rows and records the
  // error on utils; useLiveQuery's isError only covers exceptions inside sync.
  if (collection.projects.utils.isError) {
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
          <Button
            variant="primary"
            size="md"
            onClick={() => collection.projects.utils.clearError().catch(() => undefined)}
          >
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
