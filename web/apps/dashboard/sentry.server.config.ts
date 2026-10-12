import * as Sentry from "@sentry/nextjs";
import { env } from "./lib/env";
import {
  createServerErrorFilter,
  createTracesSampler,
  scrubLog,
  scrubSpanPii,
  scrubTransactionPii,
} from "./lib/sentry";

const { NEXT_PUBLIC_SENTRY_DSN: dsn } = env();
if (process.env.NODE_ENV !== "development" && dsn) {
  Sentry.init({
    dsn,
    beforeSend: createServerErrorFilter(),
    beforeSendTransaction: scrubTransactionPii,
    beforeSendSpan: scrubSpanPii,
    tracesSampler: createTracesSampler(),
    enableLogs: true,
    beforeSendLog: scrubLog,
    sendDefaultPii: false,
  });
}
