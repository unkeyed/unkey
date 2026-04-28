import { env } from "@/lib/env";
import { SignJWT } from "jose";

const TTL_SECONDS = 120;

// mintProxyJWT signs a short-lived HS256 JWT carrying the workspace, subject,
// and granted permissions. svc/api verifies it via pkg/auth/jwt against the
// matching jwt_secret. The 2-minute TTL bounds blast radius if the dashboard
// process leaks one.
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
    .setSubject(opts.subjectId)
    .setIssuedAt(now)
    .setExpirationTime(now + TTL_SECONDS)
    .sign(secret);
}
