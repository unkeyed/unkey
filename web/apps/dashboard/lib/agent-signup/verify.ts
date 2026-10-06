import { getBaseUrl } from "@/lib/utils";
import { type JWTPayload, type JWTVerifyGetKey, createRemoteJWKSet, errors, jwtVerify } from "jose";
import { z } from "zod";
import type { AgentSignupConfig } from "./config";
import { AgentSignupError } from "./errors";

const REGISTRATION_ID = /^agent_reg_[A-Za-z0-9]+$/;
const USER_ID = /^user_[A-Za-z0-9]+$/;

const registrationSchema = z
  .object({
    id: z.string(),
    agent_identity: z.object({
      userland_user_id: z.string(),
    }),
    // Documented as required. A person who has not joined an organization yet
    // may omit it, and Unkey creates the workspace organization after this check.
    organization_id: z.string().nullable().optional(),
    status: z.enum(["unverified", "verified", "expired", "revoked"]),
    kind: z.enum(["anonymous", "service_auth", "identity_assertion"]),
    claim: z
      .object({
        claim_completion: z
          .object({
            claimed_at: z.string().nullable().optional(),
          })
          .nullable()
          .optional(),
      })
      .nullable()
      .optional(),
  })
  .passthrough();

const validationSchema = z
  .object({
    valid: z.boolean(),
    registration_id: z.string().nullable(),
    expires_at: z.string().nullable(),
  })
  .passthrough();

export type VerifiedAgent = {
  registrationId: string;
  userId: string;
};

export function acceptedAgentAudiences(
  config: AgentSignupConfig,
  resource = getBaseUrl(),
): string[] {
  if (!resource || resource === config.audience) {
    return [config.audience];
  }
  return [config.audience, resource];
}

export type VerifyDeps = {
  fetch: typeof fetch;
  key: JWTVerifyGetKey;
  now?: () => number;
};

const jwksCache = new Map<string, ReturnType<typeof createRemoteJWKSet>>();

export function remoteJwks(jwksUrl: string): ReturnType<typeof createRemoteJWKSet> {
  const cached = jwksCache.get(jwksUrl);
  if (cached) {
    return cached;
  }
  const created = createRemoteJWKSet(new URL(jwksUrl));
  jwksCache.set(jwksUrl, created);
  return created;
}

function invalidToken(message: string): AgentSignupError {
  return new AgentSignupError(401, "invalid_token", message, true);
}

function readBearer(header: string | null): string {
  if (!header) {
    throw new AgentSignupError(
      401,
      "missing_token",
      "Send Authorization: Bearer and the agent access token.",
      true,
    );
  }
  const match = /^Bearer ([^\s]+)$/.exec(header.trim());
  if (!match?.[1]) {
    throw invalidToken("Authorization must be a Bearer token.");
  }
  return match[1];
}

function actSub(payload: JWTPayload): string | undefined {
  const act = payload.act;
  if (!act || typeof act !== "object" || Array.isArray(act)) {
    return undefined;
  }
  const sub = "sub" in act ? act.sub : undefined;
  return typeof sub === "string" ? sub : undefined;
}

function audienceValues(aud: JWTPayload["aud"]): string[] {
  if (aud === undefined) {
    return [];
  }
  return Array.isArray(aud) ? aud : [aud];
}

function tokenAudience(payload: JWTPayload, accepted: readonly string[]): string {
  const match = audienceValues(payload.aud).find((value) => accepted.includes(value));
  if (!match) {
    throw new AgentSignupError(
      401,
      "wrong_audience",
      "The agent access token audience was rejected.",
      true,
    );
  }
  return match;
}

async function verifyAccessToken(
  token: string,
  config: AgentSignupConfig,
  key: JWTVerifyGetKey,
): Promise<{ registrationId: string; userId: string; audience: string }> {
  const accepted = acceptedAgentAudiences(config);
  let payload: JWTPayload;
  try {
    const verified = await jwtVerify(token, key, {
      issuer: config.issuer,
      audience: accepted,
    });
    payload = verified.payload;
  } catch (error) {
    if (error instanceof errors.JWTExpired) {
      throw new AgentSignupError(401, "expired", "The agent access token is expired.", true);
    }
    if (error instanceof errors.JWTClaimValidationFailed && error.claim === "aud") {
      throw new AgentSignupError(
        401,
        "wrong_audience",
        "The agent access token audience was rejected.",
        true,
      );
    }
    throw invalidToken("The agent access token could not be verified.");
  }

  if (typeof payload.sub !== "string" || !REGISTRATION_ID.test(payload.sub)) {
    throw invalidToken("The agent access token is missing a registration id.");
  }
  const userId = actSub(payload);
  if (!userId) {
    throw new AgentSignupError(
      403,
      "unclaimed",
      "The agent is not claimed. Finish the AuthKit claim ceremony before calling Unkey.",
    );
  }
  if (!USER_ID.test(userId)) {
    throw invalidToken("The agent access token claim is not bound to a WorkOS user.");
  }
  return { registrationId: payload.sub, userId, audience: tokenAudience(payload, accepted) };
}

