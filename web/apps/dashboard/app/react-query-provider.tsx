"use client";

import { trpc } from "@/lib/trpc/client";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { httpBatchLink, httpLink, splitLink } from "@trpc/client";
import type React from "react";
import { type PropsWithChildren, useState } from "react";
import SuperJSON from "superjson";

export const ReactQueryProvider: React.FC<PropsWithChildren> = ({ children }) => {
  function getBaseUrl() {
    if (typeof window !== "undefined") {
      // browser should use relative path
      return "";
    }

    // VERCEL_URL is always the auto-generated *.vercel.app deployment URL.
    // The production custom domain lives in VERCEL_PROJECT_PRODUCTION_URL.
    // For previews, prefer VERCEL_BRANCH_URL (stable across deploys of the
    // same branch) so links and redirects remain valid between deployments.
    if (process.env.VERCEL_ENV === "production" && process.env.VERCEL_PROJECT_PRODUCTION_URL) {
      return `https://${process.env.VERCEL_PROJECT_PRODUCTION_URL}`;
    }
    if (process.env.VERCEL_BRANCH_URL) {
      return `https://${process.env.VERCEL_BRANCH_URL}`;
    }
    if (process.env.VERCEL_URL) {
      return `https://${process.env.VERCEL_URL}`;
    }
    return `http://localhost:${process.env.PORT ?? 3000}`;
  }

  // INFO: https://trpc.io/docs/client/links/splitLink#disable-batching-for-certain-requests
  // For disable batching read this
  const [queryClient] = useState(
    () =>
      new QueryClient({
        defaultOptions: {
          queries: {
            staleTime: 1000 * 60 * 2, // 2 minutes (reduced for security)
            // Keep cache for 5 minutes (reduced from 10)
            cacheTime: 1000 * 60 * 5,
            // Retry failed queries 1 time instead of default 3
            retry: 1,
            // Will refetch only if the data is stale
            refetchOnWindowFocus: true,
            // Minimize connection issues by retrying when reconnecting
            refetchOnReconnect: true,
            // Don't show stale data to prevent user data leakage
            keepPreviousData: false,
          },
        },
      }),
  );
  const [trpcClient] = useState(() =>
    trpc.createClient({
      transformer: SuperJSON,
      links: [
        splitLink({
          condition(op) {
            // check for context property `skipBatch`
            return Boolean(op.context.skipBatch);
          },
          // when condition is true, use normal request
          true: httpLink({
            url: `${getBaseUrl()}/api/trpc`,
          }),
          // when condition is false, use batching
          false: httpBatchLink({
            url: `${getBaseUrl()}/api/trpc`,
          }),
        }),
      ],
    }),
  );

  return (
    <trpc.Provider client={trpcClient} queryClient={queryClient}>
      <QueryClientProvider client={queryClient}>{children}</QueryClientProvider>
    </trpc.Provider>
  );
};
