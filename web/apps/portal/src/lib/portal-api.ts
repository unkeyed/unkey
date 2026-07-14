import { createServerFn } from "@tanstack/react-start";
import { getCookie } from "@tanstack/react-start/server";
import {
  type KeysPage,
  type RerollKeyResult,
  listKeysQuerySchema,
  listKeysResponseSchema,
  mapListKeysResponse,
  rerollKeyRequestSchema,
  rerollKeyResponseSchema,
} from "~/components/keys-table/schema/keys.schema";
import { env } from "./env";
import { SESSION_COOKIE_NAME } from "./session";

/**
 * Server-side proxy for the internal `v2/portal.*` data endpoints.
 *
 * The portal session token is an httpOnly cookie the browser cannot read, and
 * the API's portal auth service only claims cookie-authenticated requests (it
 * bails if an `Authorization` header is present). So these server functions run
 * on the portal's server, read the cookie, and forward it to the API as a
 * `Cookie: portal_session=<token>` header — the same transport the browser
 * would use if the API were same-origin. The token never reaches client code.
 */

const REQUEST_TIMEOUT_MS = 10_000;

/**
 * Message used for every unauthorized (expired/invalid session) failure. This
 * doubles as the cross-boundary discriminant: custom error fields do NOT survive
 * the server-fn boundary (the client receives a plain `Error` with only
 * `message`), so the UI keys off this exact string via {@link isUnauthorizedError}
 * to show a distinct expired-session state. Keep the two in sync.
 */
export const SESSION_EXPIRED_MESSAGE =
  "Your session has expired. Please return to the application.";

/**
 * Thrown when a portal API call fails. Carries only a human-readable `message`,
 * because that is all that crosses the server-fn boundary intact. Unauthorized
 * failures use {@link SESSION_EXPIRED_MESSAGE} so the client can detect them.
 */
export class PortalApiError extends Error {
  constructor(message: string) {
    super(message);
    this.name = "PortalApiError";
  }
}

/**
 * Whether an error (as seen on the client, after the server-fn boundary has
 * flattened it to a plain `Error`) is an expired/invalid-session failure.
 */
export function isUnauthorizedError(error: unknown): boolean {
  return error instanceof Error && error.message === SESSION_EXPIRED_MESSAGE;
}

async function portalFetch(path: string, body: Record<string, unknown>): Promise<unknown> {
  const token = getCookie(SESSION_COOKIE_NAME);
  if (!token) {
    throw new PortalApiError(SESSION_EXPIRED_MESSAGE);
  }

  const controller = new AbortController();
  const timeoutId = setTimeout(() => controller.abort(), REQUEST_TIMEOUT_MS);

  let response: Response;
  try {
    response = await fetch(`${env().UNKEY_API_URL}${path}`, {
      method: "POST",
      headers: {
        "Content-Type": "application/json",
        // Cookie transport only — never set Authorization, or the API's portal
        // auth service refuses to claim the request.
        Cookie: `${SESSION_COOKIE_NAME}=${token}`,
      },
      body: JSON.stringify(body),
      signal: controller.signal,
    });
  } catch (err) {
    const message =
      err instanceof DOMException && err.name === "AbortError"
        ? "Request timed out. Please try again."
        : "Network error. Please check your connection and try again.";
    throw new PortalApiError(message);
  } finally {
    clearTimeout(timeoutId);
  }

  if (response.status === 401 || response.status === 403) {
    throw new PortalApiError(SESSION_EXPIRED_MESSAGE);
  }
  if (!response.ok) {
    throw new PortalApiError("Something went wrong. Please try again.");
  }

  try {
    return await response.json();
  } catch {
    throw new PortalApiError("Received an unexpected response. Please try again.");
  }
}

/**
 * List the session end user's keys (one page). Scoping to the end user and the
 * portal's keyspaces happens server-side in the API from the session cookie.
 */
export const listKeys = createServerFn({ method: "GET" })
  .inputValidator((query: unknown) => listKeysQuerySchema.parse(query))
  .handler(async ({ data }): Promise<KeysPage> => {
    const body: Record<string, unknown> = {};
    if (data.cursor) {
      body.cursor = data.cursor;
    }
    if (data.limit) {
      body.limit = data.limit;
    }
    const raw = await portalFetch("/v2/portal.listKeys", body);
    return mapListKeysResponse(listKeysResponseSchema.parse(raw));
  });

/**
 * Reroll a key the session end user owns, returning the new plaintext secret
 * once. Ownership is verified server-side against the session's external id.
 */
export const rerollKey = createServerFn({ method: "POST" })
  .inputValidator((input: unknown) => rerollKeyRequestSchema.parse(input))
  .handler(async ({ data }): Promise<RerollKeyResult> => {
    const raw = await portalFetch("/v2/portal.rerollKey", {
      keyId: data.keyId,
      expiration: data.expiration,
    });
    const parsed = rerollKeyResponseSchema.parse(raw);
    return { keyId: parsed.data.keyId, plaintext: parsed.data.key };
  });
