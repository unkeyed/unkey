import { createTRPCProxyClient, httpLink } from "@trpc/client";
import superjson from "superjson";

import { getBaseUrl } from "../utils";
import type { Router } from "./routers";

export const trpc = createTRPCProxyClient<Router>({
  transformer: superjson,
  links: [
    httpLink({
      url: `${getBaseUrl()}/api/trpc`,
    }),
  ],
});
