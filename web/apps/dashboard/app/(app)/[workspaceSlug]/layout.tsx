"use client";

import { LoadingState } from "@/components/loading-state";
import { Suspense } from "react";

interface WorkspaceLayoutProps {
  children: React.ReactNode;
}

export default function WorkspaceLayout({ children }: WorkspaceLayoutProps) {
  return <Suspense fallback={<LoadingState message="Loading workspace..." />}>{children}</Suspense>;
}
