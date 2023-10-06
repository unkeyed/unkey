import { createContext } from "@/lib/trpc/context";
import { router } from "@/lib/trpc/routers";
import { fetchRequestHandler } from "@trpc/server/adapters/fetch";
import type { NextRequest } from "next/server";
export const config = {
  runtime: "edge",
};

// export API handler
async function handler(req: NextRequest) {
  return fetchRequestHandler({
    endpoint: "/api/trpc",
    router,
    req,
    createContext,
  });
}

export { handler as GET, handler as POST };
