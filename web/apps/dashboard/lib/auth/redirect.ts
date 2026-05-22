const DEFAULT_AUTH_REDIRECT = "/apis";

/**
 * Validates that a redirect URL is a safe relative path.
 * Uses an allowlist approach — every path segment must contain only
 * alphanumeric characters, hyphens, underscores, dots, or tildes.
 * Query strings and hash fragments are allowed (they stay on the same origin).
 */
export function isSafeRedirectPath(url: string): boolean {
  // Must start with exactly one forward slash
  if (!url.startsWith("/") || url.startsWith("//")) {
    return false;
  }

  // Strip query string and hash before validating path segments
  const pathOnly = url.split("?")[0].split("#")[0];

  // Every segment after splitting on "/" must only contain safe characters.
  // The first element is always "" (before the leading slash).
  const segments = pathOnly.split("/");
  for (let i = 1; i < segments.length; i++) {
    // Allow empty segments only for trailing slash (last segment)
    if (segments[i] === "" && i === segments.length - 1) {
      continue;
    }
    // Segments must be non-empty and contain only URL-safe path characters
    if (!segments[i] || !/^[a-zA-Z0-9\-._~!$&'()*+,;=@%]+$/.test(segments[i])) {
      return false;
    }
  }

  return true;
}

export function sanitizeRedirectPath(url: string | null | undefined): string {
  return url && isSafeRedirectPath(url) ? url : DEFAULT_AUTH_REDIRECT;
}
