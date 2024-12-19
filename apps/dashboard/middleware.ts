import { db } from "@/lib/db";
import { env } from "@/lib/env";
import { auth } from "@/lib/auth/server";
import { type NextFetchEvent, type NextRequest, NextResponse } from "next/server";
const findWorkspace = async ({ tenantId }: { tenantId: string }) => {
  const workspace = await db.query.workspaces.findFirst({
    where: (table, { and, eq, isNull }) =>
      and(eq(table.tenantId, tenantId), isNull(table.deletedAtM)),
  });
  return workspace;
};

export default async function (req: NextRequest, evt: NextFetchEvent) {
  const url = new URL(req.url);
  console.info("host", url.host);
  if (url.host === "gateway.new") {
    return NextResponse.redirect("https://app.unkey.com/gateway-new");
  }
  
  let res: NextResponse;
  const AUTH_PROVIDER = env().AUTH_PROVIDER;
  const isEnabled = () => AUTH_PROVIDER === 'workos';

  try {
    console.debug('Processing middleware for URL:', req.url);

    res = await auth.createMiddleware({
      enabled: isEnabled(),
      publicPaths: [
        '/auth/sign-in', 
        '/auth/sign-up',
        '/auth/sso-callback',
        '/auth/oauth-sign-in', 
        '/favicon.ico',
        '/_next',]
    })(req)
}
    
catch (error) {
    console.error('Middleware error:', error);
    // Return a basic response in case of error
    // TODO: flesh this out as an actual error
    res = new NextResponse();
}

  return res;
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
    "/new",
    "/new(.*)",
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
    "/gateway-new",
    "/(api|trpc)(.*)",
    "/((?!.+\\.[\\w]+$|_next).*)",
    "/((?!_next/static|_next/image|images|favicon.ico|$).*)",
    "/robots.txt",
  ],
};
