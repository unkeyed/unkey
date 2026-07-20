// This file configures the initialization of Sentry on the client.
// The added config here will be used whenever a users loads a page in their browser.
// https://docs.sentry.io/platforms/javascript/guides/nextjs/

import * as Sentry from "@sentry/nextjs";
import {
  DENY_URLS,
  IGNORE_ERRORS,
  createClientErrorFilter,
  createTracesSampler,
  replayPrivacyOptions,
  scrubLog,
  scrubReplayFrame,
  scrubSpanPii,
  scrubTransactionPii,
  scrubUrl,
} from "./lib/sentry";

// Skip Sentry initialization in development or when explicitly disabled
const isSentryDisabled = process.env.NEXT_PUBLIC_SENTRY_DISABLED === "true";
if (process.env.NODE_ENV !== "development" && !isSentryDisabled) {
  Sentry.init({
    dsn: "https://08589d17fe3b4b7e8b70b6c916123ee5@o4510544758046720.ingest.us.sentry.io/4510544758308864",

    // Filter expected tRPC errors and scrub PII from reported events
    beforeSend: createClientErrorFilter(),

    // Transactions bypass `beforeSend`, so scrub secrets/PII from the URLs
    // they carry (request url, Referer header, span http.url/url.full) before
    // sending.
    beforeSendTransaction: scrubTransactionPii,

    // Standalone web-vital (INP) spans bypass `beforeSendTransaction` too;
    // their page URL rides in the `transaction` attribute, so scrub it here.
    beforeSendSpan: scrubSpanPii,

    // Drop non-actionable noise (browser extensions, ResizeObserver, ad-blocker
    // network blips) so genuine errors are not buried.
    ignoreErrors: IGNORE_ERRORS,
    denyUrls: DENY_URLS,

    // Add optional integrations for additional features
    integrations: [
      // Session Replay is private by default — see lib/sentry/replay-privacy.ts
      Sentry.replayIntegration({
        ...replayPrivacyOptions,
        // Recording envelopes bypass `beforeSend`, and replay captures console
        // breadcrumbs off the scope itself, so they need scrubbing here.
        beforeAddRecordingEvent: scrubReplayFrame,
      }),
    ],

    // Use dynamic sampling to reduce non-error traces while ensuring all errors are captured
    tracesSampler: createTracesSampler(),
    // Enable logs to be sent to Sentry
    enableLogs: true,

    // Log envelopes bypass `beforeSend` entirely, and `createClientErrorFilter`
    // routes dropped tRPC errors into logs with their raw message attached.
    beforeSendLog: scrubLog,

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

  // Scrub secrets/PII from the URLs captured on replay events. Replay payloads
  // are not passed through `beforeSend`, so this dedicated processor mirrors the
  // URL scrubbing that error events receive in `scrubEventPii`.
  Sentry.addEventProcessor((event) => {
    if (event.type !== "replay_event") {
      // Return the event unchanged; returning null would drop it.
      return event;
    }

    const replayEvent = event as typeof event & { urls?: string[] };
    if (replayEvent.urls) {
      replayEvent.urls = replayEvent.urls.map(scrubUrl);
    }

    return event;
  });
}

export const onRouterTransitionStart = Sentry.captureRouterTransitionStart;
