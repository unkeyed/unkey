"use client";

import type { Router } from "@/lib/trpc/routers";
import { QueryClient } from "@tanstack/query-core";
import { createTRPCProxyClient, httpBatchLink } from "@trpc/client";
import superjson from "superjson";
import { getBaseUrl } from "../utils";

export const queryClient = new QueryClient({
  defaultOptions: { queries: { staleTime: 30_000 } },
});

// Create vanilla TRPC client for one-time calls
export const trpcClient = createTRPCProxyClient<Router>({
  transformer: superjson,
  links: [
    httpBatchLink({
      url: `${getBaseUrl()}/api/trpc`,
      maxURLLength: 2_000,
      fetch(url, options) {
        return fetch(url, {
          ...options,
          credentials: "include",
        });
      },
    }),
  ],
});
