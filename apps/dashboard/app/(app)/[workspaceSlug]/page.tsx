"use client";
import { useWorkspaceNavigation } from "@/hooks/use-workspace-navigation";
import { useRouter } from "next/navigation";
import { useEffect } from "react";

export default function WorkspacePage() {
  const router = useRouter();
  const workspace = useWorkspaceNavigation();

  useEffect(() => {
    router.replace(`/${workspace.slug}/apis`);
  }, [router, workspace.slug]);

  return null;
}
