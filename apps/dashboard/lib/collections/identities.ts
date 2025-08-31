import { createCollection, localOnlyCollectionOptions } from "@tanstack/react-db";
import SuperJSON from "superjson";
import { z } from "zod";

// Identity schema based on the tRPC IdentityResponseSchema
const identitySchema = z.object({
  id: z.string(),
  externalId: z.string(),
  workspaceId: z.string(),
  environment: z.string(),
  meta: z.record(z.unknown()).nullable(),
  createdAt: z.number(),
  updatedAt: z.number().nullable(),
});

export type Identity = z.infer<typeof identitySchema>;

// Response type from the tRPC endpoint
export type IdentitiesResponse = {
  identities: Identity[];
  hasMore: boolean;
  nextCursor?: string | null;
};

/**
 * Fetch function for identities that calls the tRPC endpoint directly
 */
async function fetchIdentities(params: {
  cursor?: string;
  limit?: number;
}): Promise<IdentitiesResponse> {
  const baseUrl =
    typeof window !== "undefined" ? "" : `http://localhost:${process.env.PORT ?? 3000}`;

  const response = await fetch(`${baseUrl}/api/trpc/identity.query`, {
    method: "POST",
    headers: {
      "Content-Type": "application/json",
    },
    body: JSON.stringify({
      json: {
        cursor: params.cursor,
        limit: params.limit || 50,
      },
    }),
  });

  if (!response.ok) {
    throw new Error(`Failed to fetch identities: ${response.status} ${response.statusText}`);
  }

  const data = await response.json();

  // Handle SuperJSON deserialization if needed
  const result = SuperJSON.deserialize(data);
  return (result as any).result.data;
}

/**
 * Fetch function for searching identities
 */
async function searchIdentities(query: string): Promise<Identity[]> {
  const baseUrl =
    typeof window !== "undefined" ? "" : `http://localhost:${process.env.PORT ?? 3000}`;

  const response = await fetch(`${baseUrl}/api/trpc/identity.search`, {
    method: "POST",
    headers: {
      "Content-Type": "application/json",
    },
    body: JSON.stringify({
      json: { query },
    }),
  });

  if (!response.ok) {
    throw new Error(`Failed to search identities: ${response.status} ${response.statusText}`);
  }

  const data = await response.json();

  // Handle SuperJSON deserialization if needed
  const result = SuperJSON.deserialize(data);
  return (result as any).result.data.identities;
}

/**
 * Factory function to create identities collection for a workspace
 *
 * This collection provides reactive access to the workspace's identities
 * with support for both listing and searching.
 */
export const createIdentitiesCollection = (workspaceId: string, initialData: Identity[] = []) => {
  return createCollection(
    localOnlyCollectionOptions({
      id: `identities-${workspaceId}`,
      getKey: (item: Identity) => item.id,
      initialData,

      // Future optimistic mutations for identity operations
      onUpdate: async () => {
        // Handle identity updates (e.g., metadata changes)
        console.log("Identity updated");

        // Here you could call a tRPC mutation to update the identity on the server
        // For now, just logging
      },

      onInsert: async () => {
        // Handle identity creation
        console.log("New identity created");

        // Here you could call the tRPC identity.create mutation
      },

      onDelete: async () => {
        // Handle identity deletion
        console.log("Identity deleted");

        // Here you could call a tRPC mutation to delete the identity
      },
    }),
  );
};

// Export the fetch functions for use in hooks
export { fetchIdentities, searchIdentities };
