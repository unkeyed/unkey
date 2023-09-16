import { db, eq, schema } from "@/lib/db";
import { authMiddleware } from "@clerk/nextjs";
import { redirectToSignIn } from "@clerk/nextjs";
import { NextFetchEvent, NextRequest, NextResponse } from "next/server";
const DEBUG_ON = process.env.CLERK_DEBUG === "true";
import { collectPageViewAnalytics } from "@/lib/analytics";

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
        // if we end up here... something is wrong with the workspace and we should redirect to the new page.
        // this should never happen.
        if(!workspace) {
          console.error("Workspace not found for orgId", auth.orgId);
          return NextResponse.redirect(new URL("/new", req.url));
        }
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

  evt.waitUntil(collectPageViewAnalytics({ req, userId, tenantId }));

  return res;
}

export const config = {
  matcher: ["/((?!.*\\..*|_next).*)", "/", "/(api|trpc)(.*)"],
};
