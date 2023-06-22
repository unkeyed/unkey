import { toBase64 } from "./encoding";
import { AuthorizationError } from "./errors";

export async function getKeyHash(authorizationHeader: string | null): Promise<string> {
  if (!authorizationHeader) {
    throw new AuthorizationError("Missing Authorization header");
  }
  const token = authorizationHeader.replace("Bearer ", "");
  if (!token) {
    throw new AuthorizationError("Missing Bearer token");
  }
  return toBase64(await crypto.subtle.digest("sha-256", new TextEncoder().encode(token)));
}
