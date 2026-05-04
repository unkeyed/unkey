import crypto from "node:crypto";
import { env } from "@/lib/env";

// Lifetime of a sealed Turnstile challenge. Long enough for a user to solve
// the widget, short enough to limit replay if a token leaks.
const CHALLENGE_TTL_MS = 10 * 60 * 1000;

export type SealedChallengeAction = "sign-up" | "sign-in";

export type SealedChallengePayload = {
  email: string;
  action: SealedChallengeAction;
  userData?: { firstName: string; lastName: string };
};

type SealedChallengeBody = SealedChallengePayload & {
  // Random per-challenge id. We pass a hash of this back to Cloudflare as
  // the widget action so the Turnstile token itself is bound to this exact
  // challenge and cannot be replayed across different emails or actions.
  nonce: string;
  expiresAt: number;
};

export type VerifiedChallenge = SealedChallengePayload & {
  // Action string sent to Cloudflare's siteverify and the Turnstile widget.
  // Callers compare this to the action returned by Cloudflare to ensure the
  // solved token is for this exact challenge.
  cloudflareAction: string;
};

// Derive a stable HMAC key from WORKOS_COOKIE_PASSWORD so we don't need a
// new secret. The password is already required in production for sealed
// session cookies.
function getSigningKey(): Buffer {
  const password = env().WORKOS_COOKIE_PASSWORD;
  if (!password) {
    throw new Error("WORKOS_COOKIE_PASSWORD is required to seal Turnstile challenges");
  }
  return crypto.createHash("sha256").update(`turnstile-challenge:${password}`).digest();
}

function base64UrlEncode(input: Buffer | string): string {
  const buf = typeof input === "string" ? Buffer.from(input, "utf8") : input;
  return buf.toString("base64").replace(/\+/g, "-").replace(/\//g, "_").replace(/=+$/, "");
}

function base64UrlDecode(input: string): Buffer {
  const padded = input.replace(/-/g, "+").replace(/_/g, "/");
  const padLen = (4 - (padded.length % 4)) % 4;
  return Buffer.from(padded + "=".repeat(padLen), "base64");
}

function sign(payload: string): string {
  return base64UrlEncode(crypto.createHmac("sha256", getSigningKey()).update(payload).digest());
}

// Cloudflare's `action` field accepts up to 32 chars matching `[a-zA-Z0-9_-]`.
// We hash the random nonce and truncate so the bound action is unique per
// challenge while staying within Cloudflare's constraints.
function deriveCloudflareAction(nonce: string): string {
  const digest = crypto.createHash("sha256").update(nonce).digest("base64url");
  return digest.slice(0, 32);
}

export function sealTurnstileChallenge(payload: SealedChallengePayload): {
  token: string;
  cloudflareAction: string;
} {
  const nonce = crypto.randomBytes(16).toString("base64url");
  const body: SealedChallengeBody = {
    ...payload,
    nonce,
    expiresAt: Date.now() + CHALLENGE_TTL_MS,
  };
  const encoded = base64UrlEncode(JSON.stringify(body));
  const signature = sign(encoded);
  return {
    token: `${encoded}.${signature}`,
    cloudflareAction: deriveCloudflareAction(nonce),
  };
}

function isUserData(value: unknown): value is { firstName: string; lastName: string } {
  if (typeof value !== "object" || value === null) {
    return false;
  }
  if (!("firstName" in value) || typeof value.firstName !== "string") {
    return false;
  }
  if (!("lastName" in value) || typeof value.lastName !== "string") {
    return false;
  }
  return true;
}

function isSealedChallengeBody(value: unknown): value is SealedChallengeBody {
  if (typeof value !== "object" || value === null) {
    return false;
  }
  if (!("email" in value) || typeof value.email !== "string") {
    return false;
  }
  if (!("action" in value) || (value.action !== "sign-up" && value.action !== "sign-in")) {
    return false;
  }
  if (!("nonce" in value) || typeof value.nonce !== "string") {
    return false;
  }
  if (!("expiresAt" in value) || typeof value.expiresAt !== "number") {
    return false;
  }
  if ("userData" in value && value.userData !== undefined && !isUserData(value.userData)) {
    return false;
  }
  return true;
}

export function verifyTurnstileChallenge(token: string): VerifiedChallenge | null {
  const parts = token.split(".");
  if (parts.length !== 2) {
    return null;
  }
  const [encoded, signature] = parts;
  const expected = sign(encoded);
  // Constant-time comparison to avoid leaking signature bytes.
  const expectedBuf = base64UrlDecode(expected);
  const providedBuf = base64UrlDecode(signature);
  if (expectedBuf.length !== providedBuf.length) {
    return null;
  }
  if (!crypto.timingSafeEqual(expectedBuf, providedBuf)) {
    return null;
  }

  let parsed: unknown;
  try {
    parsed = JSON.parse(base64UrlDecode(encoded).toString("utf8"));
  } catch {
    return null;
  }
  if (!isSealedChallengeBody(parsed)) {
    return null;
  }
  if (parsed.expiresAt < Date.now()) {
    return null;
  }
  return {
    email: parsed.email,
    action: parsed.action,
    userData: parsed.userData,
    cloudflareAction: deriveCloudflareAction(parsed.nonce),
  };
}
