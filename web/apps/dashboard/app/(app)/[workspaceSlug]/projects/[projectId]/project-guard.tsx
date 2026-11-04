"use client";

import { LoadError } from "@/components/load-error";
import { fromQuery } from "@/lib/loadable/loadable";
import { useProject } from "@/lib/queries/projects";
import { match } from "@unkey/match";
import { notFound, useParams } from "next/navigation";
import type { PropsWithChildren } from "react";

export function ProjectGuard({ children }: PropsWithChildren) {
  const { projectId } = useParams<{ projectId: string }>();
  return match(fromQuery(useProject(projectId)))
    .with({ status: "loading" }, () => children)
    .with({ status: "failed" }, ({ error, retry }) => (
      <LoadError title="Could not load this project" error={error} onRetry={retry} />
    ))
    .with({ status: "ready" }, ({ data }) => {
      // The list holds every project in the workspace, so a missing one does not
      // exist or is not accessible
      if (!data) {
        notFound();
      }
      return children;
    })
    .exhaustive();
}
