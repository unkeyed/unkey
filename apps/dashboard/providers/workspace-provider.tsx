"use client";

import { WorkspaceIdContext } from "@/hooks/use-workspace-id";
import type { Workspace } from "@/lib/db";
import { useParams } from "next/navigation";
import { createContext, useContext } from "react";
import { useEffect, useState } from "react";

interface WorkspaceContextType {
  workspaceId: string | null;
  workspace: Workspace | null;
  isLoading: boolean;
  error: string | null;
}

const WorkspaceContext = createContext<WorkspaceContextType | undefined>(undefined);

interface WorkspaceProviderProps {
  children: React.ReactNode;
  initialWorkspace?: Workspace;
}

export function WorkspaceProvider({ children, initialWorkspace }: WorkspaceProviderProps) {
  const params = useParams();
  const [workspace, setWorkspace] = useState<Workspace | null>(initialWorkspace || null);
  const [isLoading, setIsLoading] = useState(!initialWorkspace);
  const [error, setError] = useState<string | null>(null);

  const workspaceId = params?.workspace as string;

  useEffect(() => {
    if (!workspaceId) {
      setError("No workspace ID found");
      setIsLoading(false);
      return;
    }

    // If we have an initial workspace and the ID matches, use it
    if (initialWorkspace && initialWorkspace.id === workspaceId) {
      setWorkspace(initialWorkspace);
      setIsLoading(false);
      return;
    }

    // Otherwise, we could fetch the workspace data here if needed
    // For now, we'll just set the workspace ID and let components fetch their own data
    setIsLoading(false);
  }, [workspaceId, initialWorkspace]);

  return (
    <WorkspaceIdContext.Provider value={workspaceId}>
      <WorkspaceContext.Provider
        value={{
          workspaceId,
          workspace,
          isLoading,
          error,
        }}
      >
        {children}
      </WorkspaceContext.Provider>
    </WorkspaceIdContext.Provider>
  );
}

export function useWorkspace(consumerName: string) {
  const context = useContext(WorkspaceContext);
  if (context === undefined) {
    throw new Error(`\`${consumerName}\` must be used within a WorkspaceProvider`);
  }
  return context;
}
