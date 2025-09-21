"use client";
import { useWorkspace } from "@/providers/workspace-provider";
import { useRouter } from "next/navigation";

export default function SettingsPage() {
  const { workspace } = useWorkspace();
  const router = useRouter();

  router.replace(`/${workspace?.slug}/settings/general`);

  return null;
}
