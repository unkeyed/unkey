"use client";

import { prototypeHandlers } from "@/lib/trpc/prototype-handlers";
import { createPrototypeLink } from "@/lib/trpc/prototype-link";
import type { Router } from "@/lib/trpc/routers";
import { QueryClient } from "@tanstack/query-core";
import { createTRPCProxyClient, httpBatchLink } from "@trpc/client";
import superjson from "superjson";
import { getBaseUrl } from "../utils";

export const queryClient = new QueryClient();

// Create vanilla TRPC client for one-time calls
export const trpcClient = createTRPCProxyClient<Router>({
  transformer: superjson,
  links: [
    // Vibe-branch prototype: injects fake projects/namespaces into the
    // collections layer (see lib/trpc/prototype-handlers.ts).
    createPrototypeLink(prototypeHandlers),
    httpBatchLink({
      url: `${getBaseUrl()}/api/trpc`,
      fetch(url, options) {
        return fetch(url, {
          ...options,
          credentials: "include",
        });
      },
    }),
  ],
});
