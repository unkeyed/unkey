"use client"


import type { Router } from "@/lib/trpc/routers";
import { QueryClient } from "@tanstack/query-core";
import { createTRPCProxyClient, httpBatchLink } from "@trpc/client";
import superjson from "superjson";

export const queryClient = new QueryClient();

// Create vanilla TRPC client for one-time calls
export const trpcClient = createTRPCProxyClient<Router>({
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
