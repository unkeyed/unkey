"use client";
import { useWorkspaceNavigation } from "@/hooks/use-workspace-navigation";
import { projectsPath } from "@/lib/navigation/routes";
import { useRouter } from "next/navigation";
import { useEffect } from "react";

export default function WorkspacePage() {
  const router = useRouter();
  const workspace = useWorkspaceNavigation();

  useEffect(() => {
    router.replace(projectsPath({ workspaceSlug: workspace.slug }));
  }, [router, workspace.slug]);

  return null;
}
