"use client";
import { trpc } from "@/lib/trpc/client";
import type { WorkspaceWithQuota } from "./actions";

type WorkspaceClientProviderProps = {
  initialWorkspace: WorkspaceWithQuota;
  children: React.ReactNode;
};

export function WorkspaceClientProvider({
  initialWorkspace,
  children,
}: WorkspaceClientProviderProps) {
  trpc.workspace.getWorkspace.useQuery(undefined, {
    initialData: initialWorkspace,
    staleTime: 1000 * 60 * 5, // 5 minutes
    refetchOnMount: false, // Don't refetch immediately on mount
    refetchOnWindowFocus: false,
  });

  return <>{children}</>;
}
