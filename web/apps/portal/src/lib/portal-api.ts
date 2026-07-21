import { createServerFn } from "@tanstack/react-start";
import { getCookie } from "@tanstack/react-start/server";
import type { Unkey } from "@unkey/api";
import {
  type KeysPage,
  type RerollKeyResult,
  listKeysQuerySchema,
  rerollKeyRequestSchema,
} from "~/components/keys-table/schema/keys.schema";
import { env } from "./env";
import { SESSION_COOKIE_NAME } from "./session";

/**
 * Server-side proxy for the internal `v2/portal.*` data endpoints, over the
 * vendored `@unkey/api` SDK.
 *
 * The portal session token is an httpOnly cookie the browser cannot read, and
 * the API's portal auth service only claims cookie-authenticated requests (it
 * bails if an `Authorization` header is present). The SDK's portal operations
 * model auth as the `portal_session` cookie credential, so these server
 * functions run on the portal's server, read the cookie, and hand it to the SDK
 * as `portalSession` — never as a bearer token. The token never reaches client
 * code.
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

/**
 * Run `fn` against a portal-authenticated SDK client, translating the outcome's
 * failures into a {@link PortalApiError}. The SDK is imported dynamically to keep
 * it out of the client bundle (same reason `session.ts` defers its db import);
 * `retryConfig` is off because react-query owns retries at the call site.
 */
async function withPortalClient<T>(fn: (client: Unkey, token: string) => Promise<T>): Promise<T> {
  const token = getCookie(SESSION_COOKIE_NAME);
  if (!token) {
    throw new PortalApiError(SESSION_EXPIRED_MESSAGE);
  }

  const { Unkey } = await import("@unkey/api");
  const client = new Unkey({
    serverURL: env().UNKEY_API_URL,
    timeoutMs: REQUEST_TIMEOUT_MS,
    retryConfig: { strategy: "none" },
  });

  try {
    return await fn(client, token);
  } catch (err) {
    throw await toPortalApiError(err);
  }
}

/**
 * Map an SDK error onto a user-facing {@link PortalApiError}, preserving the
 * expired-session discriminant. Error classes are imported dynamically so they
 * stay out of the client bundle.
 */
async function toPortalApiError(err: unknown): Promise<PortalApiError> {
  if (err instanceof PortalApiError) {
    return err;
  }

  const { UnkeyError, HTTPClientError } = await import("@unkey/api/models/errors");

  if (err instanceof UnkeyError) {
    if (err.statusCode === 401 || err.statusCode === 403) {
      return new PortalApiError(SESSION_EXPIRED_MESSAGE);
    }
    return new PortalApiError("Something went wrong. Please try again.");
  }

  if (err instanceof HTTPClientError) {
    // RequestTimeoutError / RequestAbortedError vs ConnectionError / others.
    if (err.name === "RequestTimeoutError" || err.name === "RequestAbortedError") {
      return new PortalApiError("Request timed out. Please try again.");
    }
    return new PortalApiError("Network error. Please check your connection and try again.");
  }

  return new PortalApiError("Something went wrong. Please try again.");
}

/**
 * List the session end user's keys (one page). Scoping to the end user and the
 * portal's keyspaces happens server-side in the API from the session cookie.
 * `listKeys` returns an auto-paginating iterator whose resolved value is the
 * first page; the caller's `useInfiniteQuery` drives the cursor from there.
 */
export const listKeys = createServerFn({ method: "GET" })
  .inputValidator((query: unknown) => listKeysQuerySchema.parse(query))
  .handler(
    ({ data }): Promise<KeysPage> =>
      withPortalClient(async (client, token) => {
        const page = await client.portal.listKeys(
          { portalSession: token },
          { cursor: data.cursor, limit: data.limit },
        );
        const { data: keys, pagination } = page.result;
        return {
          keys: keys.map((k) => ({
            id: k.keyId,
            name: k.name ?? null,
            start: k.start,
            createdAt: k.createdAt,
            expires: k.expires ?? null,
            enabled: k.enabled,
            usage: [],
          })),
          cursor: pagination.cursor ?? null,
          hasMore: pagination.hasMore,
        };
      }),
  );

/**
 * Reroll a key the session end user owns, returning the new plaintext secret
 * once. Ownership is verified server-side against the session's external id.
 */
export const rerollKey = createServerFn({ method: "POST" })
  .inputValidator((input: unknown) => rerollKeyRequestSchema.parse(input))
  .handler(
    ({ data }): Promise<RerollKeyResult> =>
      withPortalClient(async (client, token) => {
        const res = await client.portal.rerollKey(
          { portalSession: token },
          { keyId: data.keyId, expiration: data.expiration },
        );
        return { keyId: res.data.keyId, plaintext: res.data.key };
      }),
  );
