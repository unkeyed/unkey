import type { Router } from "@/lib/trpc/routers";
import { QueryClient } from "@tanstack/query-core";
import { queryCollectionOptions } from "@tanstack/query-db-collection";
import { createCollection, createOptimisticAction } from "@tanstack/react-db";
import { createTRPCProxyClient, httpBatchLink } from "@trpc/client";
import superjson from "superjson";

const queryClient = new QueryClient();




// Create vanilla TRPC client for one-time calls
const trpcClient = createTRPCProxyClient<Router>({
  transformer: superjson,
  links: [
    httpBatchLink({
      url: "/api/trpc",
      fetch(url, options) {
        return fetch(url, {
          ...options,
          credentials: "include",
        });
      },
    }),
  ],
});


const ratelimitNamespaces = createCollection(
  queryCollectionOptions({
    queryClient,
    queryKey: ["ratelimitNamespaces"],
    queryFn: async () => {
      console.info("DB fetching ratelimitNamespaces")
      return await trpcClient.ratelimit.namespace.list.query();
    },
    getKey: (item) => item.id,
    onInsert: async ({ transaction }) => {
      const { changes: newNamespace } = transaction.mutations[0]
      await trpcClient.ratelimit.namespace.create.mutate({ name: newNamespace.name! });
    },
    onUpdate: async ({ transaction }) => {
      const { original, modified } = transaction.mutations[0];
      await trpcClient.ratelimit.namespace.update.name.mutate({
        namespaceId: original.id,
        name: modified.name!
      })
    },
    onDelete: async ({ transaction }) => {

      const { original } = transaction.mutations[0];
      await trpcClient.ratelimit.namespace.delete.mutate({ namespaceId: original.id });
      return { refetch: true }

    },

  }),
);

export const collection = {
  ratelimitNamespaces,
};


export const createRatelimitNamespace = createOptimisticAction<{ name: string }>({
  onMutate: (args) => {
    ratelimitNamespaces.insert({
      id: Date.now().toString(),
      name: args.name,

    })
  },
  mutationFn: async (args) => {

    await trpcClient.ratelimit.namespace.create.mutate({ name: args.name });
  }
})
