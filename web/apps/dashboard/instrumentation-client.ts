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

const dsn = process.env.NEXT_PUBLIC_SENTRY_DSN;
if (process.env.NODE_ENV !== "development" && dsn) {
  Sentry.init({
    dsn,

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
    replaysSessionSampleRate: 0,
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
