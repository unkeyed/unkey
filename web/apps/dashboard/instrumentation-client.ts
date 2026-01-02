// This file configures the initialization of Sentry on the client.
// The added config here will be used whenever a users loads a page in their browser.
// https://docs.sentry.io/platforms/javascript/guides/nextjs/

import * as Sentry from "@sentry/nextjs";

Sentry.init({
  dsn: "https://08589d17fe3b4b7e8b70b6c916123ee5@o4510544758046720.ingest.us.sentry.io/4510544758308864",

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

  // Define how likely traces are sampled. Adjust this value in production, or use tracesSampler for greater control.
  tracesSampleRate: 1,
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
