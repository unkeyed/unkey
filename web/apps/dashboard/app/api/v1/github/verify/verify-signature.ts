import { Buffer } from "node:buffer";
import crypto from "node:crypto";
import { z } from "zod";

const githubKeysSchema = z.object({
  public_keys: z.array(
    z.object({
      key_identifier: z.string(),
      key: z.string(),
      is_current: z.boolean(),
    }),
  ),
});

type PublicKeys = z.infer<typeof githubKeysSchema>["public_keys"];

// GitHub rotates its secret-scanning public keys very rarely, but the
// unauthenticated meta endpoint that serves them is rate-limited (~60 req/hr
// per IP). Fetching on every incoming webhook tripped 429s, and a 429 made
// verification return false -> the route answered 401 and returned early,
// which silently skipped the deferred leaked-key notifications. Cache the keys
// in module memory and fall back to the last known-good set whenever a refresh
// fails, so rate limiting can no longer break verification.
const KEYS_TTL_MS = 60 * 60 * 1000; // 1 hour

let cachedKeys: PublicKeys | null = null;
let cachedAt = 0;
let inFlight: Promise<PublicKeys | null> | null = null;

// Test-only hook: module-level cache would otherwise leak state across tests.
export function __resetGithubKeysCacheForTests(): void {
  cachedKeys = null;
  cachedAt = 0;
  inFlight = null;
}

async function fetchGithubKeys(githubKeysUri: string): Promise<PublicKeys | null> {
  // GitHub requires a User-Agent on API requests; unauthenticated requests
  // without one are rejected.
  const response = await fetch(githubKeysUri, {
    headers: { "User-Agent": "unkey-secret-scanning" },
  });
  if (!response.ok) {
    console.error("Github verify error", response.status, await response.text());
    return null;
  }

  const parsed = githubKeysSchema.safeParse(await response.json());
  if (!parsed.success) {
    console.error("Github keys response did not match expected shape", parsed.error.message);
    return null;
  }
  return parsed.data.public_keys;
}

async function getGithubKeys(githubKeysUri: string): Promise<PublicKeys | null> {
  if (cachedKeys && Date.now() - cachedAt < KEYS_TTL_MS) {
    console.info("[github-verify] using cached github public keys", {
      count: cachedKeys.length,
      ageMs: Date.now() - cachedAt,
    });
    return cachedKeys;
  }

  // Coalesce concurrent refreshes so a burst of webhooks makes at most one
  // request to GitHub rather than a thundering herd that trips the rate limit.
  if (!inFlight) {
    inFlight = fetchGithubKeys(githubKeysUri).finally(() => {
      inFlight = null;
    });
  }
  const refreshed = await inFlight;

  if (refreshed) {
    cachedKeys = refreshed;
    cachedAt = Date.now();
    console.info("[github-verify] refreshed github public keys", { count: refreshed.length });
    return refreshed;
  }

  // Refresh failed (e.g. rate-limited). Serve the last known-good keys if we
  // have any; they change rarely enough that a stale set still verifies.
  if (cachedKeys) {
    console.warn("[github-verify] falling back to stale cached github public keys", {
      count: cachedKeys.length,
      ageMs: Date.now() - cachedAt,
    });
    return cachedKeys;
  }
  console.error("[github-verify] no github public keys available (refresh failed, no cache)");
  return null;
}

export async function verifyGitSignature(
  payload: string,
  signature: string,
  keyId: string,
  githubKeysUri: string,
): Promise<boolean> {
  const publicKeys = await getGithubKeys(githubKeysUri);
  if (!publicKeys) {
    return false;
  }

  const publicKey = publicKeys.find((k) => k.key_identifier === keyId);
  if (!publicKey) {
    console.error("[github-verify] no matching public key for requested key id", {
      requestedKeyId: keyId,
      availableKeyIds: publicKeys.map((k) => k.key_identifier),
    });
    return false;
  }

  const verifier = crypto.createVerify("SHA256").update(payload);
  const valid = verifier.verify(publicKey.key, Buffer.from(signature, "base64"));
  console.info("[github-verify] signature check complete", { keyId, valid });
  return valid;
}
