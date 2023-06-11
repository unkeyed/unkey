import * as trpcNext from "@trpc/server/adapters/next";

import { createContext } from "@/lib/trpc/context";
import { router } from "@/lib/trpc/routers";

export default trpcNext.createNextApiHandler({
  router,
  createContext,
});
