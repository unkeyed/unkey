import type { AuthkitResponse } from "@workos-inc/authkit-nextjs";

const inFlightRefreshes = new Map<string, Promise<AuthkitResponse>>();

/**
 * Shares one AuthKit session result between the concurrent requests that carry
 * the same sealed session cookie. AuthKit refresh tokens are single-use, so
 * without this every request in a batch starts its own refresh and all but one
 * are rejected by WorkOS with HTTP 400.
 */
export function coalesceSessionRefresh(
  sessionCookie: string | undefined,
  refresh: () => Promise<AuthkitResponse>,
): Promise<AuthkitResponse> {
  if (!sessionCookie) {
    return refresh();
  }

  const inFlight = inFlightRefreshes.get(sessionCookie);
  if (inFlight) {
    return inFlight;
  }

  const started = refresh().finally(() => {
    inFlightRefreshes.delete(sessionCookie);
  });
  inFlightRefreshes.set(sessionCookie, started);
  return started;
}
