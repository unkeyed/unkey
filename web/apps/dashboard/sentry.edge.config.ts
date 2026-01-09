// This file configures the initialization of Sentry for edge features (middleware, edge routes, and so on).
// The config you add here will be used whenever one of the edge features is loaded.
// Note that this config is unrelated to the Vercel Edge Runtime and is also required when running locally.
// https://docs.sentry.io/platforms/javascript/guides/nextjs/

import * as Sentry from "@sentry/nextjs";
import { createEdgeErrorFilter } from "./lib/sentry/error-filter";

Sentry.init({
  dsn: "https://08589d17fe3b4b7e8b70b6c916123ee5@o4510544758046720.ingest.us.sentry.io/4510544758308864",

  // Filter expected tRPC errors from being reported as Sentry errors
  beforeSend: createEdgeErrorFilter(),

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

    // Default sampling rate for other operations (significantly reduced)
    return 0.1; // 10% sampling for general operations
  },

  // Enable logs to be sent to Sentry
  enableLogs: true,

  // Enable sending user PII (Personally Identifiable Information)
  // https://docs.sentry.io/platforms/javascript/guides/nextjs/configuration/options/#sendDefaultPii
  sendDefaultPii: false,
});
