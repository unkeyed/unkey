// Server-only cookie helpers. This module is intentionally NOT a "use server"
// module: that directive would expose every export below as a public POST
// endpoint identified by an action ID, including the generic setCookie /
// setCookies helpers that accept arbitrary name, value, and option fields.
// Client components must go through the narrow wrappers in ./cookies-actions.

import { cookies } from "next/headers";
import type { NextRequest, NextResponse } from "next/server";
import { getAuthCookieOptions, getDefaultCookieOptions } from "./cookie-security";
import { UNKEY_LAST_ORG_COOKIE, UNKEY_SESSION_COOKIE } from "./types";

export interface CookieOptions {
  httpOnly: boolean;
  secure: boolean;
  sameSite: "strict" | "lax" | "none";
  path?: string;
  maxAge?: number;
  expiresAt?: Date;
  domain?: string;
}

export interface Cookie {
  name: string;
  value: string;
  options?: CookieOptions;
}

/**
 * Get a cookie value by name
 */
export async function getCookie(name: string, request?: NextRequest): Promise<string | null> {
  const cookieStore = request?.cookies || (await cookies());
  return cookieStore.get(name)?.value ?? null;
}

/**
 * Set a cookie with the given name, value, and options
 */
export async function setCookie(cookie: Cookie): Promise<void> {
  const cookieStore = await cookies();
  cookieStore.set(cookie.name, cookie.value, cookie.options);
}

/**
 * Set multiple cookies at once
 */
export async function setCookies(cookieList: Cookie[]): Promise<void> {
  const cookieStore = await cookies();
  for (const cookie of cookieList) {
    cookieStore.set(cookie.name, cookie.value, cookie.options);
  }
}

/**
 * Delete a cookie by name
 */
export async function deleteCookie(name: string): Promise<void> {
  const cookieStore = await cookies();
  cookieStore.delete(name);
}

/**
 * Update or clear a secure HTTP-only cookie with optional deletion logging
 * @param cookieName - Name of the cookie to update/clear
 * @param value - Value to set (if null/undefined, cookie will be deleted)
 * @param reason - Optional reason for deletion (will be logged)
 */
export async function updateCookie(
  cookieName: string,
  value: string | null | undefined,
  reason?: string,
): Promise<void> {
  if (value) {
    await setCookie({
      name: cookieName,
      value: value,
      options: {
        ...getDefaultCookieOptions(),
      },
    });
    return;
  }

  if (reason) {
    await deleteCookie(cookieName);
  }
}

/**
 * Set cookies on a NextResponse object
 * Useful when you need to set cookies during a redirect
 */
export async function setCookiesOnResponse(
  response: NextResponse,
  cookieList: Cookie[],
): Promise<NextResponse> {
  for (const cookie of cookieList) {
    response.cookies.set(cookie.name, cookie.value, cookie.options);
  }
  return response;
}

/**
 * Encapsulates the logic for the primary session cookie required for auth functionality
 * @param params
 */
export async function setSessionCookie(params: {
  token: string;
  expiresAt: Date;
}): Promise<void> {
  const { token, expiresAt } = params;

  // The session cookie must always be SameSite=Lax, matching how sign-in
  // issues it. Strict would not be sent on cross-site top-level navigations
  // back into the app (OAuth callbacks, GitHub App install returns), which
  // makes the user appear logged out on those requests.
  await setCookie({
    name: UNKEY_SESSION_COOKIE,
    value: token,
    options: {
      ...getAuthCookieOptions(),
      maxAge: Math.floor((expiresAt.getTime() - Date.now()) / 1000),
    },
  });
}

/**
 * Encapsulates the logic for storing the last used organization ID in a cookie
 * This cookie is used for auto-selection on next login
 * @param params
 */
export async function setLastUsedOrgCookie(params: {
  orgId: string;
}): Promise<void> {
  const { orgId } = params;

  await setCookie({
    name: UNKEY_LAST_ORG_COOKIE,
    value: orgId,
    options: {
      httpOnly: false, // Allow client-side access
      secure: true,
      sameSite: "strict",
      path: "/",
      maxAge: 60 * 60 * 24 * 30, // 30 Days
    },
  });
}

export async function getCookieOptionsAsString(
  options: Partial<CookieOptions> = {},
): Promise<string> {
  // Set defaults if not provided
  const defaultOptions: CookieOptions = getDefaultCookieOptions();

  // Merge defaults with provided options
  const mergedOptions = { ...defaultOptions, ...options };

  let cookieString = `Path=${mergedOptions.path}`;

  if (mergedOptions.httpOnly) {
    cookieString += "; HttpOnly";
  }

  if (mergedOptions.secure) {
    cookieString += "; Secure";
  }

  if (mergedOptions.sameSite) {
    const capitalizedSameSite =
      mergedOptions.sameSite.charAt(0).toUpperCase() + mergedOptions.sameSite.slice(1);
    cookieString += `; SameSite=${capitalizedSameSite}`;
  }

  if (mergedOptions.maxAge !== undefined) {
    cookieString += `; Max-Age=${mergedOptions.maxAge}`;
  } else if (mergedOptions.expiresAt) {
    cookieString += `; Expires=${mergedOptions.expiresAt.toUTCString()}`;
  }

  if (mergedOptions.domain) {
    cookieString += `; Domain=${mergedOptions.domain}`;
  }

  return cookieString;
}
