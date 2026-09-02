// Server-only cookie helpers. This module is intentionally NOT a "use server"
// module: that directive would expose every export below as a public POST
// endpoint identified by an action ID, including the generic setCookie /
// setCookies helpers that accept arbitrary name, value, and option fields.
// Client components must go through the narrow wrappers in ./cookies-actions.

import * as Sentry from "@sentry/nextjs";
import { cookies } from "next/headers";
import type { NextRequest, NextResponse } from "next/server";
import { getAuthCookieOptions, getDefaultCookieOptions } from "./cookie-security";
import { UNKEY_LAST_ORG_COOKIE, UNKEY_SESSION_COOKIE } from "./types";

// Browsers cap one cookie at 4096 bytes, while Vercel caps the combined Cookie
// request header at 16KB. Four parts leave space for names and other cookies.
const SESSION_COOKIE_CHUNK_SIZE = 3500;
const MAX_SESSION_COOKIE_CHUNKS = 4;

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

// sessionCookieChunkName keeps chunk naming identical across every cookie API.
function sessionCookieChunkName(index: number): string {
  return `${UNKEY_SESSION_COOKIE}.${index}`;
}

// getCookieUpdates converts one logical session into browser-safe cookie writes.
// The final expired part stops readers before any stale parts from an older session.
function getCookieUpdates(cookie: Cookie): Cookie[] {
  if (cookie.name !== UNKEY_SESSION_COOKIE) {
    return [cookie];
  }

  const options = cookie.options ?? getAuthCookieOptions();
  const sessionCookie: Cookie = {
    ...cookie,
    options,
  };
  const expired: Cookie = {
    ...sessionCookie,
    value: "",
    options: { ...options, maxAge: 0 },
  };

  if (sessionCookie.value.length <= SESSION_COOKIE_CHUNK_SIZE) {
    return [sessionCookie, { ...expired, name: sessionCookieChunkName(0) }];
  }

  const chunks = [];
  for (let offset = 0; offset < sessionCookie.value.length; offset += SESSION_COOKIE_CHUNK_SIZE) {
    chunks.push(sessionCookie.value.slice(offset, offset + SESSION_COOKIE_CHUNK_SIZE));
  }

  if (chunks.length > MAX_SESSION_COOKIE_CHUNKS) {
    Sentry.captureMessage("WorkOS session exceeds Vercel cookie capacity", {
      level: "error",
      fingerprint: ["workos-session-cookie-capacity-exceeded"],
      tags: { component: "authentication" },
      extra: {
        sessionSizeBytes: sessionCookie.value.length,
        chunkCount: chunks.length,
        maxChunkCount: MAX_SESSION_COOKIE_CHUNKS,
      },
    });
    throw new Error("Session is too large to store in cookies");
  }

  return [
    expired,
    ...chunks.map((value, index) => ({
      ...sessionCookie,
      name: sessionCookieChunkName(index),
      value,
    })),
    { ...expired, name: sessionCookieChunkName(chunks.length) },
  ];
}

/**
 * Get a cookie value by name
 */
export async function getCookie(name: string, request?: NextRequest): Promise<string | null> {
  const cookieStore = request?.cookies || (await cookies());
  const value = cookieStore.get(name)?.value;
  if (value || name !== UNKEY_SESSION_COOKIE) {
    return value ?? null;
  }

  const chunks = [];
  for (let index = 0; index < MAX_SESSION_COOKIE_CHUNKS; index++) {
    const chunk = cookieStore.get(sessionCookieChunkName(index))?.value;
    if (!chunk) {
      break;
    }
    chunks.push(chunk);
  }
  return chunks.length > 0 ? chunks.join("") : null;
}

/**
 * Set a cookie with the given name, value, and options
 */
export async function setCookie(cookie: Cookie): Promise<void> {
  const cookieStore = await cookies();
  for (const update of getCookieUpdates(cookie)) {
    cookieStore.set(update.name, update.value, update.options);
  }
}

/**
 * Set multiple cookies at once
 */
export async function setCookies(cookieList: Cookie[]): Promise<void> {
  const cookieStore = await cookies();
  for (const cookie of cookieList) {
    for (const update of getCookieUpdates(cookie)) {
      cookieStore.set(update.name, update.value, update.options);
    }
  }
}

/**
 * Delete a cookie by name
 */
export async function deleteCookie(name: string): Promise<void> {
  const cookieStore = await cookies();
  cookieStore.delete(name);
  if (name === UNKEY_SESSION_COOKIE) {
    for (let index = 0; index <= MAX_SESSION_COOKIE_CHUNKS; index++) {
      cookieStore.delete(sessionCookieChunkName(index));
    }
  }
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
    for (const update of getCookieUpdates(cookie)) {
      response.cookies.set(update.name, update.value, update.options);
    }
  }
  return response;
}

/** Serializes all browser-safe writes for one logical cookie. */
export async function getSetCookieHeaders(cookie: Cookie): Promise<string[]> {
  return Promise.all(
    getCookieUpdates(cookie).map(
      async (update) =>
        `${update.name}=${update.value}; ${await getCookieOptionsAsString(update.options)}`,
    ),
  );
}

/**
 * Encapsulates the logic for the primary session cookie required for auth functionality
 * @param params
 */
export async function setSessionCookie(params: { token: string; expiresAt: Date }): Promise<void> {
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
export async function setLastUsedOrgCookie(params: { orgId: string }): Promise<void> {
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
