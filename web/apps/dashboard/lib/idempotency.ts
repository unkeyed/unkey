import { UnkeyError } from "@unkey/api/models/errors";

/**
 * Sends a deployment create with an Idempotency-Key so retries of one attempt
 * (double click, network retry, resubmit after an error) deduplicate
 * server-side.
 *
 * Retrying the same body reuses its key, because only the unchanged key can
 * return a deployment the failed attempt may still have created. A changed
 * body is a new deployment intent and gets its own key. A key is discarded
 * after a settled success, and after any 4xx: a rejected request created
 * nothing, and a spent key is the one error retrying the same key can never
 * clear.
 */
export async function withIdempotencyKey<T>(
  body: unknown,
  request: (idempotencyKey: string) => Promise<T>,
): Promise<T> {
  const fingerprint = JSON.stringify(body) ?? "";
  let key = readKey(fingerprint);
  if (key === undefined) {
    key = crypto.randomUUID();
    writeKey(fingerprint, key);
  }
  try {
    const result = await request(key);
    clearKey(fingerprint);
    return result;
  } catch (error) {
    if (error instanceof UnkeyError && error.statusCode >= 400 && error.statusCode < 500) {
      clearKey(fingerprint);
    }
    throw error;
  }
}

const storagePrefix = "unkey:deploy-idempotency:";

// sessionStorage carries a held key across a reload between attempts. It
// throws where site data is blocked (private mode), so keys fall back to a
// page-scoped map, which still survives the dialog unmounting.
const fallbackKeys = new Map<string, string>();

function readKey(fingerprint: string): string | undefined {
  try {
    const key = sessionStorage.getItem(storagePrefix + fingerprint);
    if (key !== null) {
      return key;
    }
  } catch {
    // fall through to the in-memory copy
  }
  return fallbackKeys.get(fingerprint);
}

function writeKey(fingerprint: string, key: string): void {
  try {
    sessionStorage.setItem(storagePrefix + fingerprint, key);
    return;
  } catch {
    fallbackKeys.set(fingerprint, key);
  }
}

function clearKey(fingerprint: string): void {
  try {
    sessionStorage.removeItem(storagePrefix + fingerprint);
  } catch {
    // a key that never reached storage lives in the map; delete below
  }
  fallbackKeys.delete(fingerprint);
}
