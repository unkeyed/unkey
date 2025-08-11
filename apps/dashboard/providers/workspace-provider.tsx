"use client";

import { trpc } from "@/lib/trpc/client";
import type { Quotas, Workspace } from "@unkey/db";
import type React from "react";
import { type PropsWithChildren, createContext, useContext } from "react";

interface WorkspaceContextType {
  workspace: Workspace | null;
  quotas: Quotas | null;
  isLoading: boolean;
  error: Error | null;
  refetch: () => void;
}

const WorkspaceContext = createContext<WorkspaceContextType | undefined>(undefined);

export const useWorkspace = () => {
  const context = useContext(WorkspaceContext);
  if (context === undefined) {
    throw new Error("useWorkspace must be used within a WorkspaceProvider");
  }
  return context;
};

export const WorkspaceProvider: React.FC<PropsWithChildren> = ({ children }) => {
  const {
    data: workspace,
    isLoading,
    error,
    refetch,
  } = trpc.workspace.getCurrent.useQuery(undefined, {
    staleTime: 1000 * 60 * 5, // 5 minutes
    cacheTime: 1000 * 60 * 10, // 10 minutes
    retry: 1,
    refetchOnWindowFocus: true,
    refetchOnReconnect: true,
  });

  const value: WorkspaceContextType = {
    workspace: workspace ?? null,
    quotas: workspace?.quotas ?? null,
    isLoading,
    error: error as Error | null,
    refetch,
  };

  return <WorkspaceContext.Provider value={value}>{children}</WorkspaceContext.Provider>;
};