async function workOsJson(
  fetchImpl: typeof fetch,
  config: AgentSignupConfig,
  path: string,
  init: { method: string; body?: string },
): Promise<unknown> {
  let response: Response;
  try {
    response = await fetchImpl(new URL(path, config.apiBase), {
      method: init.method,
      headers: {
        Authorization: `Bearer ${config.apiKey}`,
        "Content-Type": "application/json",
      },
      body: init.body,
      signal: AbortSignal.timeout(5000),
    });
  } catch {
    throw new AgentSignupError(503, "upstream", "WorkOS could not be reached.");
  }
  if (!response.ok) {
    throw new AgentSignupError(503, "upstream", "WorkOS rejected the agent credential check.");
  }
  try {
    return await response.json();
  } catch {
    throw new AgentSignupError(503, "upstream", "WorkOS returned an unreadable credential check.");
  }
}

export async function authenticateAgent(
  authorizationHeader: string | null,
  config: AgentSignupConfig,
  deps: VerifyDeps,
): Promise<VerifiedAgent> {
  const token = readBearer(authorizationHeader);
  const accessToken = await verifyAccessToken(token, config, deps.key);
  const now = deps.now ?? Date.now;

  const validationBody = await workOsJson(deps.fetch, config, "/agents/credentials/validate", {
    method: "POST",
    body: JSON.stringify({
      type: "access_token",
      credential: token,
      audience: accessToken.audience,
    }),
  });
  const validation = validationSchema.safeParse(validationBody);
  if (!validation.success || !validation.data.valid || !validation.data.registration_id) {
    throw invalidToken("WorkOS rejected the agent access token.");
  }
  if (validation.data.registration_id !== accessToken.registrationId) {
    throw invalidToken("The agent access token does not match its registration.");
  }
  const expiresAt = validation.data.expires_at
    ? Date.parse(validation.data.expires_at)
    : Number.NaN;
  if (!validation.data.expires_at || Number.isNaN(expiresAt) || expiresAt <= now()) {
    throw new AgentSignupError(401, "expired", "The agent access token is expired.", true);
  }

  const registrationBody = await workOsJson(
    deps.fetch,
    config,
    `/agents/registrations/${accessToken.registrationId}`,
    { method: "GET" },
  );
  const registration = registrationSchema.safeParse(registrationBody);
  if (!registration.success || registration.data.id !== accessToken.registrationId) {
    throw invalidToken("WorkOS did not return this agent registration.");
  }
  if (registration.data.kind === "anonymous") {
    throw new AgentSignupError(403, "anonymous", "Anonymous agent registrations cannot sign up.");
  }
  if (registration.data.kind !== "service_auth") {
    throw new AgentSignupError(
      403,
      "unsupported_registration",
      "Only a claimed service_auth registration can sign up.",
    );
  }
  if (registration.data.status === "unverified") {
    throw new AgentSignupError(
      403,
      "unclaimed",
      "The agent registration is not verified. Finish the AuthKit claim ceremony.",
    );
  }
  if (registration.data.status !== "verified") {
    throw invalidToken("The agent registration is not active.");
  }
  const claimedAt = registration.data.claim?.claim_completion?.claimed_at;
  if (!claimedAt) {
    throw new AgentSignupError(403, "unclaimed", "The agent registration has no completed claim.");
  }
  if (registration.data.agent_identity.userland_user_id !== accessToken.userId) {
    throw new AgentSignupError(
      403,
      "user_mismatch",
      "The agent access token is not bound to the user on this registration.",
    );
  }

  return {
    registrationId: accessToken.registrationId,
    userId: accessToken.userId,
  };
}
