"use client";

import { Loading } from "@unkey/ui";
import { Suspense } from "react";

interface WorkspaceLayoutProps {
  children: React.ReactNode;
}

function WorkspaceLoadingFallback() {
  return (
    <div className="flex items-center justify-center w-full h-full min-h-[200px]">
      <div className="flex flex-col items-center gap-4">
        <Loading size={24} />
        <p className="text-sm text-gray-600 dark:text-gray-400">Loading workspace...</p>
      </div>
    </div>
  );
}

export default function WorkspaceLayout({ children }: WorkspaceLayoutProps) {
  return <Suspense fallback={<WorkspaceLoadingFallback />}>{children}</Suspense>;
}
