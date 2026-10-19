import { env } from "@/lib/env";
import { SignJWT } from "jose";

export async function mintProxyJWT(params: {
  orgId: string;
  role: string;
  subject: string;
  name: string;
}): Promise<string> {
  const { UNKEY_JWT_SECRET: signingSecret } = env();
  if (!signingSecret) {
    throw new Error("UNKEY_JWT_SECRET must be configured for dashboard proxy signing");
  }

  const key = new TextEncoder().encode(signingSecret);
  const now = Math.floor(Date.now() / 1000);

  return new SignJWT({
    org: { id: params.orgId },
    name: params.name,
    roles: [params.role],
  })
    .setProtectedHeader({ alg: "HS256", typ: "JWT" })
    .setIssuer("app.unkey.com")
    .setAudience([env().UNKEY_JWT_AUDIENCE ?? "api.unkey.com"])
    .setSubject(params.subject)
    .setIssuedAt(now)
    .setNotBefore(now)
    .setExpirationTime(now + 120)
    .sign(key);
}
