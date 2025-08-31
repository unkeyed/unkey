import type { Membership, MembershipListResponse } from "@/lib/auth/types";
import { createCollection, localOnlyCollectionOptions } from "@tanstack/react-db";
import SuperJSON from "superjson";
import { z } from "zod";

// Zod schema for team member data validation
const membershipSchema = z.object({
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
 * Fetch function for team members that calls the tRPC endpoint directly
 */
async function fetchTeamMembers(orgId: string): Promise<Membership[]> {
  const baseUrl =
    typeof window !== "undefined" ? "" : `http://localhost:${process.env.PORT ?? 3000}`;

  const response = await fetch(`${baseUrl}/api/trpc/org.members.list`, {
    method: "POST",
    headers: {
      "Content-Type": "application/json",
    },
    body: JSON.stringify({
      json: orgId,
    }),
  });

  if (!response.ok) {
    throw new Error(`Failed to fetch team members: ${response.status} ${response.statusText}`);
  }

  const data = await response.json();

  // Handle SuperJSON deserialization if needed
  const result = SuperJSON.deserialize(data);
  const membershipListResponse: MembershipListResponse = (result as any).result.data;

  return membershipListResponse.data;
}

/**
 * Factory function to create team members collection for a specific organization
 *
 * For now, we'll use a LocalOnly collection and populate it manually from the tRPC data
 * This provides the TanStack DB reactive benefits without the complexity of QueryCollection
 */
export const createTeamMembersCollection = (orgId: string, initialData: Membership[] = []) => {
  return createCollection(
    localOnlyCollectionOptions({
      id: `team-members-${orgId}`,
      getKey: (item: Membership) => item.id,
      initialData,

      // Future optimistic mutations for team member management
      onUpdate: async ({ transaction }) => {
        // Handle member role updates
        const { original, modified } = transaction.mutations[0];

        // Call the role update tRPC endpoint
        const response = await fetch(`/api/trpc/org.members.update`, {
          method: "POST",
          headers: {
            "Content-Type": "application/json",
          },
          body: JSON.stringify({
            json: {
              membershipId: original.id,
              role: modified.role,
            },
          }),
        });

        if (!response.ok) {
          throw new Error(`Failed to update member role: ${response.status}`);
        }
      },

      onDelete: async ({ transaction }) => {
        // Handle member removal
        const { original } = transaction.mutations[0];

        // Call the member removal tRPC endpoint
        const response = await fetch(`/api/trpc/org.members.remove`, {
          method: "POST",
          headers: {
            "Content-Type": "application/json",
          },
          body: JSON.stringify({
            json: {
              orgId: orgId,
              membershipId: original.id,
            },
          }),
        });

        if (!response.ok) {
          throw new Error(`Failed to remove member: ${response.status}`);
        }
      },
    }),
  );
};
