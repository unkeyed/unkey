import { env } from "@/lib/env";

/**
 * Cookie Security Utilities
 *
 * This module handles environment-aware cookie security settings to resolve
 * Safari's restriction on secure cookies over HTTP connections.
 *
 * PROBLEM:
 * Safari browsers don't allow setting cookies with the `secure` flag when
 * running over HTTP (like localhost during development). This causes cookie
 * operations to fail silently in development environments.
 *
 * SOLUTION:
 * - Use `secure: true` in production and preview (both HTTPS)
 * - Use `secure: false` in development (HTTP/localhost)
 *
 * This approach maintains security in production while allowing development
 * to work seamlessly across all browsers, including Safari.
 */

/**
 * Determines if cookies should be secure based on the environment.
 *
 * @returns true if cookies should be secure (production/preview), false otherwise (development)
 */
export function shouldUseSecureCookies(): boolean {
  const environment = env();
  return environment.VERCEL_ENV !== "development";
}

/**
 * Get default cookie options with environment-appropriate security settings
 */
export function getDefaultCookieOptions() {
  return {
    httpOnly: true,
    secure: shouldUseSecureCookies(),
    sameSite: "strict" as const,
    path: "/",
  };
}

/**
 * Get cookie options for authentication flows (typically uses "lax" sameSite for OAuth)
 */
export function getAuthCookieOptions() {
  return {
    httpOnly: true,
    secure: shouldUseSecureCookies(),
    sameSite: "lax" as const,
    path: "/",
  };
}
