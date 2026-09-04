"use server";

import type { NextRequest } from "next/server";
import { getCookie, setSessionCookie } from "./cookies";
import { localAuth } from "./local";
import { UNKEY_SESSION_COOKIE, type User } from "./types";

type LocalSessionResult = {
  session: {
    userId: string;
    orgId: string | null;
    permissions?: readonly string[];
    role: string | null;
    user?: User | null;
  } | null;
};

export async function updateLocalSession(request?: NextRequest): Promise<LocalSessionResult> {
  const localSessionToken = "local_session_token";

  const existingSession = await getCookie(UNKEY_SESSION_COOKIE, request);
  if (!existingSession) {
    const expiresAt = new Date();
    expiresAt.setFullYear(expiresAt.getFullYear() + 10);

    try {
      await setSessionCookie({ token: localSessionToken, expiresAt });
    } catch {
      // Server Components cannot mutate cookies. Local mode still uses its
      // deterministic session for this request.
    }
  }

  const result = await localAuth.validateSession(localSessionToken);
  if (!result.isValid || !result.userId) {
    return { session: null };
  }

  return {
    session: {
      userId: result.userId,
      orgId: result.orgId ?? null,
      permissions: result.permissions,
      role: result.role ?? null,
      user: result.user ?? null,
    },
  };
}
