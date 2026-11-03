import type { DeviceLogin } from "./device-api";

/**
 * An app that started the login through OAuth. A third-party name is self-reported, so its host
 * is shown too. host is null for Unkey's hosted MCP server, which the API names itself.
 */
export type OAuthClient = { name: string; host: string | null };

export function oauthClientOf(
  login: Pick<DeviceLogin, "oauthClientId" | "oauthClientFirstParty" | "deviceName">,
): OAuthClient | null {
  if (!login.oauthClientId) {
    return null;
  }
  const host = new URL(login.oauthClientId).host;
  if (login.oauthClientFirstParty) {
    return { name: login.deviceName ?? "Unkey MCP", host: null };
  }
  return { name: login.deviceName ?? host, host };
}
