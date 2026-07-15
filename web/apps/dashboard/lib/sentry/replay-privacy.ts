/**
 * Sentry Session Replay privacy configuration.
 *
 * Replay is configured to be PRIVATE BY DEFAULT. Sentry's defaults already mask
 * all text and inputs and block media, but those defaults are easy to weaken by
 * accident (the previous config set `maskAllText: false`, which leaked every
 * label, workspace name, key name and email rendered anywhere in the UI). We
 * pin the safe defaults explicitly here and only allow visibility through an
 * opt-in attribute, so a replay can never silently start leaking PII.
 *
 * Visibility opt-in:
 *   - add `data-sentry-unmask` to an element to reveal its (non-sensitive) text
 *   - add `data-sentry-unblock` to an element to record a (non-sensitive) image
 *
 * Never add these attributes to anything that can render user data, keys,
 * emails, tokens, or workspace/identity identifiers.
 */

import type * as Sentry from "@sentry/nextjs";

type ReplayOptions = NonNullable<Parameters<typeof Sentry.replayIntegration>[0]>;

export const replayPrivacyOptions: ReplayOptions = {
  // Private by default: mask every text node and input, block every media
  // element. These match Sentry's defaults but are pinned so they can't be
  // weakened without a deliberate review of this file.
  maskAllText: true,
  maskAllInputs: true,
  blockAllMedia: true,

  // Defense in depth: explicitly mask known-sensitive selectors even if the
  // global masks above are ever relaxed.
  mask: [
    // Email addresses
    "[type='email']",
    ".email",
    "[data-email]",
    // API keys and secrets
    "[data-api-key]",
    "[data-secret]",
    "[data-token]",
    ".api-key",
    ".secret",
    ".token",
    // Unkey specific sensitive data
    "[data-unkey-root-key]",
    ".unkey-root-key",
    "[data-external-id]",
    ".external-id",
    // Password fields
    "[type='password']",
  ],

  // Explicit opt-in to reveal non-sensitive UI chrome (nav, static labels).
  unmask: ["[data-sentry-unmask]"],

  // Block sensitive media outright; allow opt-in for known-safe images.
  block: ["[data-sensitive-media]", ".sensitive-media"],
  unblock: ["[data-sentry-unblock]"],

  // Never record values typed into password/sensitive inputs.
  ignore: ["[type='password']", "[data-sensitive-input]", ".sensitive-input"],

  // Do NOT capture request/response bodies or headers in replay network
  // breadcrumbs. Leaving these empty keeps Sentry's default of recording only
  // method, status, and (scrubbed) URL — bodies that could contain keys or PII
  // are never collected.
  networkDetailAllowUrls: [],
  networkCaptureBodies: false,
  networkRequestHeaders: [],
  networkResponseHeaders: [],
};
