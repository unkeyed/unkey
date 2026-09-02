"use client";
import { useRouter } from "next/navigation";
import { useWorkspaceNavigation } from "@/hooks/use-workspace-navigation";
import { routes } from "@/lib/navigation/routes";

export const dynamic = "force-dynamic";

export default function OverviewPage() {
  const router = useRouter();
  const workspace = useWorkspaceNavigation();

  router.replace(routes.apis.list({ workspaceSlug: workspace.slug }));
  return null;
}
