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
  scopes: string[];
  preview: boolean;
  expiresAt: number;
};

/**
 * The session `scopes` column holds the grant `portal.createSession` persists:
 * `{ keyspaceIds, scopes: ["keys:read", ...] }`. The portal only needs the
 * capability list for tab/visibility decisions, so normalize to that array
 * here. Tolerates a plain string array too, in case the shape changes.
 */
function readScopes(raw: unknown): string[] {
  if (Array.isArray(raw)) {
    return raw.filter((s): s is string => typeof s === "string");
  }
  if (raw && typeof raw === "object" && "scopes" in raw) {
    const inner = (raw as { scopes?: unknown }).scopes;
    if (Array.isArray(inner)) {
      return inner.filter((s): s is string => typeof s === "string");
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
 * Exchange a single-use code for a 24-hour access token. Sets an httpOnly
 * cookie on success. The token is never returned to the caller.
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
          ? "Request timed out. Please try again."
          : "Network error. Please check your connection and try again.";
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
        error: "Received an unexpected response. Please try again.",
      };
    }

    const parsed = exchangeResponseSchema.safeParse(body);
    if (!parsed.success) {
      return {
        success: false,
        error: "Received an unexpected response. Please try again.",
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
 * Load the session + portal config from the database. Returns both in one
 * server function call so the route only needs a single round-trip.
 */
export const getSessionWithConfig = createServerFn({ method: "GET" }).handler(
  async (): Promise<{
    session: SessionData;
    config: Portal | null;
    logsRetentionDays: number;
  } | null> => {
    const accessToken = getCookie(SESSION_COOKIE_NAME);
    if (!accessToken) {
      return null;
    }

    // Dynamic import to keep mysql2/drizzle out of the client bundle.
    const { db } = await import("./db");
    const { loadPortal } = await import("./portal");

    // The column stores only the hash, so the lookup hashes the cookie. This is
    // the same helper that hashes API keys, and its test suite pins it to the
    // Go implementation the API writes with.
    const accessTokenHash = await sha256(accessToken);

    const nowMs = Date.now();
    const session = await db.query.portalSessions.findFirst({
      where: (t, { eq }) => eq(t.accessTokenHash, accessTokenHash),
      columns: {
        id: true,
        workspaceId: true,
        portalId: true,
        externalId: true,
        scopes: true,
        preview: true,
        accessTokenExpiresAt: true,
        revokedAt: true,
      },
    });

    if (!session) {
      return null;
    }

    // State is derived from the row against the current clock, mirroring the
    // API's session state helper: revocation takes precedence over expiry, and
    // a missing expiry is treated as unusable rather than unbounded.
    if (session.revokedAt !== null) {
      return null;
    }
    if (session.accessTokenExpiresAt === null || session.accessTokenExpiresAt <= nowMs) {
      return null;
    }

    const [config, logsRetentionDays] = await Promise.all([
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

    return {
      session: {
        id: session.id,
        portalId: session.portalId,
        externalId: session.externalId,
        scopes: readScopes(session.scopes),
        preview: session.preview,
        expiresAt: session.accessTokenExpiresAt,
      },
      config,
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
