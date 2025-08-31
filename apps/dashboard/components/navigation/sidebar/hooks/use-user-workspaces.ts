import type { Membership } from "@/lib/auth/types";
import { createUserWorkspacesCollection } from "@/lib/collections/user-workspaces";
import { trpc } from "@/lib/trpc/client";
import { useLiveQuery } from "@tanstack/react-db";
import { useEffect, useMemo } from "react";

type UseUserWorkspacesParams = {
  userId: string;
  enabled?: boolean;
};

export function useUserWorkspaces({ userId, enabled = true }: UseUserWorkspacesParams) {
  // Fetch data using tRPC (existing implementation)
  const {
    data: tRPCData,
    isLoading: tRPCLoading,
    isError: tRPCError,
  } = trpc.user.listMemberships.useQuery(userId, {
    enabled: enabled && !!userId,
  });

  // Create the collection for this user
  const collection = useMemo(() => {
    if (!enabled || !userId) return null;

    // Initialize with tRPC data if available
    const initialData = tRPCData?.data || [];
    return createUserWorkspacesCollection(userId, initialData);
  }, [userId, enabled, tRPCData?.data]);

  // Update collection when tRPC data changes
  useEffect(() => {
    if (collection && tRPCData?.data) {
      // Clear existing data and insert fresh data from tRPC
      const currentItems = collection.toArray;
      const currentKeys = currentItems.map((item) => item.id);

      // Remove items that are no longer in the tRPC response
      currentKeys.forEach((key) => {
        if (!tRPCData.data.find((item) => item.id === key)) {
          collection.delete(key);
        }
      });

      // Insert or update items from tRPC response
      tRPCData.data.forEach((membership) => {
        const existing = currentItems.find((item) => item.id === membership.id);
        if (existing) {
          // Update existing membership
          collection.update(membership.id, () => membership);
        } else {
          // Insert new membership
          collection.insert(membership);
        }
      });
    }
  }, [collection, tRPCData?.data]);

  // Use live query to get reactive data from the collection
  const liveQueryResult = collection
    ? useLiveQuery(collection)
    : { data: [], isLoading: false, isError: false };

  const {
    data: workspaces,
    isError: liveQueryError,
    isLoading: liveQueryLoading,
  } = liveQueryResult;

  // Transform the data to match the expected format
  const transformedWorkspaces = useMemo(() => {
    if (!workspaces || !Array.isArray(workspaces)) return [];
    return workspaces as Membership[];
  }, [workspaces]);

  return {
    data: transformedWorkspaces.length > 0 ? { data: transformedWorkspaces } : tRPCData,
    isLoading: tRPCLoading || liveQueryLoading,
    error: tRPCError || liveQueryError,
    // Helper methods for optimistic mutations
    updateWorkspaceRole: (membershipId: string, newRole: string) => {
      if (!collection) return;

      collection.update(membershipId, (draft) => {
        draft.role = newRole;
      });
    },
    leaveWorkspace: (membershipId: string) => {
      if (!collection) return;

      collection.delete(membershipId);
    },
  };
}
