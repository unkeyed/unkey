"use server";

import { env } from "@/lib/env";
import type { NextRequest } from "next/server";
import { getCookie, getCookieOptionsAsString, setSessionCookie } from "./cookies";
import { auth } from "./server";
import { LOCAL_ORG_ID, LOCAL_ORG_ROLE, LOCAL_USER_ID, UNKEY_SESSION_COOKIE } from "./types";

type SessionResult = {
  session: {
    userId: string;
    orgId: string | null;
    role: string | null;
    impersonator?: {
      email: string;
      reason?: string | null;
    };
  } | null;
  headers: Headers;
};

export async function updateSession(request?: NextRequest): Promise<SessionResult> {
  const UNKEY_SESSION_HEADER = `x-${UNKEY_SESSION_COOKIE}`;
  const headers = new Headers();
  const environment = env();

  // Remove any lingering session headers
  headers.delete(UNKEY_SESSION_HEADER);

  // Check if we're using local auth provider
  if (environment.AUTH_PROVIDER === "local") {
    // Generate a local session token that doesn't expire
    const localSessionToken = "local_session_token";

    // Set the header for internal use
    headers.set(UNKEY_SESSION_HEADER, localSessionToken);

    // set the cookie for local auth
    const sessionToken = await getCookie(UNKEY_SESSION_COOKIE, request);
    if (!sessionToken) {
      try {
        const expiresAt = new Date();
        expiresAt.setFullYear(expiresAt.getFullYear() + 10);

        await setSessionCookie({ token: localSessionToken, expiresAt });
      } catch (_error) {
        headers.append(
          "Set-Cookie",
          `${UNKEY_SESSION_COOKIE}=${localSessionToken}; Path=/; SameSite=Strict; Max-Age=${60 * 60 * 24 * 365 * 10}`,
        );
      }
    }

    return {
      session: {
        userId: LOCAL_USER_ID,
        orgId: LOCAL_ORG_ID,
        role: LOCAL_ORG_ROLE,
      },
      headers,
    };
  }
  try {
    // Get session token from cookie - handle case when request is undefined
    let sessionToken: string | null | undefined;

    if (request) {
      // If we have a request object, use it to get the cookie
      sessionToken = await getCookie(UNKEY_SESSION_COOKIE, request);
    } else {
      // If no request, try to get the cookie from the client-side or server components
      try {
        sessionToken = await getCookie(UNKEY_SESSION_COOKIE);
      } catch (_error) {
        // If getCookie throws when no request is provided, handle gracefully
        sessionToken = undefined;
      }
    }

    if (!sessionToken) {
      return { session: null, headers };
    }

    try {
      // Validate the session
      const sessionValidationResult = await auth.validateSession(sessionToken);

      if (sessionValidationResult.isValid && sessionValidationResult.userId) {
        headers.set(UNKEY_SESSION_HEADER, sessionToken);

        return {
          session: {
            userId: sessionValidationResult.userId,
            orgId: sessionValidationResult.orgId ?? null,
            role: sessionValidationResult.role ?? null,
            impersonator: sessionValidationResult.impersonator,
          },
          headers,
        };
      }

      // If session needs refreshing
      if (sessionValidationResult.shouldRefresh) {
        try {
          const refreshedSession = await auth.refreshSession(sessionToken);

          // Use different methods to set cookies based on whether we have a request
          if (request) {
            // For middleware/trpc routes with request object
            headers.append(
              "Set-Cookie",
              `${UNKEY_SESSION_COOKIE}=${refreshedSession.newToken}; ${await getCookieOptionsAsString({ expiresAt: refreshedSession.expiresAt })}`,
            );
          } else {
            // For client-side or RSC or when no request is available
            // Only use cookies() API when NOT in middleware context
            try {
              await setSessionCookie({
                token: refreshedSession.newToken,
                expiresAt: refreshedSession.expiresAt,
              });
            } catch (_cookieError) {
              // Fall back to headers approach if cookie setting fails
              headers.append(
                "Set-Cookie",
                `${UNKEY_SESSION_COOKIE}=${refreshedSession.newToken}; ${await getCookieOptionsAsString({ expiresAt: refreshedSession.expiresAt })}`,
              );
            }
          }

          headers.set(UNKEY_SESSION_HEADER, refreshedSession.newToken);

          if (refreshedSession.session) {
            return {
              session: {
                userId: refreshedSession.session?.userId,
                orgId: refreshedSession.session?.orgId ?? null,
                role: refreshedSession.session?.role ?? null,
                impersonator: refreshedSession.impersonator,
              },
              headers,
            };
          }
        } catch (refreshError) {
          console.error("Failed to refresh session:", refreshError);
          // If refresh fails, treat as no session
          return { session: null, headers };
        }
      }

      // Session is neither valid nor refreshable
      return { session: null, headers };
    } catch (validationError) {
      console.error("Session validation failed:", validationError);
      return { session: null, headers };
    }
  } catch (error) {
    console.error("Error in updateSession:", error);
    return { session: null, headers };
  }
}
