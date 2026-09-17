"use client";

import { useProject } from "@/hooks/use-project";
import { notFound } from "next/navigation";
import type { PropsWithChildren } from "react";

export function ProjectGuard({ children }: PropsWithChildren) {
  const { project, isLoading } = useProject();

  // The projects collection holds every project in the workspace, so once it has
  // finished loading an absent project means it does not exist (or is inaccessible).
  if (!isLoading && !project) {
    notFound();
  }

  return children;
}
