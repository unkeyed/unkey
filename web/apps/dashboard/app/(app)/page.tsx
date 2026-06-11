"use client";

import { routes } from "@/lib/navigation/routes";
import { useWorkspace } from "@/providers/workspace-provider";
import { useRouter } from "next/navigation";
import { useEffect } from "react";

export default function AppHomePage() {
  const { workspace } = useWorkspace();
  const router = useRouter();

  useEffect(() => {
    if (workspace) {
      router.push(routes.projects.list({ workspaceSlug: workspace.slug }));
    }
  }, [workspace, router]);

  return null; // Layout handles loading states
}
