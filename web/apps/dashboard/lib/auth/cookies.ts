import { cookies } from "next/headers";
import type { NextRequest } from "next/server";
import { getAuthCookieOptions } from "./cookie-security";
import { UNKEY_LAST_ORG_COOKIE, UNKEY_SESSION_COOKIE } from "./types";

export async function getCookie(name: string, request?: NextRequest): Promise<string | null> {
  const cookieStore = request?.cookies ?? (await cookies());
  return cookieStore.get(name)?.value ?? null;
}

export async function deleteCookie(name: string): Promise<void> {
  (await cookies()).delete(name);
}

export async function setSessionCookie(params: {
  token: string;
  expiresAt: Date;
}): Promise<void> {
  (await cookies()).set(UNKEY_SESSION_COOKIE, params.token, {
    ...getAuthCookieOptions(),
    maxAge: Math.floor((params.expiresAt.getTime() - Date.now()) / 1000),
  });
}

export async function setLastUsedOrgCookie(params: { orgId: string }): Promise<void> {
  (await cookies()).set(UNKEY_LAST_ORG_COOKIE, params.orgId, {
    httpOnly: false,
    secure: true,
    sameSite: "strict",
    path: "/",
    maxAge: 60 * 60 * 24 * 30,
  });
}
