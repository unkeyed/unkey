"use client";

import { Suspense } from "react";

interface WorkspaceLayoutProps {
  children: React.ReactNode;
}

export default function WorkspaceLayout({ children }: WorkspaceLayoutProps) {
  return <Suspense fallback={null}>{children}</Suspense>;
}
