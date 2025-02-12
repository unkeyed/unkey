"use client";

import { TRPCProvider } from "@/lib/trpc/client";
import { Router } from "@/lib/trpc/routers";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { httpBatchLink, createTRPCClient } from "@trpc/client";
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
    createTRPCClient<Router>({
      links: [
        httpBatchLink({
          transformer: SuperJSON,
          url: `${getBaseUrl()}/api/trpc`,
        }),
      ],
    }),
  );

  return (
    (<TRPCProvider trpcClient={trpcClient} queryClient={queryClient}>
      <QueryClientProvider client={queryClient}>{children}</QueryClientProvider>
    </TRPCProvider>)
  );
};
