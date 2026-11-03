import { getUnkeyClient } from "@/lib/unkey-client";
import type { Unkey } from "@unkey/api";
import type { RequestOptions } from "@unkey/api/lib/sdks";

export type DeviceLogin = Awaited<ReturnType<Unkey["cli"]["getDeviceLogin"]>>["data"];
export type CliSession = Awaited<
  ReturnType<Unkey["cli"]["listSessions"]>
>["data"]["sessions"][number];

const requestOptions = { retries: { strategy: "none" } } satisfies RequestOptions;

export async function getDeviceLogin(userCode: string): Promise<DeviceLogin> {
  const response = await getUnkeyClient().cli.getDeviceLogin({ userCode }, requestOptions);
  return response.data;
}

/** Resolves to where the browser goes next, which only OAuth logins have. */
export async function approveDeviceLogin(
  request: Required<Omit<Parameters<Unkey["cli"]["decideDeviceLogin"]>[0], "decision">>,
): Promise<string | undefined> {
  const response = await getUnkeyClient().cli.decideDeviceLogin(
    { ...request, decision: "approve" },
    requestOptions,
  );
  return clientRedirect(response.data.redirectUrl);
}

/** Resolves to where the browser goes next, which only OAuth logins have. */
export async function denyDeviceLogin(userCode: string): Promise<string | undefined> {
  const response = await getUnkeyClient().cli.decideDeviceLogin(
    { userCode, decision: "deny" },
    requestOptions,
  );
  return clientRedirect(response.data.redirectUrl);
}

/**
 * The API only issues https or loopback http redirects. Checking the scheme
 * again keeps a bad response from running script in the dashboard origin.
 */
export function clientRedirect(redirectUrl: string | undefined): string | undefined {
  if (!redirectUrl) {
    return undefined;
  }
  const protocol = URL.parse(redirectUrl)?.protocol;
  if (protocol !== "https:" && protocol !== "http:") {
    throw new Error("The API returned a redirect the dashboard will not open.");
  }
  return redirectUrl;
}

export async function listCliSessions(): Promise<CliSession[]> {
  const response = await getUnkeyClient().cli.listSessions(requestOptions);
  return response.data.sessions;
}

export async function revokeCliSession(sessionId: string): Promise<void> {
  await getUnkeyClient().cli.revokeSession({ sessionId }, requestOptions);
}
