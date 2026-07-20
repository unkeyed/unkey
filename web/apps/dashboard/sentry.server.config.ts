// This file configures the initialization of Sentry on the server.
// The config you add here will be used whenever the server handles a request.
// https://docs.sentry.io/platforms/javascript/guides/nextjs/

import * as Sentry from "@sentry/nextjs";
import { env } from "./lib/env";
import {
  createServerErrorFilter,
  createTracesSampler,
  scrubLog,
  scrubSpanPii,
  scrubTransactionPii,
} from "./lib/sentry";

// Skip Sentry initialization in development or when explicitly disabled
const envVars = env();
if (process.env.NODE_ENV !== "development" && !envVars.SENTRY_DISABLED) {
  Sentry.init({
    dsn: "https://08589d17fe3b4b7e8b70b6c916123ee5@o4510544758046720.ingest.us.sentry.io/4510544758308864",

    // Filter expected tRPC errors from being reported as Sentry errors
    beforeSend: createServerErrorFilter(),

    // Transactions bypass `beforeSend`, so scrub secrets/PII from the URLs
    // they carry (request url, Referer header, span http.url/url.full) before
    // sending.
    beforeSendTransaction: scrubTransactionPii,

    // Standalone spans bypass `beforeSendTransaction` too; scrub them here.
    beforeSendSpan: scrubSpanPii,

    // Use dynamic sampling to reduce non-error traces while ensuring all errors are captured
    tracesSampler: createTracesSampler(),

    // Enable logs to be sent to Sentry
    enableLogs: true,

    // Log envelopes bypass `beforeSend` entirely, and the error filter routes
    // dropped tRPC errors into logs with their raw message attached.
    beforeSendLog: scrubLog,

    // Enable sending user PII (Personally Identifiable Information)
    // https://docs.sentry.io/platforms/javascript/guides/nextjs/configuration/options/#sendDefaultPii
    sendDefaultPii: false,
  });
}
