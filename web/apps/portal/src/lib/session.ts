import { createServerFn } from "@tanstack/react-start";
import { deleteCookie, getCookie, setCookie } from "@tanstack/react-start/server";
import { env } from "./env";
import type { PortalConfig } from "./portal-config";

const SESSION_COOKIE_NAME = "portal_session";
const SESSION_COOKIE_MAX_AGE_SECONDS = 24 * 60 * 60; // 24 hours

export type SessionData = {
  id: string;
  portalConfigId: string;
  externalId: string;
  permissions: string[];
  preview: boolean;
  expiresAt: number;
};

type ExchangeResult = { success: true; token: string } | { success: false; error: string };

/**
 * Exchange a short-lived session ID for a long-lived browser session token.
 * Sets an httpOnly cookie on success.
 */
export const exchangeSession = createServerFn({ method: "POST" })
  .inputValidator((d: string) => d)
  .handler(async ({ data: sessionId }: { data: string }): Promise<ExchangeResult> => {
    const apiUrl = env().UNKEY_API_URL;

    const response = await fetch(`${apiUrl}/v2/portal.exchangeSession`, {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify({ sessionId }),
    });

    if (!response.ok) {
      return {
        success: false,
        error: "Session expired or invalid. Please request a new session from your application.",
      };
    }

    const body = await response.json();
    const data: { token: string; expiresAt: number } = body.data;

    setCookie(SESSION_COOKIE_NAME, data.token, {
      httpOnly: true,
      secure: process.env.NODE_ENV === "production",
      sameSite: "strict",
      path: "/",
      maxAge: SESSION_COOKIE_MAX_AGE_SECONDS,
    });

    return { success: true, token: data.token };
  });

/**
 * Load the session + portal config from the database. Returns both in one
 * server function call so the route only needs a single round-trip.
 */
export const getSessionWithConfig = createServerFn({ method: "GET" }).handler(
  async (): Promise<{ session: SessionData; config: PortalConfig | null } | null> => {
    const token = getCookie(SESSION_COOKIE_NAME);
    if (!token) {
      return null;
    }

    // Dynamic import to keep mysql2/drizzle out of the client bundle.
    const { db } = await import("./db");
    const { loadPortalConfig } = await import("./portal-config");

    const nowMs = Date.now();
    const session = await db.query.portalSessions.findFirst({
      where: (t, { eq, gt, and }) => and(eq(t.id, token), gt(t.expiresAt, nowMs)),
      columns: {
        id: true,
        portalConfigId: true,
        externalId: true,
        permissions: true,
        preview: true,
        expiresAt: true,
      },
    });

    if (!session) {
      return null;
    }

    let config: PortalConfig | null = null;
    try {
      config = await loadPortalConfig(session.portalConfigId);
    } catch (err) {
      console.error("Failed to load portal config", {
        portalConfigId: session.portalConfigId,
        err,
      });
    }

    return { session, config };
  },
);

/**
 * Clear the portal session cookie.
 */
export const clearSession = createServerFn({ method: "POST" }).handler(async () => {
  deleteCookie(SESSION_COOKIE_NAME);
});
