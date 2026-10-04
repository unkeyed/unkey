import type * as Sentry from "@sentry/nextjs";

type ReplayOptions = NonNullable<Parameters<typeof Sentry.replayIntegration>[0]>;

const SENSITIVE_MARKERS = [
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
] as const;

function withNestedFields(selectors: readonly string[]): string[] {
  return selectors.flatMap((selector) => [selector, `${selector} input`, `${selector} textarea`]);
}

export const replayPrivacyOptions: ReplayOptions = {
  maskAllText: false,
  maskAllInputs: false,
  blockAllMedia: false,
  mask: ["[type='email']", "[type='password']", ...withNestedFields(SENSITIVE_MARKERS)],
  unmask: ["[data-sentry-unmask]"],
  block: ["[data-sensitive-media]", ".sensitive-media"],
  unblock: ["[data-sentry-unblock]"],
  ignore: ["[type='password']", "[data-sensitive-input]", ".sensitive-input"],
  networkDetailAllowUrls: [],
  networkCaptureBodies: false,
  networkRequestHeaders: [],
  networkResponseHeaders: [],
};

export const replaySampleRates = {
  replaysSessionSampleRate: 0,
  replaysOnErrorSampleRate: 1,
} as const;

export const replayMaskedLogSectionTitles = new Set([
  "Request Body",
  "Request Header",
  "Response Body",
  "Response Header",
]);
