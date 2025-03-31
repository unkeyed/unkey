import { createTRPCProxyClient, httpLink } from "@trpc/client";
import superjson from "superjson";

import type { Router } from "./routers";
import { getBaseUrl } from "../utils";

export const trpc = createTRPCProxyClient<Router>({
  transformer: superjson,
  links: [
    httpLink({
      url: `${getBaseUrl()}/api/trpc`,
    }),
  ],
});
