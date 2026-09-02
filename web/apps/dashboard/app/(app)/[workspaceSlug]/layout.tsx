"use client";

import { Suspense } from "react";
import { LoadingState } from "@/components/loading-state";

interface WorkspaceLayoutProps {
  children: React.ReactNode;
}

export default function WorkspaceLayout({ children }: WorkspaceLayoutProps) {
  return <Suspense fallback={<LoadingState message="Loading workspace..." />}>{children}</Suspense>;
}
