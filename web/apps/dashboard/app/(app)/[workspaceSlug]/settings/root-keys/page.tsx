"use client";
import RootKeysPage from "@/app/(app)/[workspaceSlug]/root-keys/page";
import { useWorkspaceNavigation } from "@/hooks/use-workspace-navigation";
import { useFlag } from "@/lib/flags/provider";
import { routes } from "@/lib/navigation/routes";
import { redirect } from "next/navigation";

// Under projects-first navigation Root Keys lives at the top level; the old
// settings URL keeps working by redirecting there.
export default function SettingsRootKeysPage() {
  const workspace = useWorkspaceNavigation();
  const projectsNav = useFlag("projectsNav");
  if (projectsNav) {
    redirect(routes.rootKeys.list({ workspaceSlug: workspace.slug }));
  }
  return <RootKeysPage />;
}
