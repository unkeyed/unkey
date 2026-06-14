import { isSafeRedirectPath } from "@/app/auth/sign-in/redirect-utils";
import { SIGN_IN_URL, type VerificationResult } from "@/lib/auth/types";

/**
 * Applies a verification result on the client: navigates on success or when
 * another step (org selection, follow-up challenge) is required, and returns
 * the error message to display otherwise.
 */
export function applyVerificationResult(
  result: VerificationResult,
  redirectParam?: string | null,
): string | null {
  const safeRedirect = redirectParam && isSafeRedirectPath(redirectParam) ? redirectParam : null;

  if (result.success) {
    window.location.href = safeRedirect || result.redirectTo;
    return null;
  }

  if ("challengeType" in result) {
    const redirectSuffix = safeRedirect ? `&redirect=${encodeURIComponent(safeRedirect)}` : "";
    window.location.href = `${SIGN_IN_URL}?challenge=${result.challengeType}${redirectSuffix}`;
    return null;
  }

  if ("organizations" in result && Array.isArray(result.organizations)) {
    // /auth/continue auto-selects the last used organization server-side and
    // falls back to the manual selector on the sign-in page.
    const orgsParam = encodeURIComponent(JSON.stringify(result.organizations));
    const redirectSuffix = safeRedirect ? `&redirect=${encodeURIComponent(safeRedirect)}` : "";
    window.location.href = `/auth/continue?orgs=${orgsParam}${redirectSuffix}`;
    return null;
  }

  return result.message;
}
