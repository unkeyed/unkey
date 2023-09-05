import { db, eq, schema } from "@/lib/db";
import { authMiddleware } from "@clerk/nextjs";
import { redirectToSignIn } from "@clerk/nextjs";
import { NextResponse } from "next/server";
const DEBUG_ON = process.env.CLERK_DEBUG === "true";

const findWorkspace = async ({ tenantId }: { tenantId: string }) => {
  const workspace = await db.query.workspaces.findFirst({
    where: eq(schema.workspaces.tenantId, tenantId),
  });
  return workspace;
};

export default authMiddleware({
  publicRoutes: [
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
  ],
  signInUrl: "/auth/sign-in",
  debug: DEBUG_ON,
  async afterAuth(auth, req) {
    if (!(auth.userId || auth.isPublicRoute)) {
      return redirectToSignIn({ returnBackUrl: req.url });
    }
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
      console.log("we also here");
      const workspace = await findWorkspace({ tenantId: auth.userId });
      if (!workspace) {
        return NextResponse.redirect(new URL("/new", req.url));
      }
    }
  },
});

export const config = {
  matcher: ["/((?!.*\\..*|_next).*)", "/", "/(api|trpc)(.*)"],
};
