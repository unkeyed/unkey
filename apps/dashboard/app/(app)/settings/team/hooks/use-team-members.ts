import type { Membership } from "@/lib/auth/types";
import { createTeamMembersCollection } from "@/lib/collections/team-members";
import { trpc } from "@/lib/trpc/client";
import { useLiveQuery } from "@tanstack/react-db";
import { useEffect, useMemo } from "react";

type UseTeamMembersParams = {
  orgId: string;
  enabled?: boolean;
};

export function useTeamMembers({ orgId, enabled = true }: UseTeamMembersParams) {
  // Fetch data using tRPC (existing implementation)
  const {
    data: tRPCData,
    isLoading: tRPCLoading,
    isError: tRPCError,
  } = trpc.org.members.list.useQuery(orgId, {
    enabled: enabled && !!orgId,
  });

  // Create the collection for this organization
  const collection = useMemo(() => {
    if (!enabled || !orgId) return null;

    // Initialize with tRPC data if available
    const initialData = tRPCData?.data || [];
    return createTeamMembersCollection(orgId, initialData);
  }, [orgId, enabled, tRPCData?.data]);

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
      tRPCData.data.forEach((member) => {
        const existing = currentItems.find((item) => item.id === member.id);
        if (existing) {
          // Update existing member
          collection.update(member.id, () => member);
        } else {
          // Insert new member
          collection.insert(member);
        }
      });
    }
  }, [collection, tRPCData?.data]);

  // Use live query to get reactive data from the collection
  const liveQueryResult = collection
    ? useLiveQuery(collection)
    : { data: [], isLoading: false, isError: false };

  const { data: members, isError: liveQueryError, isLoading: liveQueryLoading } = liveQueryResult;

  // Transform the data to match the expected format
  const transformedMembers = useMemo(() => {
    if (!members || !Array.isArray(members)) return [];
    return members as Membership[];
  }, [members]);

  return {
    data: transformedMembers.length > 0 ? { data: transformedMembers } : tRPCData,
    isLoading: tRPCLoading || liveQueryLoading,
    error: tRPCError || liveQueryError,
    // Helper methods for optimistic mutations
    updateMemberRole: (membershipId: string, newRole: string) => {
      if (!collection) return;

      collection.update(membershipId, (draft) => {
        draft.role = newRole;
      });
    },
    removeMember: (membershipId: string) => {
      if (!collection) return;

      collection.delete(membershipId);
    },
  };
}
