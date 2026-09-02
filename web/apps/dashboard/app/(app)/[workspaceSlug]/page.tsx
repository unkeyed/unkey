"use client";
import { useRouter } from "next/navigation";
import { useEffect } from "react";
import { useWorkspaceNavigation } from "@/hooks/use-workspace-navigation";
import { routes } from "@/lib/navigation/routes";

export default function WorkspacePage() {
  const router = useRouter();
  const workspace = useWorkspaceNavigation();

  useEffect(() => {
    router.replace(routes.projects.list({ workspaceSlug: workspace.slug }));
  }, [router, workspace.slug]);

  return null;
}
