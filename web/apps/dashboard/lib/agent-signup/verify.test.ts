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
  claims: { audience?: string; expired?: boolean; act?: { sub: string }; orgId?: string },
): Promise<string> {
  const issuedAt = Math.floor(Date.now() / 1000) - 120;
  const payload: Record<string, unknown> = {};
  if (claims.act) {
    payload.act = claims.act;
  }
  if (claims.orgId) {
    payload.org_id = claims.orgId;
  }
  let jwt = new SignJWT(payload)
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
  audience?: string;
}) {
  return vi.fn(async (input: RequestInfo | URL, init?: RequestInit) => {
    const url = String(input);
    if (url.endsWith("/agents/credentials/validate")) {
      const body = JSON.parse(String(init?.body));
      expect(body).toMatchObject({
        type: "access_token",
        audience: options.audience ?? config.audience,
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

  it("accepts the protected resource audience and rejects any other", async () => {
    process.env.DASHBOARD_BASE_URL = "https://app.unkey.com";
    try {
      const { privateKey, key } = await signingKey();
      const accessToken = await token(privateKey, {
        act: { sub: userId },
        audience: "https://app.unkey.com",
      });
      const fetchImpl = workOsFetch({ audience: "https://app.unkey.com" });
      await expect(
        authenticateAgent(`Bearer ${accessToken}`, config, { fetch: fetchImpl, key }),
      ).resolves.toEqual({ registrationId, userId });

      const other = await token(privateKey, {
        act: { sub: userId },
        audience: "https://evil.example",
      });
      await expect(
        authenticateAgent(`Bearer ${other}`, config, { fetch: workOsFetch({}), key }),
      ).rejects.toMatchObject({ code: "wrong_audience", status: 401 });
    } finally {
      delete process.env.DASHBOARD_BASE_URL;
    }
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

  it("rejects a verified registration bound to a different user when no organization was selected", async () => {
    const { privateKey, key } = await signingKey();
    const accessToken = await token(privateKey, { act: { sub: userId } });
    await expect(
      authenticateAgent(`Bearer ${accessToken}`, config, {
        fetch: workOsFetch({
          registrationBody: registration({
            organization_id: null,
            agent_identity: { userland_user_id: "user_other" },
          }),
        }),
        key,
      }),
    ).rejects.toMatchObject({ code: "user_mismatch", status: 403 });
  });

  it("accepts an existing user who selected an organization during claim", async () => {
    const { privateKey, key } = await signingKey();
    const existingUserId = "user_existing";
    const accessToken = await token(privateKey, {
      act: { sub: existingUserId },
      orgId: "org_01EHQMYV6MBK39QC5PZXHY59C3",
    });
    await expect(
      authenticateAgent(`Bearer ${accessToken}`, config, {
        fetch: workOsFetch({
          registrationBody: registration({
            organization_id: "org_01EHQMYV6MBK39QC5PZXHY59C3",
            agent_identity: { userland_user_id: "user_identity" },
          }),
        }),
        key,
      }),
    ).resolves.toEqual({ registrationId, userId: existingUserId });
  });

  it("accepts an org-scoped token when the registration organization is still empty", async () => {
    const { privateKey, key } = await signingKey();
    const accessToken = await token(privateKey, {
      act: { sub: userId },
      orgId: "org_selected",
    });
    await expect(
      authenticateAgent(`Bearer ${accessToken}`, config, {
        fetch: workOsFetch({
          registrationBody: registration({
            organization_id: null,
            agent_identity: { userland_user_id: "user_identity" },
          }),
        }),
        key,
      }),
    ).resolves.toEqual({ registrationId, userId });
  });

  it("accepts a claimed registration whose agent identity has no user yet", async () => {
    const { privateKey, key } = await signingKey();
    const accessToken = await token(privateKey, { act: { sub: userId } });
    await expect(
      authenticateAgent(`Bearer ${accessToken}`, config, {
        fetch: workOsFetch({
          registrationBody: registration({
            agent_identity: { userland_user_id: null },
          }),
        }),
        key,
      }),
    ).resolves.toEqual({ registrationId, userId });
  });

  it("rejects a claim completed by a different user", async () => {
    const { privateKey, key } = await signingKey();
    const accessToken = await token(privateKey, {
      act: { sub: userId },
      orgId: "org_01EHQMYV6MBK39QC5PZXHY59C3",
    });
    await expect(
      authenticateAgent(`Bearer ${accessToken}`, config, {
        fetch: workOsFetch({
          registrationBody: registration({
            organization_id: "org_01EHQMYV6MBK39QC5PZXHY59C3",
            agent_identity: { userland_user_id: userId },
            claim: {
              claimed_by: {
                user_id: "user_other",
                organization_id: "org_01EHQMYV6MBK39QC5PZXHY59C3",
              },
              claim_completion: { claimed_at: "2026-01-15T12:00:00.000Z" },
            },
          }),
        }),
        key,
      }),
    ).rejects.toMatchObject({ code: "user_mismatch", status: 403 });
  });
});
