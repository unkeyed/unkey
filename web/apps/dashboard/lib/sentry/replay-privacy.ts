import type * as Sentry from "@sentry/nextjs";

type ReplayOptions = NonNullable<Parameters<typeof Sentry.replayIntegration>[0]>;

// Text is visible by default so replays show what the user did. Anything that
// renders secrets or PII must carry `data-sentry-mask` (or a selector below).
export const replayPrivacyOptions: ReplayOptions = {
  maskAllText: false,
  maskAllInputs: true,
  blockAllMedia: false,
  mask: [
    "[data-sentry-mask]",
    "[type='email']",
    ".email",
    "[data-email]",
    "[data-api-key]",
    "[data-secret]",
    "[data-token]",
    ".api-key",
    ".secret",
    ".token",
    "[data-unkey-root-key]",
    ".unkey-root-key",
    "[data-external-id]",
    ".external-id",
    "[type='password']",
    "[data-sonner-toast]",
  ],
  unmask: [],
  block: ["img", "video", "picture", "[data-sensitive-media]", ".sensitive-media"],
  unblock: [],
  ignore: ["[type='password']", "[data-sensitive-input]", ".sensitive-input"],
  networkDetailAllowUrls: [],
  networkCaptureBodies: false,
  networkRequestHeaders: [],
  networkResponseHeaders: [],
};
