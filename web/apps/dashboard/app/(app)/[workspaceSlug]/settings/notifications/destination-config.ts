import type { DestinationChannel } from "./notification-model";

const DISCORD_HOSTS = new Set([
  "discord.com",
  "discordapp.com",
  "ptb.discord.com",
  "canary.discord.com",
]);

const HEADER_NAME = /^[!#$%&'*+.^_`|~0-9A-Za-z-]+$/;

export const URL_PLACEHOLDER: Record<DestinationChannel, string> = {
  discord: "https://discord.com/api/webhooks/…",
  slack: "https://hooks.slack.com/services/…",
  webhook: "https://example.com/unkey/notifications",
};

export function destinationUrlProblem(channel: DestinationChannel, raw: string): string | null {
  let url: URL;
  try {
    url = new URL(raw.trim());
  } catch {
    return "Enter a full URL, starting with https://.";
  }
  if (url.protocol !== "https:") {
    return "Use an https:// URL.";
  }
  if (channel === "discord") {
    return DISCORD_HOSTS.has(url.hostname) && url.pathname.startsWith("/api/webhooks/")
      ? null
      : "Use a Discord webhook URL from Channel settings > Integrations > Webhooks.";
  }
  if (channel === "slack") {
    return url.hostname === "hooks.slack.com" && url.pathname.startsWith("/services/")
      ? null
      : "Use a Slack incoming webhook URL that starts with https://hooks.slack.com/services/.";
  }
  return null;
}

export function headerNameProblem(name: string): string | null {
  if (name.trim() === "") {
    return "Enter a header name.";
  }
  return HEADER_NAME.test(name.trim()) ? null : "Use letters, digits, and dashes only.";
}

export function urlHint(raw: string): string {
  const url = new URL(raw.trim());
  const path = url.pathname.replace(/\/+$/, "");
  const firstSegment = path.split("/").filter(Boolean)[0];
  const tail = path.slice(-4);
  if (!firstSegment) {
    return url.hostname;
  }
  return `${url.hostname}/${firstSegment}/…${tail}`;
}

export function generateSigningSecret(): string {
  const bytes = crypto.getRandomValues(new Uint8Array(24));
  return `whsec_${Array.from(bytes, (byte) => byte.toString(16).padStart(2, "0")).join("")}`;
}

export function secretHint(secret: string): string {
  return `whsec_…${secret.slice(-4)}`;
}
