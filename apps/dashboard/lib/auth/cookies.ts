"use server";

import { cookies } from "next/headers";
import type { NextRequest, NextResponse } from "next/server";
import { UNKEY_SESSION_COOKIE } from "./types";

export interface CookieOptions {
  secure: boolean;
  httpOnly: boolean;
  sameSite: "lax" | "strict" | "none";
  path?: string;
  maxAge?: number;
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
  const cookieStore = request?.cookies || cookies();
  return cookieStore.get(name)?.value ?? null;
}

/**
 * Set a cookie with the given name, value, and options
 */
export async function setCookie(cookie: Cookie): Promise<void> {
  const cookieStore = cookies();
  cookieStore.set(cookie.name, cookie.value, cookie.options);
}

/**
 * Set multiple cookies at once
 */
export async function setCookies(cookieList: Cookie[]): Promise<void> {
  const cookieStore = cookies();
  for (const cookie of cookieList) {
    cookieStore.set(cookie.name, cookie.value, cookie.options);
  }
}

/**
 * Delete a cookie by name
 */
export async function deleteCookie(name: string): Promise<void> {
  const cookieStore = cookies();
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
        httpOnly: true,
        secure: true,
        sameSite: "lax",
      },
    });
    return;
  }

  if (reason) {
    console.error("Session refresh failed:", reason);
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
export async function SetSessionCookie(params: {
  token: string;
  expiresAt: Date;
}): Promise<void> {
  const { token, expiresAt } = params;

  await setCookie({
    name: UNKEY_SESSION_COOKIE,
    value: token,
    options: {
      httpOnly: true,
      secure: true,
      sameSite: "lax",
      path: "/",
      maxAge: Math.floor((expiresAt.getTime() - Date.now()) / 1000),
    },
  });
}
