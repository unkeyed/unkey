"use client";
import { useWorkspaceWithRedirect } from "@/hooks/use-workspace-with-redirect";
import { useRouter } from "next/navigation";

export default function SettingsPage() {
  const { workspace } = useWorkspaceWithRedirect();
  const router = useRouter();

  router.replace(`/${workspace.slug}/settings/general`);

  return null;
}
