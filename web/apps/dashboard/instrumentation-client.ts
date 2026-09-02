import * as Sentry from "@sentry/nextjs";
import {
  createClientErrorFilter,
  createTracesSampler,
  DENY_URLS,
  IGNORE_ERRORS,
  replayPrivacyOptions,
  scrubLog,
  scrubReplayFrame,
  scrubSpanPii,
  scrubTransactionPii,
  scrubUrl,
} from "./lib/sentry";

const isSentryDisabled = process.env.NEXT_PUBLIC_SENTRY_DISABLED === "true";
if (process.env.NODE_ENV !== "development" && !isSentryDisabled) {
  Sentry.init({
    dsn: "https://08589d17fe3b4b7e8b70b6c916123ee5@o4510544758046720.ingest.us.sentry.io/4510544758308864",

    beforeSend: createClientErrorFilter(),
    beforeSendTransaction: scrubTransactionPii,
    beforeSendSpan: scrubSpanPii,
    ignoreErrors: IGNORE_ERRORS,
    denyUrls: DENY_URLS,
    integrations: [
      Sentry.replayIntegration({
        ...replayPrivacyOptions,
        beforeAddRecordingEvent: scrubReplayFrame,
      }),
    ],

    tracesSampler: createTracesSampler(),
    enableLogs: true,
    beforeSendLog: scrubLog,
    replaysSessionSampleRate: 0.1,
    replaysOnErrorSampleRate: 1.0,
    sendDefaultPii: false,
  });

  Sentry.addEventProcessor((event) => {
    if (event.type !== "replay_event") {
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
