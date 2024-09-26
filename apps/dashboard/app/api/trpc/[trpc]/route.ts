import { createContext } from "@/lib/trpc/context";
import { router } from "@/lib/trpc/routers";
import { fetchRequestHandler } from "@trpc/server/adapters/fetch";
import type { NextRequest } from "next/server";

export const runtime = "edge";
// export API handler
const handler = async function handler(req: NextRequest) {
  return fetchRequestHandler({
    endpoint: "/api/trpc",
    router,
    req,
    createContext,
  });
};

export const GET = handler;
export const POST = handler;
