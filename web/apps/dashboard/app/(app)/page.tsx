"use client";

import { routes } from "@/lib/navigation/routes";
import { useWorkspace } from "@/providers/workspace-provider";
import type { Route } from "next";
import { useRouter, useSearchParams } from "next/navigation";
import { useEffect } from "react";

export default function AppHomePage() {
  const { workspace } = useWorkspace();
  const router = useRouter();
  const searchParams = useSearchParams();

  useEffect(() => {
    if (workspace) {
      // Forward query (e.g. ?new=true from gateway.new) onto the projects
      // landing so onboarding entrypoints keep working after the redirect.
      const base = routes.projects.list({ workspaceSlug: workspace.slug });
      const query = searchParams?.toString();
      router.push((query ? `${base}?${query}` : base) as Route);
    }
  }, [workspace, router, searchParams]);

  return null; // Layout handles loading states
}
