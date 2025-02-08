import { createContext } from "@/lib/trpc/context";
import { router } from "@/lib/trpc/routers";
import { fetchRequestHandler } from "@trpc/server/adapters/fetch";
// export API handler
export async function handler(req: Request) {
  try {
    return fetchRequestHandler({
      endpoint: "/api/trpc",
      router,
      req,
      createContext,
    });
  } catch (err) {
    console.log(err);
  }
}

export { handler as GET, handler as POST };
