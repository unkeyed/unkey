import { authMiddleware } from "@clerk/nextjs";
import { redirectToSignIn } from "@clerk/nextjs";
import { NextResponse } from "next/server";
import { db, schema, eq } from "@unkey/db";
const DEBUG_ON = process.env.CLERK_DEBUG === "true";

const checktenancy = async ({ tenantId }: { tenantId: string }) => {
  const workspace = await db.query.workspaces.findFirst({
    where: eq(schema.workspaces.tenantId, tenantId),
  });
  return workspace;
};

export default authMiddleware({
  publicRoutes: ["/", "/auth(.*)", "/discord", "/pricing", "/docs", "/about", "/blog(.*)", "/api/v1/stripe/webhooks"],
  signInUrl: "/auth/sign-in",
  debug: DEBUG_ON,
  async afterAuth(auth, req) {
    if (!(auth.userId || auth.isPublicRoute)) {
      return redirectToSignIn({ returnBackUrl: req.url });
    }
    if (auth.userId && req.nextUrl.pathname === "/app/apis") {
      const tenantId = auth.orgId ?? auth.userId;
      const workspace = await checktenancy({ tenantId });
      if (!workspace) {
        return NextResponse.redirect(new URL("/onboarding", req.url));
      }
    }
  },
});

export const config = {
  matcher: ["/((?!.*\\..*|_next).*)", "/", "/(api|trpc)(.*)"],
};
