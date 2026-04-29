import { env } from "@/lib/env";
import { SignJWT } from "jose";

const TTL_SECONDS = 120;

// ISSUER and AUDIENCE pin the token to this minter and verifier pair.
// svc/api rejects any token whose iss/aud differ from these strings, so
// even a token signed with the right secret can't be replayed against a
// different Unkey service that happens to share infrastructure. Use the
// public domains so the values are stable across environments and self-
// documenting. Must match Issuer and Audience in pkg/auth/jwt/verifier.go.
const ISSUER = "app.unkey.com";
const AUDIENCE = "api.unkey.com";

// mintProxyJWT signs a short-lived HS256 JWT carrying the workspace, subject,
// and granted permissions. svc/api verifies it via pkg/auth/jwt against the
// matching jwt_secret. The 2-minute TTL bounds blast radius if the dashboard
// process leaks one.
//
// All claims svc/api requires are set here:
//   - sub: stable subject ID for audit logs
//   - iss/aud: pinned to this minter/verifier pair
//   - iat: issuance time
//   - nbf: not-before, pinned to iat so a token can never be valid before
//     it was issued; svc/api rejects tokens missing nbf to defend against
//     timing/clock-skew attacks
//   - exp: iat + TTL, capped on the verify side at 5 minutes
export async function mintProxyJWT(opts: {
  workspaceId: string;
  subjectId: string;
  subjectName: string;
  permissions: string[];
}): Promise<string> {
  const secret = new TextEncoder().encode(env().UNKEY_JWT_SECRET);
  const now = Math.floor(Date.now() / 1000);
  return await new SignJWT({
    wid: opts.workspaceId,
    name: opts.subjectName,
    perms: opts.permissions,
  })
    .setProtectedHeader({ alg: "HS256" })
    .setIssuer(ISSUER)
    .setAudience(AUDIENCE)
    .setSubject(opts.subjectId)
    .setIssuedAt(now)
    .setNotBefore(now)
    .setExpirationTime(now + TTL_SECONDS)
    .sign(secret);
}
