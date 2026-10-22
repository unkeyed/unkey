"use client";
import { useWorkspaceNavigation } from "@/hooks/use-workspace-navigation";
import { routes } from "@/lib/navigation/routes";
import { redirect } from "next/navigation";

// Root Keys moved to the top level; old settings links keep working.
export default function SettingsRootKeysPage() {
  const workspace = useWorkspaceNavigation();
  redirect(routes.rootKeys.list({ workspaceSlug: workspace.slug }));
}
