"use client";

import { useParams } from "next/navigation";

export function useProjectScope(): { projectId?: string } {
  const { projectId } = useParams<{ projectId?: string }>();
  return { projectId };
}
