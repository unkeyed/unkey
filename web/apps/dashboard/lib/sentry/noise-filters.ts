/**
 * Sentry noise filters.
 *
 * "Catch all the errors it should" also means NOT drowning real errors in noise
 * we can never act on: browser-extension crashes, benign ResizeObserver loop
 * warnings, and network blips from ad-blockers. These run via Sentry's built-in
 * `ignoreErrors` (message/exception matching) and `denyUrls` (stack-frame origin
 * matching) so they apply before events are queued.
 */

/**
 * Error messages that are non-actionable for us. Matched against the event
 * title and exception values (substring or regex).
 */
export const IGNORE_ERRORS: (string | RegExp)[] = [
  // Benign layout notification fired by browsers; not a real error.
  "ResizeObserver loop limit exceeded",
  "ResizeObserver loop completed with undelivered notifications",

  // Network errors that are almost always client connectivity / ad-blockers,
  // not application bugs. tRPC/query layers surface actionable failures.
  "Failed to fetch",
  "NetworkError when attempting to fetch resource",
  "Load failed",
  "The network connection was lost",

  // Navigation aborts and benign cancellations.
  "AbortError",
  "The operation was aborted",
  "Non-Error promise rejection captured",

  // Browser-extension and injected-script noise.
  /extension\//i,
  "Can't find variable: ZiteReader",
  /^Script error\.?$/,
];

/**
 * Stack-frame URL patterns whose errors originate outside our code (browser
 * extensions, injected third-party scripts). Events whose top frames match are
 * dropped.
 */
export const DENY_URLS: (string | RegExp)[] = [
  // Browser extensions
  /extensions\//i,
  /^chrome-extension:\/\//i,
  /^moz-extension:\/\//i,
  /^safari-extension:\/\//i,
  /^safari-web-extension:\/\//i,
  // Injected scripts referenced from anonymous origins
  /^chrome:\/\//i,
];
