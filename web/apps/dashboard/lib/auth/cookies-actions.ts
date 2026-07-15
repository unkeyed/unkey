"use server";

// Narrow Server Actions exposed to client components.
//
// Files marked "use server" expose every export as a public POST endpoint
// identified by an action ID. The generic helpers in ./cookies.ts (setCookie,
// setCookies, setCookiesOnResponse, deleteCookie) accept arbitrary name,
// value, and option fields, which makes them unsafe to expose. This module
// only re-exports the narrow, fixed-name operations that client components
// actually need.

import {
  getCookie as getCookieInternal,
  setLastUsedOrgCookie as setLastUsedOrgCookieInternal,
  setSessionCookie as setSessionCookieInternal,
} from "./cookies";
import { PENDING_SESSION_COOKIE, UNKEY_LAST_ORG_COOKIE } from "./types";

// Cookies the client may read by name. PENDING_SESSION_COOKIE backs the
// org-selection step before a full session exists, and UNKEY_LAST_ORG_COOKIE
// stores the last selected workspace for auto-selection on next sign-in.
function isReadableCookieName(name: string): boolean {
  return name === PENDING_SESSION_COOKIE || name === UNKEY_LAST_ORG_COOKIE;
}

export async function getCookie(name: string): Promise<string | null> {
  if (!isReadableCookieName(name)) {
    throw new Error(`Cookie ${name} is not readable from the client`);
  }
  return getCookieInternal(name);
}

export async function setSessionCookie(params: {
  token: string;
  expiresAt: Date;
}): Promise<void> {
  // The token is validated by the WorkOS sealed-session check on the next
  // request that reads the session cookie; an attacker who calls this action
  // can only plant a value into their own browser, where it will fail
  // session validation. The internal helper hardcodes the cookie name, so
  // this action cannot be misused to set arbitrary cookies.
  await setSessionCookieInternal(params);
}

export async function setLastUsedOrgCookie(params: {
  orgId: string;
}): Promise<void> {
  await setLastUsedOrgCookieInternal(params);
}
