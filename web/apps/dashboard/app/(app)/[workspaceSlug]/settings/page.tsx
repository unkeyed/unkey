"use client";
import { useWorkspaceNavigation } from "@/hooks/use-workspace-navigation";
import { routes } from "@/lib/navigation/routes";
import { useRouter } from "next/navigation";

export default function SettingsPage() {
  const workspace = useWorkspaceNavigation();
  const router = useRouter();

  router.replace(routes.settings.general({ workspaceSlug: workspace.slug }));

  return null;
}
