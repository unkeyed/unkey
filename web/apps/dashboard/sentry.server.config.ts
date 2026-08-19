import * as Sentry from "@sentry/nextjs";
import { env } from "./lib/env";
import {
  createServerErrorFilter,
  createTracesSampler,
  scrubLog,
  scrubSpanPii,
  scrubTransactionPii,
} from "./lib/sentry";

const envVars = env();
if (process.env.NODE_ENV !== "development" && !envVars.SENTRY_DISABLED) {
  Sentry.init({
    dsn: "https://08589d17fe3b4b7e8b70b6c916123ee5@o4510544758046720.ingest.us.sentry.io/4510544758308864",
    beforeSend: createServerErrorFilter(),
    beforeSendTransaction: scrubTransactionPii,
    beforeSendSpan: scrubSpanPii,
    tracesSampler: createTracesSampler(),
    enableLogs: true,
    beforeSendLog: scrubLog,
    sendDefaultPii: false,
  });
}
