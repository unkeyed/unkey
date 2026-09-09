"use client";
import { useRouter } from "next/navigation";
import { useWorkspaceNavigation } from "@/hooks/use-workspace-navigation";
import { routes } from "@/lib/navigation/routes";

export default function SettingsPage() {
  const workspace = useWorkspaceNavigation();
  const router = useRouter();

  router.replace(routes.settings.general({ workspaceSlug: workspace.slug }));

  return null;
}
