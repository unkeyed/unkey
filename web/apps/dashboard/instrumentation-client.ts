// This file configures the initialization of Sentry on the client.
// The added config here will be used whenever a users loads a page in their browser.
// https://docs.sentry.io/platforms/javascript/guides/nextjs/

import * as Sentry from "@sentry/nextjs";
import { createClientErrorFilter } from "./lib/sentry/error-filter";

Sentry.init({
  dsn: "https://08589d17fe3b4b7e8b70b6c916123ee5@o4510544758046720.ingest.us.sentry.io/4510544758308864",

  // Filter expected tRPC errors from being reported as Sentry errors
  beforeSend: createClientErrorFilter(),

  // Add optional integrations for additional features
  integrations: [
    Sentry.replayIntegration({
      // Unmask all text content by default
      maskAllText: false,
      // Unblock all media elements by default
      blockAllMedia: false,
      // Mask sensitive data selectors
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
        // Password fields (always masked by default)
        "[type='password']",
      ],
      // Block sensitive media
      block: ["[data-sensitive-media]", ".sensitive-media"],
      // Ignore sensitive input events
      ignore: ["[type='password']", "[data-sensitive-input]", ".sensitive-input"],
    }),
  ],

  // Use dynamic sampling to reduce non-error traces while ensuring all errors are captured
  tracesSampler: (samplingContext) => {
    const { name, attributes } = samplingContext;

    // Handle cases where name might be undefined
    if (typeof name !== "string") {
      return 0.1; // Default sampling for unnamed transactions
    }
    // Always sample traces that might contain errors or are critical operations
    if (
      name.includes("error") ||
      name.includes("auth") ||
      name.includes("payment") ||
      name.includes("api/key") ||
      name.includes("trpc") ||
      (attributes?.["http.status_code"] && Number(attributes["http.status_code"]) >= 400)
    ) {
      return 1.0; // 100% sampling for error-prone or critical operations
    }

    // Reduce sampling for health checks and monitoring endpoints
    if (
      name.includes("healthcheck") ||
      name.includes("health") ||
      name.includes("ping") ||
      name.includes("metrics")
    ) {
      return 0.01; // 1% sampling for health checks
    }

    // Reduce sampling for static assets and non-critical operations
    if (
      name.includes("_next/static") ||
      name.includes("favicon") ||
      name.includes("robots.txt") ||
      name.includes("sitemap")
    ) {
      return 0.00; // 0% sampling for static assets
    }

    // Default sampling rate for other operations (significantly reduced)
    return 0.1; // 10% sampling for general operations
  },
  // Enable logs to be sent to Sentry
  enableLogs: true,

  // Define how likely Replay events are sampled.
  // This sets the sample rate to be 10%. You may want this to be 100% while
  // in development and sample at a lower rate in production
  replaysSessionSampleRate: 0.1,

  // Define how likely Replay events are sampled when an error occurs.
  replaysOnErrorSampleRate: 1.0,

  // Enable sending user PII (Personally Identifiable Information)
  // https://docs.sentry.io/platforms/javascript/guides/nextjs/configuration/options/#sendDefaultPii
  sendDefaultPii: false,
});

// Add event processor to scrub URLs in replay events
Sentry.addEventProcessor((event) => {
  // Ensure that we specifically look at replay events
  if (event.type !== "replay_event") {
    // Return the event, otherwise the event will be dropped
    return event;
  }

  // URL scrubbing function to remove potential API keys and tokens
  function urlScrubber(url: string) {
    return url.replace(/([a-zA-Z0-9_-]{20,})/g, "[REDACTED]");
  }

  // Scrub all URLs with the scrubbing function
  const replayEvent = event as { urls?: string[] };
  replayEvent.urls = replayEvent.urls?.map(urlScrubber);

  return event;
});

export const onRouterTransitionStart = Sentry.captureRouterTransitionStart;
