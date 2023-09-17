import { collectPageViewAnalytics } from "@/lib/analytics";
import { db, eq, schema } from "@/lib/db";
import { authMiddleware, clerkClient } from "@clerk/nextjs";
import { redirectToSignIn } from "@clerk/nextjs";
import { NextFetchEvent, NextRequest, NextResponse } from "next/server";

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
    debug: process.env.CLERK_DEBUG === "true",

    afterAuth: async (auth, req) => {
      if (!(auth.userId || auth.isPublicRoute)) {
        return redirectToSignIn({ returnBackUrl: req.url });
      }
      userId = auth.userId ?? undefined;
      tenantId = auth.orgId ?? auth.userId ?? undefined;
      if (auth.orgId && !auth.isPublicRoute) {
        const workspace = await findWorkspace({ tenantId: auth.orgId });
        if (!workspace && req.nextUrl.pathname !== "/new") {
          console.error("Workspace not found for orgId", auth.orgId);
          await clerkClient.organizations.deleteOrganization(auth.orgId);
          console.log("Deleted orgId", auth.orgId, " sending to create new workspace.");
          return NextResponse.redirect(new URL("/new", req.url));
        }
        // this stops users if they haven't paid.
        if (!["/app/stripe", "/app/apis", "/app", "/new"].includes(req.nextUrl.pathname)) {
          if (workspace?.plan === "free") {
            return NextResponse.redirect(new URL("/app/stripe", req.url));
          }
          return NextResponse.next();
        }
      }
      if (auth.userId && !auth.orgId && !auth.isPublicRoute) {
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
