import { collectPageViewAnalytics } from "@/lib/analytics";
import { db } from "@/lib/db";
// import { authMiddleware, clerkClient } from "@clerk/nextjs";
// import { redirectToSignIn } from "@clerk/nextjs";
import { env } from "@/lib/env";
import { auth } from "@/lib/auth/index"
import { NextFetchEvent, NextRequest, NextResponse } from "next/server";

const findWorkspace = async ({ tenantId }: { tenantId: string }) => {
  const workspace = await db.query.workspaces.findFirst({
    where: (table, { and, eq, isNull }) =>
      and(eq(table.tenantId, tenantId), isNull(table.deletedAt)),
  });
  return workspace;
};

const SIGN_IN_URL = '/auth/sign-in';

export default async function (request: NextRequest, evt: NextFetchEvent) {
  let response: NextResponse;
  const AUTH_PROVIDER = env().AUTH_PROVIDER;
  const isEnabled = () => AUTH_PROVIDER === 'workos';
    
    try {
        console.debug('Processing middleware for URL:', request.url);

        response = await auth.createMiddleware({
          enabled: isEnabled(),
        })(request)
    }
        
    catch (error) {
        console.error('Middleware error:', error);
        // Return a basic response in case of error
        response = new NextResponse();
    }
    
    // // Handle analytics
    // evt.waitUntil(
    //     collectPageViewAnalytics({
    //         req: request,
    //         userId,
    //         tenantId
    //     })
    // );
    
    return response;
}

export const config = {
  matcher: [
    "/",
    "/apis",
    "/apis/(.*)",
    "/audit",
    "/audit/(.*)",
    "/authorization",
    "/authorization/(.*)",
    "/debug",
    "/debug/(.*)",
    "/gateways",
    "/gateways/(.*)",
    "/overview",
    "/overview/(.*)",
    "/ratelimits",
    "/ratelimits/(.*)",
    "/secrets",
    "/secrets/(.*)",
    "/semant-cache",
    "/semantic-cache/(.*)",
    "/settings",
    "/settings/(.*)",
    "/success",
    "/success/(.*)",
    "/auth/(.*)",
    "/(api|trpc)(.*)",
    "/((?!.+\\.[\\w]+$|_next).*)",
    "/((?!_next/static|_next/image|images|favicon.ico|$).*)",
  ],
};
