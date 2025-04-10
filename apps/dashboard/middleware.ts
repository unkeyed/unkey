import { auth } from "@/lib/auth/server";
import { env } from "@/lib/env";
import { type NextFetchEvent, type NextRequest, NextResponse } from "next/server";

export default async function (req: NextRequest, _evt: NextFetchEvent) {
  const url = new URL(req.url);
  if (url.host === "gateway.new") {
    return NextResponse.redirect("https://app.unkey.com/gateway-new");
  }

  let res: NextResponse;
  const AUTH_PROVIDER = env().AUTH_PROVIDER;
  const isEnabled = () => AUTH_PROVIDER === "workos";

  try {
    res = await auth.createMiddleware({
      enabled: isEnabled(),
      publicPaths: [
        "/auth/sign-in",
        "/auth/sign-up",
        "/auth/sso-callback",
        "/auth/oauth-sign-in",
        "/auth/join",
        "/favicon.ico",
        "/api/webhooks/stripe",
        "/api/v1/workos/webhooks",
        "/api/v1/github/verify",
        "/api/auth/refresh",
        "/_next",
      ],
    })(req);
  } catch (error) {
    console.error("Middleware error:", error);
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
