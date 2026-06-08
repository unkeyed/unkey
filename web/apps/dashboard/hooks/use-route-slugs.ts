"use client";

import { useParams } from "next/navigation";

function useStringParam(name: string): string | undefined {
  const params = useParams();
  const value = params?.[name];
  return typeof value === "string" ? value : undefined;
}

/** Current [projectSlug] route param, or undefined when off a project route. */
export function useProjectSlug(): string | undefined {
  return useStringParam("projectSlug");
}

/** Current [appSlug] route param, or undefined when off an app route. */
export function useAppSlug(): string | undefined {
  return useStringParam("appSlug");
}
