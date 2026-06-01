import { createContext } from "@/lib/trpc/context";
import { router } from "@/lib/trpc/routers";
import { wasReportedToSentry } from "@/lib/trpc/trpc";
import * as Sentry from "@sentry/nextjs";
import { fetchRequestHandler } from "@trpc/server/adapters/fetch";

async function handler(req: Request) {
  // The previous implementation caught errors and returned `undefined`, which both
  // hid the failure from Sentry and produced a malformed response (Next would 500
  // silently with no body). Let unhandled exceptions propagate so Sentry's
  // `captureRequestError` instrumentation reports them, but capture explicitly here
  // as well in case the runtime swallows it before the wrapper sees it.
  try {
    return await fetchRequestHandler({
      endpoint: "/api/trpc",
      router,
      req,
      createContext,
      onError({ error, path, type }) {
        // We only forward failures the procedure-level middleware did not already
        // capture — those are errors thrown outside the procedure pipeline (input
        // parsing, context bootstrapping). `handleTRPCError` flags any error it
        // captured via `markReportedToSentry`, so we skip it here to avoid emitting
        // the same event twice.
        if (wasReportedToSentry(error)) {
          return;
        }
        if (!error.code || error.code === "INTERNAL_SERVER_ERROR") {
          Sentry.captureException(error, {
            tags: { trpc_path: path ?? "unknown", trpc_type: type },
          });
        }
      },
    });
  } catch (err) {
    // Log to stderr so the failure shows up in Vercel logs alongside Sentry.
    console.error("tRPC route handler error:", err);
    Sentry.captureException(err, { tags: { handler: "trpc_route" } });
    throw err;
  }
}

export { handler as GET, handler as POST };
