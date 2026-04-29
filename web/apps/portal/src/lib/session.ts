import { createServerFn } from "@tanstack/react-start";
import { deleteCookie, getCookie, setCookie } from "@tanstack/react-start/server";
import { z } from "zod";
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

type ExchangeResult = { success: true } | { success: false; error: string };

const exchangeResponseSchema = z.object({
  data: z.object({
    token: z.string().min(1),
    expiresAt: z.number(),
  }),
});

/**
 * Exchange a short-lived session ID for a long-lived browser session token.
 * Sets an httpOnly cookie on success. The token is never returned to the caller.
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

    const parsed = exchangeResponseSchema.safeParse(await response.json());
    if (!parsed.success) {
      return {
        success: false,
        error: "Received an unexpected response. Please try again.",
      };
    }

    setCookie(SESSION_COOKIE_NAME, parsed.data.data.token, {
      httpOnly: true,
      secure: process.env.NODE_ENV === "production",
      sameSite: "strict",
      path: "/",
      maxAge: SESSION_COOKIE_MAX_AGE_SECONDS,
    });

    return { success: true };
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
