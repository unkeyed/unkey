"use client";

import { useRouter } from "next/navigation";
import { useEffect } from "react";

export default function WorkspacePage({ params }: { params: { workspaceId: string } }) {
  const router = useRouter();

  useEffect(() => {
    if (params.workspaceId) {
      router.replace(`/${params.workspaceId}/apis`);
    } else {
      router.replace("/new");
    }
  }, [params.workspaceId, router]);

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
