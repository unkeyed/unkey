import { env } from "@/lib/env";
import { SignJWT } from "jose";

export type ProxyJWTParams = {
  orgId: string;
  role: string;
  subject: string;
  name: string;
};

// mintProxyJWT creates the short-lived credential that lets the dashboard call
// svc/api without exposing a root key to the browser. The dashboard only needs
// the active signing secret. svc/api carries the ordered verification set so old
// secrets can remain valid during a rotation window.
export async function mintProxyJWT(params: ProxyJWTParams): Promise<string> {
  const { UNKEY_JWT_SECRET: signingSecret } = env();
  if (!signingSecret) {
    throw new Error("UNKEY_JWT_SECRET must be configured for dashboard proxy signing");
  }

  const key = new TextEncoder().encode(signingSecret);
  const now = Math.floor(Date.now() / 1000);

  return (
    new SignJWT({
      org: {
        id: params.orgId,
      },
      name: params.name,
      roles: [params.role],
    })
      .setProtectedHeader({ alg: "HS256", typ: "JWT" })
      .setIssuer("app.unkey.com")
      // Mirror the WorkOS JWT template's audience so svc/api's WorkOS auth entry
      // (configured with audience = "api.unkey.com") verifies this fallback token
      // the same way it verifies a forwarded WorkOS access token.
      .setAudience(["api.unkey.com"])
      .setSubject(params.subject)
      .setIssuedAt(now)
      .setNotBefore(now)
      .setExpirationTime(now + 120)
      .sign(key)
  );
}
