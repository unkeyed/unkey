import { createServerFn } from "@tanstack/react-start";
import { deleteCookie, getCookie, setCookie } from "@tanstack/react-start/server";
import { sha256 } from "@unkey/hash";
import { z } from "zod";
import { env } from "./env";
import type { Portal } from "./portal";

export const SESSION_COOKIE_NAME = "portal_session";
const SESSION_COOKIE_MAX_AGE_SECONDS = 24 * 60 * 60; // 24 hours

export type SessionData = {
  id: string;
  portalId: string;
  externalId: string;
  permissions: string[];
  expiresAt: number;
};

/**
 * The session `permissions` column holds the grant `portal.createSession`
 * persists: `{ keyspaceIds, permissions: ["keys:read", ...] }`. The portal only
 * needs the capability list for tab/visibility decisions, so normalize to that
 * array here. Tolerates a plain string array too, in case the shape changes.
 */
function readCapabilities(raw: unknown): string[] {
  if (Array.isArray(raw)) {
    return raw.filter((p): p is string => typeof p === "string");
  }
  if (raw && typeof raw === "object" && "permissions" in raw) {
    const inner = (raw as { permissions?: unknown }).permissions;
    if (Array.isArray(inner)) {
      return inner.filter((p): p is string => typeof p === "string");
    }
  }
  return [];
}

type ExchangeResult = { success: true } | { success: false; error: string };

const exchangeResponseSchema = z.object({
  data: z.object({
    accessToken: z.string().min(1),
    expiresAt: z.number(),
  }),
});

/**
 * Redeem a short-lived exchange code for an access token.
 * Sets an httpOnly cookie on success. The access token is never returned to the caller.
 */
export const exchangeCode = createServerFn({ method: "POST" })
  .inputValidator((d: string) => d)
  .handler(async ({ data: code }: { data: string }): Promise<ExchangeResult> => {
    const apiUrl = env().UNKEY_API_URL;

    const controller = new AbortController();
    const timeoutId = setTimeout(() => controller.abort(), 10_000);

    let response: Response;
    try {
      response = await fetch(`${apiUrl}/v2/portal.exchangeCode`, {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ code }),
        signal: controller.signal,
      });
    } catch (err) {
      const message =
        err instanceof DOMException && err.name === "AbortError"
          ? "Request timed out. Please request a new session from your application."
          : "Network error. Please request a new session from your application.";
      return { success: false, error: message };
    } finally {
      clearTimeout(timeoutId);
    }

    if (!response.ok) {
      return {
        success: false,
        error: "Session expired or invalid. Please request a new session from your application.",
      };
    }

    let body: unknown;
    try {
      body = await response.json();
    } catch {
      return {
        success: false,
        error: "Received an unexpected response. Please request a new session.",
      };
    }

    const parsed = exchangeResponseSchema.safeParse(body);
    if (!parsed.success) {
      return {
        success: false,
        error: "Received an unexpected response. Please request a new session.",
      };
    }

    setCookie(SESSION_COOKIE_NAME, parsed.data.data.accessToken, {
      httpOnly: true,
      secure: process.env.NODE_ENV === "production",
      sameSite: "strict",
      path: "/",
      maxAge: SESSION_COOKIE_MAX_AGE_SECONDS,
    });

    return { success: true };
  });

/**
 * Load the session and portal from the database. Returns both in one
 * server function call so the route only needs a single round-trip.
 */
export const getSessionWithPortal = createServerFn({ method: "GET" }).handler(
  async (): Promise<{
    session: SessionData;
    portal: Portal | null;
    logsRetentionDays: number;
  } | null> => {
    const accessToken = getCookie(SESSION_COOKIE_NAME);
    if (!accessToken) {
      return null;
    }

    // Dynamic import to keep mysql2/drizzle out of the client bundle.
    const { db } = await import("./db");
    const { loadPortal } = await import("./portal");

    const nowMs = Date.now();
    const accessTokenHash = await sha256(accessToken);
    const session = await db.query.portalSessions.findFirst({
      where: (t, { eq, gt, and }) =>
        and(eq(t.accessTokenHash, accessTokenHash), gt(t.accessTokenExpiresAt, nowMs)),
      columns: {
        id: true,
        workspaceId: true,
        portalId: true,
        externalId: true,
        permissions: true,
        accessTokenExpiresAt: true,
      },
    });

    if (!session || session.accessTokenExpiresAt === null) {
      return null;
    }

    const [portal, logsRetentionDays] = await Promise.all([
      loadPortal(session.portalId).catch((err) => {
        console.error("Failed to load portal", {
          portalId: session.portalId,
          err,
        });
        return null;
      }),
      // The workspace's log retention bounds how far back analytics can query;
      // the analytics page uses it to only offer periods within retention. A
      // failed/missing lookup falls back to 0 ("unknown"), which the UI treats
      // as uncapped rather than blocking the page.
      db.query.limits
        .findFirst({
          where: (t, { eq }) => eq(t.workspaceId, session.workspaceId),
          columns: { logsRetentionDaysMax: true },
        })
        .then((limits) => limits?.logsRetentionDaysMax ?? 0)
        .catch((err) => {
          console.error("Failed to load workspace limits", {
            workspaceId: session.workspaceId,
            err,
          });
          return 0;
        }),
    ]);

    // workspaceId is only needed server-side (limits lookup above); keep it off
    // the client-facing session, which stays as SessionData.
    const { workspaceId: _workspaceId, accessTokenExpiresAt, ...sessionColumns } = session;
    return {
      session: {
        ...sessionColumns,
        permissions: readCapabilities(session.permissions),
        expiresAt: accessTokenExpiresAt,
      },
      portal,
      logsRetentionDays,
    };
  },
);

/**
 * Clear the portal session cookie.
 */
export const clearSession = createServerFn({ method: "POST" }).handler(async () => {
  deleteCookie(SESSION_COOKIE_NAME);
});
