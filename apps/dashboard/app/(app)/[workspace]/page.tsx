"use client";

import { useWorkspace } from "@/providers/workspace-provider";
import { Loading } from "@unkey/ui";
import { useRouter } from "next/navigation";

export default function WorkspacePage() {
  const router = useRouter();
  const { workspace, isLoading } = useWorkspace();

  if (workspace && !isLoading) {
    router.replace(`/${workspace.slug}/apis`);
  }

  // Show loading state while redirecting
  return (
    <div className="min-h-screen flex flex-col w-full h-full items-center justify-center">
      <div className="flex items-center gap-3">
        <Loading size={24} />
      </div>
    </div>
  );
}
