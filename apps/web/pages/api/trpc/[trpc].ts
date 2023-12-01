import { createContext } from "@/lib/trpc/context";
import { router } from "@/lib/trpc/routers";
import { fetchRequestHandler } from "@trpc/server/adapters/fetch";
import type { NextRequest } from "next/server";

// export API handler
export default async function handler(req: NextRequest) {
  return fetchRequestHandler({
    endpoint: "/api/trpc",
    router,
    req,
    createContext,
  });
}
