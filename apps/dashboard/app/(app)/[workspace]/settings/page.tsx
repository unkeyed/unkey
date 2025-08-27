"use client";
import { useWorkspace } from "@/providers/workspace-provider";
import { useRouter } from "next/navigation";

export default function SettingsPage() {
  const { workspace } = useWorkspace();
  const router = useRouter();

  if (workspace) {
    router.replace(`/${workspace.id}/settings/general`);
  }

  if (!workspace) {
    router.push("/new");
  }
}
