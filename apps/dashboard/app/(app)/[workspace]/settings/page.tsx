"use client";
import { useWorkspaceNavigation } from "@/hooks/use-workspace-navigation";
import { useRouter, redirect } from "next/navigation";

export default function SettingsPage() {
  const workspace = useWorkspaceNavigation();
  const router = useRouter();

  router.replace(`/${workspace.slug}/settings/general`);

  return null;
}
