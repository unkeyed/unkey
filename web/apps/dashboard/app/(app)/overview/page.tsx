"use client";
import { useWorkspaceNavigation } from "@/hooks/use-workspace-navigation";
import { routes } from "@/lib/navigation/routes";
import { useRouter } from "next/navigation";

export const dynamic = "force-dynamic";

export default function OverviewPage() {
  const router = useRouter();
  const workspace = useWorkspaceNavigation();

  router.replace(routes.apis.list({ workspaceSlug: workspace.slug }));
  return null;
}
