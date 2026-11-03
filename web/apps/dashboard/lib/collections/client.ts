"use client";

import { QueryClient } from "@tanstack/query-core";

export const queryClient = new QueryClient({
  defaultOptions: { queries: { staleTime: 30_000 } },
});
