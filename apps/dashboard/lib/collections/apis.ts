import { createCollection, localOnlyCollectionOptions } from "@tanstack/react-db";
import type { ApiOverview } from "@/lib/trpc/routers/api/overview/query-overview/schemas";

export interface ApiOverviewWithId extends ApiOverview {
  id: string;
}

export const createApisCollection = (initialData: ApiOverview[] = []) => {
  return createCollection(
    localOnlyCollectionOptions({
      id: "apis",
      getKey: (item: ApiOverviewWithId) => item.id,
      initialData: initialData.map(api => ({ ...api, id: api.id })),
      onInsert: async () => {
        // Optimistic insert - API creation handled by tRPC mutation
      },
      onUpdate: async () => {
        // Optimistic update - API updates handled by tRPC mutation  
      },
      onDelete: async () => {
        // Optimistic delete - API deletion handled by tRPC mutation
      }
    })
  );
};

// Helper function to fetch APIs data for infinite query simulation
export async function fetchApis(params: { cursor?: string; limit?: number; }): Promise<{
  apiList: ApiOverview[];
  nextCursor?: string;
  total: number;
}> {
  const { cursor, limit = 10 } = params;
  
  // This would be called by the hook to sync with tRPC
  // For now, return empty structure - actual data comes from tRPC sync
  return {
    apiList: [],
    nextCursor: undefined,
    total: 0
  };
}

// Helper function to search APIs
export async function searchApis(query: string): Promise<ApiOverview[]> {
  // This would be called by the hook to search APIs
  // For now, return empty array - actual search handled by tRPC sync
  return [];
}