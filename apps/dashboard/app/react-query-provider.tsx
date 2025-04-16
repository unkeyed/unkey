"use client";

import { trpc } from "@/lib/trpc/client";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { ReactQueryDevtools } from "@tanstack/react-query-devtools";
import { httpBatchLink } from "@trpc/client";
import type React from "react";
import { type PropsWithChildren, useState } from "react";
import SuperJSON from "superjson";

export const ReactQueryProvider: React.FC<PropsWithChildren> = ({ children }) => {
  function getBaseUrl() {
    if (typeof window !== "undefined") {
      // browser should use relative path
      return "";
    }

    if (process.env.VERCEL_URL) {
      // reference for vercel.com
      return `https://${process.env.VERCEL_URL}`;
    }

    // assume localhost
    return `http://localhost:${process.env.PORT ?? 3000}`;
  }

  const [queryClient] = useState(() => new QueryClient());
  const [trpcClient] = useState(() =>
    trpc.createClient({
      transformer: SuperJSON,
      links: [
        httpBatchLink({
          url: `${getBaseUrl()}/api/trpc`,
        }),
      ],
    }),
  );

  return (
    // The tRPC Provider should wrap the QueryClientProvider
    <trpc.Provider client={trpcClient} queryClient={queryClient}>
      <QueryClientProvider client={queryClient}>
        {/* Render the main application content */}
        {children}

        {/* Render React Query DevTools only in development */}
        {process.env.NODE_ENV === "development" && <ReactQueryDevtools initialIsOpen={false} />}
      </QueryClientProvider>
    </trpc.Provider>
  );
};
