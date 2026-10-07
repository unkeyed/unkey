import type { NextRequest } from "next/server";
import { UNKEY_SESSION_COOKIE } from "./types";

export function expireLegacySession(request: NextRequest, response: Response): Response {
  if (!request.cookies.has(UNKEY_SESSION_COOKIE)) {
    return response;
  }
  response.headers.append(
    "Set-Cookie",
    `${UNKEY_SESSION_COOKIE}=; Path=/; HttpOnly; SameSite=Lax; Max-Age=0`,
  );
  return response;
}
