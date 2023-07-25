import { fetchRequestHandler } from "@trpc/server/adapters/fetch";
import type { NextRequest } from "next/server";
import { createContext } from "@/lib/trpc/context";
import { router } from "@/lib/trpc/routers";
import { clerkClient } from "@clerk/nextjs";
export const config = {
  runtime: "edge",
};

clerkClient.users.verifyPassword({
  password: "test",
  userId: "user_2132141",
});

// export API handler
export default async function handler(req: NextRequest) {
  return fetchRequestHandler({
    endpoint: "/api/trpc",
    router,
    req,
    createContext,
  });
}
