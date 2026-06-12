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
  if (result.success) {
    window.location.href = redirectParam || result.redirectTo;
    return null;
  }

  if ("challengeType" in result) {
    const redirectSuffix = redirectParam ? `&redirect=${encodeURIComponent(redirectParam)}` : "";
    window.location.href = `${SIGN_IN_URL}?challenge=${result.challengeType}${redirectSuffix}`;
    return null;
  }

  if ("organizations" in result && Array.isArray(result.organizations)) {
    const orgsParam = encodeURIComponent(JSON.stringify(result.organizations));
    const redirectSuffix = redirectParam ? `&redirect=${encodeURIComponent(redirectParam)}` : "";
    window.location.href = `${SIGN_IN_URL}?orgs=${orgsParam}${redirectSuffix}`;
    return null;
  }

  return result.message;
}
