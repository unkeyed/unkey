import { collectPageViewAnalytics } from "@/lib/analytics";
import { db } from "@/lib/db";
import { authMiddleware, clerkClient } from "@clerk/nextjs";
// import { redirectToSignIn } from "@clerk/nextjs";
// import { type NextFetchEvent, type NextRequest, NextResponse } from "next/server";
import { authkitMiddleware, getSession } from "@workos-inc/authkit-nextjs";
import { env } from "./lib/env";
import { NextFetchEvent, NextRequest, NextResponse } from "next/server";

const findWorkspace = async ({ tenantId }: { tenantId: string }) => {
  const workspace = await db.query.workspaces.findFirst({
    where: (table, { and, eq, isNull }) =>
      and(eq(table.tenantId, tenantId), isNull(table.deletedAt)),
  });
  return workspace;
};

const SIGN_IN_URL = '/auth/sign-in';

async function workosMiddleware(request: NextRequest, evt: NextFetchEvent) {
  let userId: string | undefined = undefined;
  let tenantId: string | undefined = undefined;
  try {
    const result = await authkitMiddleware({
      debug: true,
      middlewareAuth: {
          enabled: true,
          unauthenticatedPaths: ['/auth/(.*)']
      }
    })(request, evt);

    if (!result) {
      console.log("no response");
      //return NextResponse.redirect(new URL(SIGN_IN_URL, request.url));
    }

    // Ensure we have a NextResponse object to make `getSession` happy :(
    const response = result instanceof NextResponse 
      ? result 
      : NextResponse.next();
    const session = await getSession(response);

    if (!session) {
      console.log("no session");
      //return NextResponse.redirect(new URL(SIGN_IN_URL, request.url));
    }

    // Extract user info for analytics
    console.log("mcs user", session)
    // userId = session.user?.id;
    // tenantId = session.tenant?.id;

    return response;
  }
  catch (error) {
    console.error('Auth middleware error:', error);
    return NextResponse.redirect(new URL(SIGN_IN_URL, request.url));
  }
}

async function localMiddleware(request: NextRequest) {
  const response = new NextResponse();
  return response;
}

export default async function (request: NextRequest, evt: NextFetchEvent) {
  let response: NextResponse;
  const AUTH_PROVIDER = env().AUTH_PROVIDER;
  // const privateMatch = "^/";
    
    try {
        console.debug('Processing middleware for URL:', request.url);
        
        switch (AUTH_PROVIDER) {
            case 'workos':
                response = await workosMiddleware(request, evt);
                break;
                
            case 'local':
                response = await localMiddleware(request);
                break;
                
            default:
                throw new Error(`Unsupported AUTH_PROVIDER: ${AUTH_PROVIDER}`);
        }
        
        
    } catch (error) {
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
