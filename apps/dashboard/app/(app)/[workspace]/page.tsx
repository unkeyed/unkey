"use client";

import { useWorkspaceWithRedirect } from "@/hooks/use-workspace-with-redirect";
import { useRouter } from "next/navigation";

export default function WorkspacePage() {
  const router = useRouter();
  const { workspace } = useWorkspaceWithRedirect();

  router.replace(`/${workspace.slug}/apis`);

  return null;
}
