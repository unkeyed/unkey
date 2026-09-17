"use client";

import { collection } from "@/lib/collections";
import type { Project } from "@/lib/collections/deploy/projects";
import { eq, useLiveQuery } from "@tanstack/react-db";
import { useParams } from "next/navigation";

export function useProject(): { project: Project | undefined; isLoading: boolean } {
  const { projectId } = useParams<{ projectId: string }>();

  const query = useLiveQuery(
    (q) =>
      q.from({ project: collection.projects }).where(({ project }) => eq(project.id, projectId)),
    [projectId],
  );

  return { project: query.data?.at(0), isLoading: query.isLoading };
}
