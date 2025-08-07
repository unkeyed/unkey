"use client";

import type { Workspace } from "@/lib/db";
import { WorkspaceProvider } from "./workspace-provider";

interface WorkspaceProviderWrapperProps {
  children: React.ReactNode;
  initialWorkspace?: Workspace;
}

export function WorkspaceProviderWrapper({
  children,
  initialWorkspace,
}: WorkspaceProviderWrapperProps) {
  return <WorkspaceProvider initialWorkspace={initialWorkspace}>{children}</WorkspaceProvider>;
}
