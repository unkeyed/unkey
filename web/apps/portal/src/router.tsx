import { MutationCache, QueryCache, QueryClient } from "@tanstack/react-query";
import { createRouter } from "@tanstack/react-router";
import { isUnauthorizedError } from "~/lib/portal-api";
import { sessionQueryKey } from "~/lib/session";
import { routeTree } from "./routeTree.gen";

/**
 * Defaults mirror the dashboard's security-conscious settings: short stale/gc
 * windows and a single retry so end-user data is not held or retried
 * aggressively.
 */
function createQueryClient(): QueryClient {
  // The cookie this tab cached a session for is gone. Dropped rather than
  // invalidated: `ensureQueryData` hands back cached data however stale it
  // is, so only an absent entry makes the next navigation ask the server.
  const dropSessionOnUnauthorized = (error: unknown) => {
    if (isUnauthorizedError(error)) {
      client.removeQueries({ queryKey: sessionQueryKey });
    }
  };
  const client: QueryClient = new QueryClient({
    queryCache: new QueryCache({ onError: dropSessionOnUnauthorized }),
    mutationCache: new MutationCache({ onError: dropSessionOnUnauthorized }),
    defaultOptions: {
      queries: {
        staleTime: 1000 * 60 * 2, // 2 minutes
        gcTime: 1000 * 60 * 5, // 5 minutes
        retry: 1,
        refetchOnWindowFocus: true,
        refetchOnReconnect: true,
      },
    },
  });
  return client;
}

export function getRouter() {
  const router = createRouter({
    routeTree,
    context: { queryClient: createQueryClient() },
    scrollRestoration: true,
    defaultPreload: "intent",
  });

  return router;
}

declare module "@tanstack/react-router" {
  interface Register {
    router: ReturnType<typeof getRouter>;
  }
}
