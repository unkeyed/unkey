"use client";

import { useWorkspace } from "@/providers/workspace-provider";
import { useRouter } from "next/navigation";

export default function WorkspacePage() {
  const router = useRouter();
  const { workspace } = useWorkspace();

  router.replace(`/${workspace?.slug}/apis`);

  return null;
}
