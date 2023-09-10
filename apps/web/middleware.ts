import { db, eq, schema } from "@/lib/db";
import { authMiddleware } from "@clerk/nextjs";
import { redirectToSignIn } from "@clerk/nextjs";
import { NextFetchEvent, NextRequest, NextResponse } from "next/server";
const DEBUG_ON = process.env.CLERK_DEBUG === "true";
import { getSessionId, ingestPageView } from "@/lib/analytics";

const findWorkspace = async ({ tenantId }: { tenantId: string }) => {
  const workspace = await db.query.workspaces.findFirst({
    where: eq(schema.workspaces.tenantId, tenantId),
  });
  return workspace;
};

const publicRoutes = [
  "/",
  "/auth(.*)",
  "/discord",
  "/pricing",
  "/about",
  "/blog",
  "/blog/(.*)",
  "/changelog",
  "/changelog(.*)",
  "/policies",
  "/policies/(.*)",
  "/docs",
  "/docs(.*)",
  "/og",
  "/og/(.*)",
  "/api/v1/stripe/webhooks",
  "/api/v1/cron/(.*)",
  "/api/v1/clerk/webhooks",
];

export default async function (req: NextRequest, evt: NextFetchEvent) {
  let userId: string | undefined = undefined;
  let tenantId: string | undefined = undefined;

  const res = await authMiddleware({
    publicRoutes,
    signInUrl: "/auth/sign-in",
    debug: DEBUG_ON,

    afterAuth: async (auth, req) => {
      if (!(auth.userId || auth.isPublicRoute)) {
        return redirectToSignIn({ returnBackUrl: req.url });
      }
      userId = auth.userId ?? undefined;
      tenantId = auth.orgId ?? auth.userId ?? undefined;
      // Stops users from accessing the application if they have not paid yet.
      if (
        auth.orgId &&
        !["/app/stripe", "/app/apis", "/app", "/new"].includes(req.nextUrl.pathname)
      ) {
        const workspace = await findWorkspace({ tenantId: auth.orgId });
        if (workspace?.plan === "free") {
          return NextResponse.redirect(new URL("/app/stripe", req.url));
        }
        return NextResponse.next();
      }
      if (auth.userId && !auth.orgId && req.nextUrl.pathname === "/app/apis") {
        const workspace = await findWorkspace({ tenantId: auth.userId });
        if (!workspace) {
          return NextResponse.redirect(new URL("/new", req.url));
        }
      }
    },
  })(req, evt);

  // @ts-ignore
  const sessionId = getSessionId(req, res);

  // replace ids to make aggregations easier
  const path = req.nextUrl.pathname
    .replace(/\/(api_[a-zA-Z0-9]+)/g, "[apiId]")
    .replace(/\/(key_[a-zA-Z0-9]+)/g, "[keyId]");

  evt.waitUntil(
    ingestPageView({
      time: Date.now(),
      sessionId,
      userId,
      tenantId,
      path,
    }),
  );

  return res;
}

export const config = {
  matcher: ["/((?!.*\\..*|_next).*)", "/", "/(api|trpc)(.*)"],
};
