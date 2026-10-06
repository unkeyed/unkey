// @vitest-environment node
import { type KeyLike, SignJWT, createLocalJWKSet, exportJWK, generateKeyPair } from "jose";
import { describe, expect, it, vi } from "vitest";
import type { AgentSignupConfig } from "./config";
import { AgentSignupError } from "./errors";
import { authenticateAgent } from "./verify";

const config: AgentSignupConfig = {
  apiKey: "sk_test",
  clientId: "client_123",
  issuer: "https://auth.example.com",
  audience: "client_123",
  apiBase: "https://api.workos.com",
  jwksUrl: "https://auth.example.com/oauth2/jwks",
};

const registrationId = "agent_reg_abc";
const userId = "user_abc";

async function signingKey() {
  const { publicKey, privateKey } = await generateKeyPair("RS256", { extractable: true });
  const jwk = await exportJWK(publicKey);
  jwk.alg = "RS256";
  jwk.kid = "test";
  jwk.use = "sig";
  return { privateKey, key: createLocalJWKSet({ keys: [jwk] }) };
}

async function token(
  privateKey: KeyLike,
  claims: { audience?: string; expired?: boolean; act?: { sub: string } },
): Promise<string> {
  const issuedAt = Math.floor(Date.now() / 1000) - 120;
  let jwt = new SignJWT(claims.act ? { act: claims.act } : {})
    .setProtectedHeader({ alg: "RS256", kid: "test" })
    .setIssuer(config.issuer)
    .setAudience(claims.audience ?? config.audience)
    .setSubject(registrationId)
    .setIssuedAt(issuedAt);
  jwt = jwt.setExpirationTime(claims.expired ? issuedAt + 30 : issuedAt + 3600);
  return jwt.sign(privateKey);
}

function registration(overrides: Record<string, unknown> = {}) {
  return {
    id: registrationId,
    agent_identity: { userland_user_id: userId },
    organization_id: null,
    status: "verified",
    kind: "service_auth",
    claim: { claim_completion: { claimed_at: "2026-01-15T12:00:00.000Z" } },
    ...overrides,
  };
}

function workOsFetch(options: {
  valid?: boolean;
  expiresAt?: string | null;
  registrationBody?: unknown;
}) {
  return vi.fn(async (input: RequestInfo | URL, init?: RequestInit) => {
    const url = String(input);
    if (url.endsWith("/agents/credentials/validate")) {
      const body = JSON.parse(String(init?.body));
      expect(body).toMatchObject({
        type: "access_token",
        audience: config.audience,
      });
      return Response.json({
        valid: options.valid ?? true,
        registration_id: options.valid === false ? null : registrationId,
        expires_at:
          options.expiresAt === undefined ? "2099-01-15T12:00:00.000Z" : options.expiresAt,
      });
    }
    if (url.endsWith(`/agents/registrations/${registrationId}`)) {
      return Response.json(options.registrationBody ?? registration());
    }
    return new Response("missing", { status: 404 });
  });
}

describe("authenticateAgent", () => {
  it("accepts a claimed service_auth token", async () => {
    const { privateKey, key } = await signingKey();
    const accessToken = await token(privateKey, { act: { sub: userId } });
    const fetchImpl = workOsFetch({});
    await expect(
      authenticateAgent(`Bearer ${accessToken}`, config, { fetch: fetchImpl, key }),
    ).resolves.toEqual({ registrationId, userId });
    expect(fetchImpl).toHaveBeenCalledTimes(2);
  });

  it("rejects an expired token", async () => {
    const { privateKey, key } = await signingKey();
    const accessToken = await token(privateKey, { act: { sub: userId }, expired: true });
    const fetchImpl = workOsFetch({});
    await expect(
      authenticateAgent(`Bearer ${accessToken}`, config, { fetch: fetchImpl, key }),
    ).rejects.toMatchObject({ code: "expired", status: 401 });
    expect(fetchImpl).not.toHaveBeenCalled();
  });

  it("rejects the wrong audience", async () => {
    const { privateKey, key } = await signingKey();
    const accessToken = await token(privateKey, { act: { sub: userId }, audience: "client_other" });
    await expect(
      authenticateAgent(`Bearer ${accessToken}`, config, { fetch: workOsFetch({}), key }),
    ).rejects.toMatchObject({ code: "wrong_audience", status: 401 });
  });

  it("rejects an unclaimed token and an anonymous registration", async () => {
    const { privateKey, key } = await signingKey();
    const unclaimed = await token(privateKey, {});
    await expect(
      authenticateAgent(`Bearer ${unclaimed}`, config, { fetch: workOsFetch({}), key }),
    ).rejects.toMatchObject({ code: "unclaimed", status: 403 });

    const claimed = await token(privateKey, { act: { sub: userId } });
    await expect(
      authenticateAgent(`Bearer ${claimed}`, config, {
        fetch: workOsFetch({
          registrationBody: registration({ kind: "anonymous", status: "unverified" }),
        }),
        key,
      }),
    ).rejects.toMatchObject({ code: "anonymous", status: 403 });
  });

  it("rejects a credential WorkOS marks invalid or expired", async () => {
    const { privateKey, key } = await signingKey();
    const accessToken = await token(privateKey, { act: { sub: userId } });
    await expect(
      authenticateAgent(`Bearer ${accessToken}`, config, {
        fetch: workOsFetch({ valid: false }),
        key,
      }),
    ).rejects.toBeInstanceOf(AgentSignupError);

    await expect(
      authenticateAgent(`Bearer ${accessToken}`, config, {
        fetch: workOsFetch({ expiresAt: "2000-01-01T00:00:00.000Z" }),
        key,
      }),
    ).rejects.toMatchObject({ code: "expired" });
  });

  it("rejects a verified registration bound to a different user", async () => {
    const { privateKey, key } = await signingKey();
    const accessToken = await token(privateKey, { act: { sub: userId } });
    await expect(
      authenticateAgent(`Bearer ${accessToken}`, config, {
        fetch: workOsFetch({
          registrationBody: registration({
            agent_identity: { userland_user_id: "user_other" },
          }),
        }),
        key,
      }),
    ).rejects.toMatchObject({ code: "user_mismatch", status: 403 });
  });
});
