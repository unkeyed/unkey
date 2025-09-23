"use client";
import { useWorkspaceNavigation } from "@/hooks/use-workspace-navigation";
import { useRouter } from "next/navigation";

export const dynamic = "force-dynamic";

export default function OverviewPage() {
  const router = useRouter();
  const workspace = useWorkspaceNavigation();

  router.replace(`/${workspace.id}/apis`);
  return null;
}
