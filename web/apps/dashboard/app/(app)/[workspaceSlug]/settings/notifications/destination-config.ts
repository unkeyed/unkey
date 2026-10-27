import type {
  V2NotificationsCreateDestinationRequestBody,
  V2NotificationsUpdateDestinationRequestBody,
} from "@unkey/api/models/components";
import { z } from "zod";
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

export function generateSigningSecret(): string {
  const bytes = crypto.getRandomValues(new Uint8Array(24));
  return `whsec_${Array.from(bytes, (byte) => byte.toString(16).padStart(2, "0")).join("")}`;
}

export function destinationFormSchema(editing: boolean) {
  return z
    .object({
      channel: z.enum(["discord", "slack", "webhook"]).nullable(),
      name: z.string().trim().min(1, "Enter a name.").max(128, "Use 128 characters or fewer."),
      url: z.string().max(8192, "Use 8192 characters or fewer."),
      headersMode: z.enum(["keep", "replace"]),
      headers: z.array(z.object({ name: z.string(), value: z.string() })),
      enabled: z.boolean(),
    })
    .superRefine((values, ctx) => {
      if (!values.channel) {
        ctx.addIssue({ code: "custom", path: ["channel"], message: "Choose a channel." });
        return;
      }
      const problem =
        editing && values.url.trim() === ""
          ? null
          : destinationUrlProblem(values.channel, values.url);
      if (problem) {
        ctx.addIssue({ code: "custom", path: ["url"], message: problem });
      }
      if (values.channel !== "webhook" || values.headersMode === "keep") {
        return;
      }
      if (values.headers.length > 16) {
        ctx.addIssue({ code: "custom", path: ["headers"], message: "Use 16 headers or fewer." });
      }
      const names = new Set<string>();
      values.headers.forEach((header, index) => {
        const problem =
          header.name.trim().length > 128
            ? "Use 128 characters or fewer."
            : headerNameProblem(header.name);
        if (problem) {
          ctx.addIssue({ code: "custom", path: ["headers", index, "name"], message: problem });
        }
        const name = header.name.trim().toLowerCase();
        if (names.has(name)) {
          ctx.addIssue({
            code: "custom",
            path: ["headers", index, "name"],
            message: "Duplicate header.",
          });
        }
        names.add(name);
        if (header.value === "") {
          ctx.addIssue({
            code: "custom",
            path: ["headers", index, "value"],
            message: "Enter a replacement value.",
          });
        }
        if (header.value.length > 2048) {
          ctx.addIssue({
            code: "custom",
            path: ["headers", index, "value"],
            message: "Use 2048 characters or fewer.",
          });
        }
      });
    });
}

export type DestinationFormValues = z.infer<ReturnType<typeof destinationFormSchema>>;

export function destinationCreateInput(
  values: DestinationFormValues,
  idempotencyKey: string,
  signingSecret: string | null,
): V2NotificationsCreateDestinationRequestBody {
  if (!values.channel || (values.channel === "webhook" && !signingSecret)) {
    throw new Error("A destination requires a channel and webhooks require a signing secret");
  }
  return {
    name: values.name.trim(),
    channel: values.channel,
    enabled: values.enabled,
    url: values.url.trim(),
    headers:
      values.channel === "webhook"
        ? values.headers.map(({ name, value }) => ({ name: name.trim(), value }))
        : undefined,
    signingSecret: values.channel === "webhook" ? (signingSecret ?? undefined) : undefined,
    idempotencyKey,
  };
}

export function destinationUpdateInput(
  destinationId: string,
  values: DestinationFormValues,
  signingSecret: string | null,
): V2NotificationsUpdateDestinationRequestBody {
  return {
    destinationId,
    name: values.name.trim(),
    enabled: values.enabled,
    url: values.url.trim() || undefined,
    headers:
      values.channel === "webhook" && values.headersMode === "replace"
        ? values.headers.map(({ name, value }) => ({ name: name.trim(), value }))
        : undefined,
    signingSecret: values.channel === "webhook" ? (signingSecret ?? undefined) : undefined,
  };
}
