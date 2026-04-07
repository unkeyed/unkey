import { createServerFn } from "@tanstack/react-start";
import { getCookie, setCookie, deleteCookie } from "@tanstack/react-start/server";
import { env } from "./env";

const SESSION_COOKIE_NAME = "portal_session";
const SESSION_COOKIE_MAX_AGE_SECONDS = 24 * 60 * 60; // 24 hours

type ExchangeResult =
  | { success: true; token: string }
  | { success: false; error: string };

/**
 * Exchange a short-lived session ID for a long-lived browser session token.
 * Sets an httpOnly cookie on success.
 */
export const exchangeSession = createServerFn({ method: "POST" })
  .inputValidator((d: string) => d)
  .handler(async ({ data: sessionId }): Promise<ExchangeResult> => {
    const apiUrl = env().UNKEY_API_URL;

    // TODO: Replace with @unkey/api SDK call once portal endpoints are added to the public OpenAPI spec
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
 * Read the portal session token from the cookie.
 */
export const getSessionToken = createServerFn({ method: "GET" }).handler(
  async (): Promise<string | null> => {
    return getCookie(SESSION_COOKIE_NAME) ?? null;
  },
);

/**
 * Clear the portal session cookie.
 */
export const clearSession = createServerFn({ method: "POST" }).handler(
  async () => {
    deleteCookie(SESSION_COOKIE_NAME);
  },
);
