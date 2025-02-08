import { createContext } from "@/lib/trpc/context";
import { router } from "@/lib/trpc/routers";
import { fetchRequestHandler } from "@trpc/server/adapters/fetch";
// export API handler
export default async function handler(req: Request) {
  try {
    return fetchRequestHandler({
      endpoint: "/api/trpc",
      router,
      req,
      createContext,
    });
  } catch (err) {
    console.error(err);
  }
}

export { handler as GET, handler as POST };
