import type { DeviceLogin } from "./device-api";

/** An app that started the login through OAuth. Its name is self-reported, so the host is shown too. */
export type OAuthClient = { name: string; host: string };

export function oauthClientOf(
  login: Pick<DeviceLogin, "oauthClientId" | "deviceName">,
): OAuthClient | null {
  if (!login.oauthClientId) {
    return null;
  }
  const host = new URL(login.oauthClientId).host;
  return { name: login.deviceName ?? host, host };
}
