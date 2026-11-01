"use client";

import { type Project, type QueryResult, useProjects } from "@/lib/deploy-data";
import { useFlag } from "@/lib/flags/provider";

export function useVisibleProjects(): QueryResult<Project[]> {
  const projectsNav = useFlag("projectsNav");
  const projects = useProjects();
  return projectsNav
    ? projects
    : { ...projects, data: projects.data.filter((project) => !project.isDefault) };
}
