"use client";

import { collection } from "@/lib/collections";
import { useFlag } from "@/lib/flags/provider";
import { eq, useLiveQuery } from "@tanstack/react-db";

export function useVisibleProjects() {
  const projectsNav = useFlag("projectsNav");

  return useLiveQuery(
    (q) => {
      const all = q
        .from({ project: collection.projects })
        .orderBy(({ project }) => project.createdAt, "desc");
      return projectsNav ? all : all.where(({ project }) => eq(project.isDefault, false));
    },
    [projectsNav],
  );
}
