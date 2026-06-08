"use client";

import { useParams } from "next/navigation";

/** Current [projectSlug] route param, or undefined when off a project route. */
export function useProjectSlug(): string | undefined {
  const params = useParams();
  return typeof params?.projectSlug === "string" ? params.projectSlug : undefined;
}

/** Current [appSlug] route param, or undefined when off an app route. */
export function useAppSlug(): string | undefined {
  const params = useParams();
  return typeof params?.appSlug === "string" ? params.appSlug : undefined;
}
