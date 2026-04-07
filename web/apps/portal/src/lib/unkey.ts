import { Unkey } from "@unkey/api";
import { env } from "./env";

/**
 * Create an Unkey client authenticated with a portal session token.
 * The session token is passed as the rootKey — the API's portalSessionAuth
 * mechanism accepts it and resolves workspace_id + externalId.
 */
export function createPortalClient(sessionToken: string) {
  return new Unkey({
    rootKey: sessionToken,
    serverURL: env().UNKEY_API_URL,
  });
}
