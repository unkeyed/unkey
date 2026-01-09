/**
 * Sentry Trace Sampling Configuration
 *
 * This module provides a shared trace sampling function that implements
 * dynamic sampling to reduce non-error traces while ensuring all errors
 * and critical operations are captured.
 */

import type * as Sentry from "@sentry/nextjs";

type SamplingContext = Parameters<NonNullable<Sentry.NodeOptions["tracesSampler"]>>[0];

/**
 * Dynamic trace sampler that adjusts sampling rates based on operation type
 * and potential error conditions.
 *
 * @param samplingContext - The Sentry sampling context containing transaction details
 * @returns Sampling rate between 0.0 (never sample) and 1.0 (always sample)
 */
export function createTracesSampler(): NonNullable<Sentry.NodeOptions["tracesSampler"]> {
  return (samplingContext: SamplingContext): number => {
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

    // Reduce sampling for static assets and non-critical operations (client only)
    if (
      name.includes("_next/static") ||
      name.includes("favicon") ||
      name.includes("robots.txt") ||
      name.includes("sitemap")
    ) {
      return 0.0; // 0% sampling for static assets
    }

    // Default sampling rate for other operations (significantly reduced)
    return 0.1; // 10% sampling for general operations
  };
}
