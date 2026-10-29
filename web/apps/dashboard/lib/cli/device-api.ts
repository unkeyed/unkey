import { getUnkeyClient } from "@/lib/unkey-client";
import type { Unkey } from "@unkey/api";
import type { RequestOptions } from "@unkey/api/lib/sdks";

type DeviceLoginResponse = Awaited<ReturnType<Unkey["cli"]["getDeviceLogin"]>>["data"];
type DecisionResponse = Awaited<ReturnType<Unkey["cli"]["decideDeviceLogin"]>>["data"];
export type CliSession = Awaited<
  ReturnType<Unkey["cli"]["listSessions"]>
>["data"]["sessions"][number];

type DeviceLoginBase = Omit<DeviceLoginResponse, "delivery" | "client">;

export type OAuthClient = {
  id: string;
  name: string;
  redirectUri: string;
  redirectHost: string;
};

export type DeviceLogin =
  | (DeviceLoginBase & { delivery: "device" })
  | (DeviceLoginBase & { delivery: "redirect"; client: OAuthClient });

export type Decision = { delivery: "device" } | { delivery: "redirect"; redirectTo: string };

const requestOptions = { retries: { strategy: "none" } } satisfies RequestOptions;

export function parseDeviceLogin(response: DeviceLoginResponse): DeviceLogin {
  const { delivery, client, ...login } = response;
  if (delivery === "device") {
    return { ...login, delivery };
  }
  if (!client) {
    throw new Error("The login is missing the app that asked for access.");
  }
  return {
    ...login,
    delivery,
    client: { ...client, redirectHost: webUrl(client.redirectUri).host },
  };
}

// The browser follows redirectTo blindly, so it must stay on the origin the
// approver was shown before deciding.
export function parseDecision(login: DeviceLogin, response: DecisionResponse): Decision {
  if (login.delivery === "device") {
    return { delivery: "device" };
  }
  if (!response.redirectTo) {
    throw new Error("The decision is missing where to send the browser.");
  }
  const target = webUrl(response.redirectTo);
  if (target.origin !== webUrl(login.client.redirectUri).origin) {
    throw new Error("The decision points somewhere other than the app's redirect address.");
  }
  return { delivery: "redirect", redirectTo: target.href };
}

function webUrl(value: string): URL {
  const url = URL.canParse(value) ? new URL(value) : null;
  if (!url || (url.protocol !== "https:" && url.protocol !== "http:")) {
    throw new Error("Expected an http or https address.");
  }
  return url;
}

export async function getDeviceLogin(userCode: string): Promise<DeviceLogin> {
  const response = await getUnkeyClient().cli.getDeviceLogin({ userCode }, requestOptions);
  return parseDeviceLogin(response.data);
}

export async function approveDeviceLogin(
  login: DeviceLogin,
  permissions: string[],
): Promise<Decision> {
  const response = await getUnkeyClient().cli.decideDeviceLogin(
    { userCode: login.userCode, permissions, decision: "approve" },
    requestOptions,
  );
  return parseDecision(login, response.data);
}

export async function denyDeviceLogin(login: DeviceLogin): Promise<Decision> {
  const response = await getUnkeyClient().cli.decideDeviceLogin(
    { userCode: login.userCode, decision: "deny" },
    requestOptions,
  );
  return parseDecision(login, response.data);
}

export async function listCliSessions(): Promise<CliSession[]> {
  const response = await getUnkeyClient().cli.listSessions(requestOptions);
  return response.data.sessions;
}

export async function revokeCliSession(sessionId: string): Promise<void> {
  await getUnkeyClient().cli.revokeSession({ sessionId }, requestOptions);
}
