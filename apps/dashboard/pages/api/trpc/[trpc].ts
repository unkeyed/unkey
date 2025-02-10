import { createContext } from "@/lib/trpc/context";
import { router } from "@/lib/trpc/routers";
import { fetchRequestHandler } from "@trpc/server/adapters/fetch";
import type { NextRequest } from "next/server";

export const runtime = "edge";
// export API handler
export default async function handler(req: NextRequest) {

  console.log("API handler", req.headers)

  return fetchRequestHandler({
    endpoint: "/api/trpc",
    router,
    req,
    createContext,
  });
}
