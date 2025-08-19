"use client";
import { useWorkspace } from "@/providers/workspace-provider";
import { useRouter } from "next/navigation";

export const dynamic = "force-dynamic";

export default function OverviewPage() {
  const router = useRouter();
  const { workspace } = useWorkspace();
  if (!workspace) {
    return router.replace("/new");
  }
  router.replace(`/${workspace.id}/apis`);
}
