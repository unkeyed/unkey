"use client";

import { useProject } from "@/hooks/use-project";
import { collection } from "@/lib/collections";
import { useQuery } from "@tanstack/react-query";
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
import { notFound, useParams } from "next/navigation";
import type { PropsWithChildren } from "react";

export function ProjectGuard({ children }: PropsWithChildren) {
  const { projectId } = useParams<{ projectId: string }>();
  const { project, isLoading } = useProject();
  const projectCheck = useQuery({
    queryKey: ["project-existence", projectId],
    queryFn: async () => {
      await collection.projects.utils.refetch({ throwOnError: true });
      return collection.projects.has(projectId);
    },
    enabled: !isLoading && !project,
    retry: false,
    staleTime: 0,
    cacheTime: 0,
  });

  // A failed fetch leaves the query collection ready with no rows and records the
  // error on utils; useLiveQuery's isError only covers exceptions inside sync.
  if (collection.projects.utils.isError || projectCheck.isError) {
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
          <Button variant="primary" size="md" onClick={() => void projectCheck.refetch()}>
            Retry
          </Button>
        </EmptyStateActions>
      </EmptyState>
    );
  }

  if (
    !isLoading &&
    !project &&
    projectCheck.isFetchedAfterMount &&
    !projectCheck.isFetching &&
    projectCheck.data === false
  ) {
    notFound();
  }

  return children;
}
