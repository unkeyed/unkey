"use client";

import { useWorkspace } from "@/providers/workspace-provider";
import { useRouter } from "next/navigation";
import { useEffect } from "react";

export default function WorkspacePage() {
  const router = useRouter();
  const { workspace } = useWorkspace();
  useEffect(() => {
    if (workspace) {
      router.replace(`/${workspace.id}/apis`);
    } else {
      router.replace("/new");
    }
  }, [workspace, router]);

  // Show loading state while redirecting
  return (
    <div className="min-h-screen bg-background flex items-center justify-center">
      <div className="flex items-center gap-3">
        <div className="animate-spin rounded-full h-8 w-8 border-b-2 border-brand" />
        <span className="text-content-subtle">Redirecting...</span>
      </div>
    </div>
  );
}
