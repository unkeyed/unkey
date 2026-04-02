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

const REDIRECT_STORAGE_KEY = "unkey_auth_redirect";

/**
 * Persists a validated redirect URL to sessionStorage.
 * Called once when the sign-in page first loads with a redirect param.
 */
export function saveRedirectUrl(url: string): void {
  if (isSafeRedirectPath(url)) {
    try {
      sessionStorage.setItem(REDIRECT_STORAGE_KEY, url);
    } catch {
      // sessionStorage unavailable (e.g. private browsing edge cases)
    }
  }
}

/**
 * Reads and clears the stored redirect URL from sessionStorage.
 * Returns null if nothing is stored or the value fails validation.
 */
export function consumeRedirectUrl(): string | null {
  try {
    const url = sessionStorage.getItem(REDIRECT_STORAGE_KEY);
    sessionStorage.removeItem(REDIRECT_STORAGE_KEY);
    if (url && isSafeRedirectPath(url)) {
      return url;
    }
  } catch {
    // sessionStorage unavailable
  }
  return null;
}

/**
 * Rewrites the workspace slug in a redirect URL to match the selected workspace.
 * URLs follow the pattern /:workspaceSlug/rest/of/path.
 * If the slug in the URL doesn't match the authenticated workspace, replace it.
 * Returns null for unsafe redirect URLs to prevent open redirects.
 */
export function resolveRedirectUrl(
  redirectParam: string | null,
  workspaceSlug?: string,
): string | null {
  if (!redirectParam || !isSafeRedirectPath(redirectParam)) {
    return null;
  }

  if (!workspaceSlug) {
    return redirectParam;
  }

  // Separate path from query string/hash to only rewrite the path
  const queryStart = redirectParam.indexOf("?");
  const hashStart = redirectParam.indexOf("#");
  const suffixStart =
    queryStart >= 0 && hashStart >= 0
      ? Math.min(queryStart, hashStart)
      : queryStart >= 0
        ? queryStart
        : hashStart >= 0
          ? hashStart
          : -1;
  const pathOnly = suffixStart >= 0 ? redirectParam.slice(0, suffixStart) : redirectParam;
  const suffix = suffixStart >= 0 ? redirectParam.slice(suffixStart) : "";

  // URLs are like /workspace-slug/settings/billing
  // Split into segments: ["", "workspace-slug", "settings", "billing"]
  const segments = pathOnly.split("/");
  if (segments.length >= 2 && segments[1] && segments[1] !== workspaceSlug) {
    segments[1] = workspaceSlug;
    return segments.join("/") + suffix;
  }

  return redirectParam;
}
