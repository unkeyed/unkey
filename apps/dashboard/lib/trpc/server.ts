import { createTRPCClient, httpLink } from "@trpc/client";
import superjson from "superjson";

import type { Router } from "./routers";

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

export const trpc = createTRPCClient<Router>({
  links: [
    httpLink({
      transformer: superjson,
      url: `${getBaseUrl()}/api/trpc`,
    }),
  ],
});
