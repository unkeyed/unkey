import { env } from "@/lib/env";
import { db, eq, schema } from "@/lib/db";
import { PENDING_SESSION_COOKIE } from "@/lib/auth/types";
import { getAuthCookieOptions } from "@/lib/auth/cookie-security";
import type { NextRequest } from "next/server";
import { NextResponse } from "next/server";

/**
 * Better Auth catch-all API route handler.
 *
 * This route handles:
 * - OAuth callbacks at `/api/auth/callback/github` and `/api/auth/callback/google`
 * - Internal Better Auth endpoints (session management, etc.)
 *
 * The handler delegates all requests to the Better Auth instance.
 * Only active when AUTH_PROVIDER=better-auth.
 *
 * For OAuth callbacks, we intercept the response to handle multi-org scenarios:
 * - If user has multiple orgs, redirect to org selection instead of the callbackURL
 * - If user has single org, let the normal flow continue
 */

// Lazy initialization to avoid loading Better Auth when not needed
let handler: {
  GET: (req: NextRequest) => Promise<Response>;
  POST: (req: NextRequest) => Promise<Response>;
} | null = null;

async function getHandler() {
  if (!handler) {
    const config = env();
    if (config.AUTH_PROVIDER !== "better-auth") {
      // Return a 404 handler when Better Auth is not the configured provider
      return {
        GET: async () => new Response("Not Found", { status: 404 }),
        POST: async () => new Response("Not Found", { status: 404 }),
      };
    }

    // Dynamically import to avoid loading Better Auth when not needed
    const { getBetterAuthInstance } = await import("@/lib/auth/better-auth-server");
    const { toNextJsHandler } = await import("better-auth/next-js");
    handler = toNextJsHandler(getBetterAuthInstance());
  }
  return handler;
}

/**
 * Checks if this is an OAuth callback request
 */
function isOAuthCallback(req: NextRequest): boolean {
  const url = new URL(req.url);
  return url.pathname.startsWith("/api/auth/callback/");
}

/**
 * Handles OAuth callback responses to check for multi-org scenarios.
 * If user has multiple orgs and no activeOrganizationId, redirect to org selection.
 */
async function handleOAuthCallbackResponse(
  req: NextRequest,
  response: Response,
): Promise<Response> {
  // Only intercept redirects (302/303)
  if (response.status !== 302 && response.status !== 303) {
    return response;
  }

  // Get the session token from the response cookies
  const setCookieHeader = response.headers.get("set-cookie");
  if (!setCookieHeader) {
    return response;
  }

  // Parse the session token from Set-Cookie header
  const sessionTokenMatch = setCookieHeader.match(/better-auth\.session_token=([^;]+)/);
  if (!sessionTokenMatch) {
    return response;
  }

  const sessionToken = decodeURIComponent(sessionTokenMatch[1]);

  try {
    // Validate the session to get user info
    const { getBetterAuthInstance } = await import("@/lib/auth/better-auth-server");
    const auth = getBetterAuthInstance();

    const sessionResult = await auth.api.getSession({
      headers: {
        cookie: `better-auth.session_token=${sessionToken}`,
      },
    });

    if (!sessionResult?.session || !sessionResult?.user) {
      return response;
    }

    const { session, user } = sessionResult;

    // If activeOrganizationId is already set, let the normal flow continue
    if (session.activeOrganizationId) {
      return response;
    }

    // Check user's org memberships
    const memberships = await db.query.baMember.findMany({
      where: eq(schema.baMember.userId, user.id),
    });

    if (memberships.length === 0) {
      // No orgs - redirect to /new for workspace creation
      const redirectUrl = new URL("/new", req.url);
      const newResponse = NextResponse.redirect(redirectUrl);
      
      // Copy the session cookie from the original response
      newResponse.headers.set("set-cookie", setCookieHeader);
      
      return newResponse;
    }

    if (memberships.length === 1) {
      // Single org - set it as active and continue
      // The TRPC context will handle this case
      return response;
    }

    // Multiple orgs - redirect to org selection
    const orgIds = memberships.map((m) => m.organizationId);
    const orgs = await db.query.baOrganization.findMany({
      where: (table, { inArray }) => inArray(table.id, orgIds),
    });

    const orgsParam = encodeURIComponent(
      JSON.stringify(orgs.map((o) => ({ id: o.id, name: o.name }))),
    );

    const redirectUrl = new URL(`/auth/sign-in?orgs=${orgsParam}`, req.url);
    const newResponse = NextResponse.redirect(redirectUrl);

    // Set the pending session cookie for org selection
    // This is the ONLY cookie we set - we intentionally do NOT set the main session cookie
    // because the user hasn't selected an org yet. The main session cookie will be set
    // after org selection in completeOrgSelection.
    const cookieOptions = getAuthCookieOptions();
    newResponse.cookies.set(PENDING_SESSION_COOKIE, sessionToken, {
      ...cookieOptions,
      maxAge: 60 * 10, // 10 minutes
    });

    return newResponse;
  } catch (error) {
    console.error("[OAuth Callback] Error handling multi-org check:", error);
    // On error, let the normal flow continue
    return response;
  }
}

export async function GET(req: NextRequest) {
  const h = await getHandler();
  const response = await h.GET(req);

  // For OAuth callbacks, check for multi-org scenarios
  if (isOAuthCallback(req)) {
    return handleOAuthCallbackResponse(req, response);
  }

  return response;
}

export async function POST(req: NextRequest) {
  const h = await getHandler();
  return h.POST(req);
}
