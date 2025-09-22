"use client";
import { useWorkspaceNavigation } from "@/hooks/use-workspace-navigation";
import { useRouter, redirect } from "next/navigation";

export default function WorkspacePage() {
  const router = useRouter();
  const workspace = useWorkspaceNavigation();

  router.replace(`/${workspace.slug}/apis`);

  return null;
}
