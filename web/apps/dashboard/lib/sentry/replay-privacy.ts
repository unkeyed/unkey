import type * as Sentry from "@sentry/nextjs";

type ReplayOptions = NonNullable<Parameters<typeof Sentry.replayIntegration>[0]>;

export const replayPrivacyOptions: ReplayOptions = {
  maskAllText: true,
  maskAllInputs: true,
  blockAllMedia: true,
  mask: [
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
  ],
  unmask: ["[data-sentry-unmask]"],
  block: ["[data-sensitive-media]", ".sensitive-media"],
  unblock: ["[data-sentry-unblock]"],
  ignore: ["[type='password']", "[data-sensitive-input]", ".sensitive-input"],
  networkDetailAllowUrls: [],
  networkCaptureBodies: false,
  networkRequestHeaders: [],
  networkResponseHeaders: [],
};
