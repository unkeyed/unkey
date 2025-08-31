import type { Membership, MembershipListResponse } from "@/lib/auth/types";
import { createCollection, localOnlyCollectionOptions } from "@tanstack/react-db";
import SuperJSON from "superjson";
import { z } from "zod";

// Zod schema for user membership data validation (reusing from team members)
const userMembershipSchema = z.object({
  id: z.string(),
  user: z.object({
    id: z.string(),
    email: z.string(),
    firstName: z.string().nullable(),
    lastName: z.string().nullable(),
    avatarUrl: z.string().nullable(),
    fullName: z.string().nullable(),
  }),
  organization: z.object({
    id: z.string(),
    name: z.string(),
    createdAt: z.string().optional(),
    updatedAt: z.string().optional(),
  }),
  role: z.string(),
  createdAt: z.string(),
  updatedAt: z.string(),
  status: z.enum(["pending", "active", "inactive"]),
});

/**
 * Fetch function for user memberships that calls the tRPC endpoint directly
 */
async function fetchUserMemberships(userId: string): Promise<Membership[]> {
  const baseUrl =
    typeof window !== "undefined" ? "" : `http://localhost:${process.env.PORT ?? 3000}`;

  const response = await fetch(`${baseUrl}/api/trpc/user.listMemberships`, {
    method: "POST",
    headers: {
      "Content-Type": "application/json",
    },
    body: JSON.stringify({
      json: userId,
    }),
  });

  if (!response.ok) {
    throw new Error(`Failed to fetch user memberships: ${response.status} ${response.statusText}`);
  }

  const data = await response.json();

  // Handle SuperJSON deserialization if needed
  const result = SuperJSON.deserialize(data);
  const membershipListResponse: MembershipListResponse = (result as any).result.data;

  return membershipListResponse.data;
}

/**
 * Factory function to create user workspaces collection for a specific user
 *
 * This collection provides reactive access to the user's workspace memberships
 * for the workspace switcher dropdown.
 */
export const createUserWorkspacesCollection = (userId: string, initialData: Membership[] = []) => {
  return createCollection(
    localOnlyCollectionOptions({
      id: `user-workspaces-${userId}`,
      getKey: (item: Membership) => item.id,
      initialData,

      // Future optimistic mutations for workspace operations
      onUpdate: async ({ transaction }) => {
        // This could handle operations like changing role within a workspace
        // For now, workspace switching is handled via the existing tRPC mutation
        const { original, modified } = transaction.mutations[0];
        console.log(`User workspace role updated: ${original.role} -> ${modified.role}`);
      },

      onDelete: async ({ transaction }) => {
        // This could handle leaving a workspace
        // For now, not implementing as it's not a common user operation
        const { original } = transaction.mutations[0];
        console.log(`User left workspace: ${original.organization.name}`);
      },
    }),
  );
};
