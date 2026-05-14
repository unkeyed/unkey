"use client";

import { trpc } from "@/lib/trpc/client";
import { shouldReportToSentry } from "@/lib/utils/error-classification";
import * as Sentry from "@sentry/nextjs";
import { MutationCache, QueryCache, QueryClient, QueryClientProvider } from "@tanstack/react-query";
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
        // React Query swallows errors into the query/mutation result by design — components
        // render the error state but nothing reaches Sentry's automatic instrumentation.
        // Wire the global caches so every unexpected failure is reported, while still
        // filtering the user/permission errors classified as "expected".
        queryCache: new QueryCache({
          onError: (error, query) => {
            if (!shouldReportToSentry(error)) {
              return;
            }
            Sentry.captureException(error, {
              tags: {
                source: "react_query",
                query_type: "query",
                // tRPC react-query keys are nested: [['path','segments'], { input, type }].
                // Joining the outer array stringifies the inner array and input object,
                // producing useless tag values like "path,segments.[object Object]".
                trpc_path: Array.isArray(query.queryKey?.[0])
                  ? query.queryKey[0].join(".")
                  : "unknown",
              },
            });
          },
        }),
        mutationCache: new MutationCache({
          onError: (error, _variables, _context, mutation) => {
            if (!shouldReportToSentry(error)) {
              return;
            }
            const mutationKey = mutation.options.mutationKey;
            Sentry.captureException(error, {
              tags: {
                source: "react_query",
                query_type: "mutation",
                trpc_path: Array.isArray(mutationKey?.[0])
                  ? mutationKey[0].join(".")
                  : "unknown",
              },
            });
          },
        }),
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
