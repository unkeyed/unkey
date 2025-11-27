"use client";
import { TRPCProvider } from '@/lib/trpc/client';
import type { AppRouter } from '@/lib/trpc/routers';
import { QueryClient, QueryClientProvider } from '@tanstack/react-query';
import { createTRPCClient, httpBatchLink, httpLink, splitLink } from '@trpc/client';
import type React from 'react';
import { PropsWithChildren, useState } from 'react';
import SuperJSON from 'superjson';


function makeQueryClient() {
  return new QueryClient({
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
}

export let browserQueryClient: QueryClient | undefined = undefined;

function getQueryClient() {
  if (typeof window === 'undefined') {
    // Server: always make a new query client
    return makeQueryClient();
  }
  // Browser: reuse the same client across re-renders
  if (!browserQueryClient) {
    browserQueryClient = makeQueryClient();
  }
  return browserQueryClient;
}

function getBaseUrl() {
  if (typeof window !== 'undefined') {
    // Browser should use relative path
    return '';
  }
  if (process.env.VERCEL_URL) {
    return `https://${process.env.VERCEL_URL}`;
  }
  // Assume localhost
  return `http://localhost:${process.env.PORT ?? 3000}`;
}

export function ReactQueryProvider({ children }: PropsWithChildren) {
  const queryClient = getQueryClient();
  const [trpcClient] = useState(() =>
    createTRPCClient<AppRouter>({
      links: [
        splitLink({
          condition(op) {
            // check for context property `skipBatch`
            return Boolean(op.context.skipBatch);
          },
          // when condition is true, use normal request
          true: httpLink({
            transformer: SuperJSON,
            url: `${getBaseUrl()}/api/trpc`,
          }),
          // when condition is false, use batching
          false: httpBatchLink({
            transformer: SuperJSON,
            url: `${getBaseUrl()}/api/trpc`,
          }),
        }),
      ],
    })
  );
  return (
    <QueryClientProvider client={queryClient}>
      <TRPCProvider trpcClient={trpcClient} queryClient={queryClient}>
        {children}
      </TRPCProvider>
    </QueryClientProvider>
  );
}
