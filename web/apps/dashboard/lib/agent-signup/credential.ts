import { type KeyObject, createPrivateKey, createPublicKey } from "node:crypto";
import { env } from "@/lib/env";
import { SignJWT, exportJWK } from "jose";

export const AGENT_SIGNUP_ISSUER = "https://app.unkey.com/agent-signup";
export const AGENT_SIGNUP_AUDIENCE = "api.unkey.com";
export const AGENT_SIGNUP_ROLE = "agent_signup";

const SIGNING_KEY_ID = "agent-signup";
const TOKEN_TTL_SECONDS = 120;

function pem(raw: string): string {
  return raw.trim().replace(/^"|"$/g, "").replace(/\\n/g, "\n");
}

async function signingKey(): Promise<{ key: KeyObject | Uint8Array; alg: "RS256" | "HS256" }> {
  const privateKey = env().UNKEY_AGENT_SIGNUP_JWT_PRIVATE_KEY?.trim();
  if (privateKey) {
    return { key: createPrivateKey(pem(privateKey)), alg: "RS256" };
  }
  const secret = env().UNKEY_AGENT_SIGNUP_JWT_SECRET?.trim();
  if (secret) {
    if (secret.length < 32) {
      throw new Error("UNKEY_AGENT_SIGNUP_JWT_SECRET must be at least 32 bytes");
    }
    return { key: new TextEncoder().encode(secret), alg: "HS256" };
  }
  throw new Error("Agent signup JWT signing is not configured");
}

export async function mintAgentSignupJWT(params: {
  orgId: string;
  subject: string;
  name: string;
}): Promise<string> {
  const { key, alg } = await signingKey();
  const now = Math.floor(Date.now() / 1000);
  return new SignJWT({
    org: { id: params.orgId },
    name: params.name,
    roles: [AGENT_SIGNUP_ROLE],
  })
    .setProtectedHeader(
      alg === "RS256" ? { alg, typ: "JWT", kid: SIGNING_KEY_ID } : { alg, typ: "JWT" },
    )
    .setIssuer(AGENT_SIGNUP_ISSUER)
    // Go's JWT claims type unmarshals aud as a string array. A single audience
    // string fails verification even when the signature and secret match.
    .setAudience([AGENT_SIGNUP_AUDIENCE])
    .setSubject(params.subject)
    .setIssuedAt(now)
    .setNotBefore(now)
    .setExpirationTime(now + TOKEN_TTL_SECONDS)
    .sign(key);
}

export async function agentSignupJwks(): Promise<{ keys: Array<Record<string, unknown>> } | null> {
  const privateKey = env().UNKEY_AGENT_SIGNUP_JWT_PRIVATE_KEY?.trim();
  if (!privateKey) {
    return null;
  }
  const jwk = await exportJWK(createPublicKey(createPrivateKey(pem(privateKey))));
  return {
    keys: [
      {
        ...jwk,
        alg: "RS256",
        use: "sig",
        kid: SIGNING_KEY_ID,
      },
    ],
  };
}
