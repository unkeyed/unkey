"use client";

import { useWorkspace } from "@/providers/workspace-provider";
import { Loading } from "@unkey/ui";
import { redirect, useRouter } from "next/navigation";
import { useEffect, useState } from "react";

export default function WorkspacePage() {
  const router = useRouter();
  const { workspace, isLoading, error } = useWorkspace();
  const [redirectAttempted, setRedirectAttempted] = useState(false);

  useEffect(() => {
    if (workspace?.slug && !isLoading && !redirectAttempted) {
      setRedirectAttempted(true);
      router.replace(`/${workspace.slug}/apis`);
    }
  }, [workspace, isLoading, redirectAttempted, router]);

  if (error) {
    redirect("/new");
  }

  // Show loading state while redirecting
  return (
    <div className="min-h-screen flex flex-col w-full h-full items-center justify-center">
      <div className="flex items-center gap-3">
        <Loading size={24} />
        <span className="text-gray-600">Loading workspace...</span>
      </div>
    </div>
  );
}
