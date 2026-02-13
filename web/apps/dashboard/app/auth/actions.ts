"use server";

import {
  deleteCookie,
  getCookie,
  setCookies,
  setLastUsedOrgCookie,
  setSessionCookie,
} from "@/lib/auth/cookies";
import { auth } from "@/lib/auth/server";
import {
  AuthErrorCode,
  type AuthErrorResponse,
  type EmailAuthResult,
  type NavigationResponse,
  type OAuthResult,
  PENDING_SESSION_COOKIE,
  type PendingTurnstileResponse,
  type SignInViaOAuthOptions,
  type UserData,
  type VerificationResult,
  errorMessages,
} from "@/lib/auth/types";
import { requireEmailMatch } from "@/lib/auth/utils";
import { env } from "@/lib/env";
import { Ratelimit } from "@unkey/ratelimit";
import { cookies, headers } from "next/headers";
import { redirect } from "next/navigation";

// Helper to extract request metadata for Radar
async function getRequestMetadata() {
  const headersList = await headers();
  const ipAddress =
    headersList.get("x-forwarded-for")?.split(",")[0].trim() ||
    headersList.get("x-real-ip") ||
    undefined;
  const userAgent = headersList.get("user-agent") || undefined;

  return { ipAddress, userAgent };
}

// Turnstile verification helper
async function verifyTurnstileToken(token: string): Promise<boolean> {
  const environment = env();
  const secretKey = environment.CLOUDFLARE_TURNSTILE_SECRET_KEY;

  if (!secretKey) {
    return false;
  }

  try {
    const response = await fetch("https://challenges.cloudflare.com/turnstile/v0/siteverify", {
      method: "POST",
      headers: {
        "Content-Type": "application/json",
      },
      body: JSON.stringify({
        secret: secretKey,
        response: token,
      }),
    });

    if (!response.ok) {
      return false;
    }

    const data = await response.json();

    return data.success === true;
  } catch (_error) {
    return false;
  }
}

// Authentication Actions
export async function signUpViaEmail(params: UserData): Promise<EmailAuthResult> {
  const metadata = getRequestMetadata();
  return await auth.signUpViaEmail({ ...params, ...metadata });
}

export async function signInViaEmail(email: string): Promise<EmailAuthResult> {
  const metadata = getRequestMetadata();
  return await auth.signInViaEmail({ email, ...metadata });
}

export async function verifyAuthCode(params: {
  email: string;
  code: string;
  invitationToken?: string;
}): Promise<VerificationResult> {
  const { email, code, invitationToken } = params;
  try {
    if (invitationToken) {
      await requireEmailMatch({ email, invitationToken });
    }

    const result = await auth.verifyAuthCode({ email, code, invitationToken });

    // If we have an invitation token and got organization_selection_required,
    // automatically select the invited organization
    if (
      invitationToken &&
      !result.success &&
      result.code === AuthErrorCode.ORGANIZATION_SELECTION_REQUIRED &&
      "organizations" in result
    ) {
      try {
        // Get the invitation details to find the organization ID
        const invitation = await auth.getInvitation(invitationToken);

        if (invitation?.organizationId && result.cookies) {
          // Set the pending session cookies first
          await setCookies(result.cookies);

          // Find the invited organization in the list of available organizations
          const invitedOrg = result.organizations.find(
            (org) => org.id === invitation.organizationId,
          );

          if (invitedOrg) {
            // Automatically complete the organization selection
            const orgSelectionResult = await completeOrgSelection(invitation.organizationId);

            if (orgSelectionResult.success) {
              // Try to get organization name for better UX in success page
              let redirectUrl = "/apis";
              try {
                const org = await auth.getOrg(invitation.organizationId);
                if (org?.name) {
                  const params = new URLSearchParams({
                    from_invite: "true",
                    org_name: org.name,
                  });
                  redirectUrl = `/join/success?${params.toString()}`;
                }
              } catch (_error) {
                // Don't fail the redirect if we can't get org name
              }

              return {
                success: true,
                redirectTo: redirectUrl,
                cookies: [],
              };
            }
          }
        }
      } catch (_error) {
        // Fall through to return the original result if auto-selection fails
      }
    }

    if (result.cookies) {
      await setCookies(result.cookies);
    }

    // If we have an invitation token and verification was successful,
    // handle invitation acceptance and redirect to join success page
    if (invitationToken && result.success) {
      try {
        // Get invitation details to show organization context
        const invitation = await auth.getInvitation(invitationToken);

        if (invitation?.organizationId) {
          // For new users, we need to explicitly accept the invitation
          // as it might not be automatically accepted during verification
          if (invitation.state === "pending") {
            try {
              await auth.acceptInvitation(invitation.id);
            } catch (_acceptError) {
              // Don't fail - invitation might already be accepted
            }
          }

          // Try to get organization name for better UX
          let redirectUrl = result.redirectTo;
          try {
            const org = await auth.getOrg(invitation.organizationId);
            if (org?.name) {
              const params = new URLSearchParams({
                from_invite: "true",
                org_name: org.name,
              });
              redirectUrl = `/join/success?${params.toString()}`;
            } else {
              const params = new URLSearchParams({ from_invite: "true" });
              redirectUrl = `/join/success?${params.toString()}`;
            }
          } catch (_error) {
            // Don't fail if we can't get org name, just use join success without org name
            const params = new URLSearchParams({ from_invite: "true" });
            redirectUrl = `/join/success?${params.toString()}`;
          }

          return {
            success: true,
            redirectTo: redirectUrl,
            cookies: [],
          };
        }
      } catch (_error) {
        // Fall through to return original result
      }
    }

    return result;
  } catch (_error) {
    return {
      success: false,
      code: AuthErrorCode.UNKNOWN_ERROR,
      message: errorMessages[AuthErrorCode.UNKNOWN_ERROR],
    };
  }
}

export async function verifyEmail(code: string): Promise<VerificationResult> {
  try {
    // get the pending auth token
    // it's only good for 10 minutes
    const token = await getCookie(PENDING_SESSION_COOKIE);

    if (!token) {
      return {
        success: false,
        code: AuthErrorCode.UNKNOWN_ERROR,
        message: errorMessages[AuthErrorCode.UNKNOWN_ERROR],
      };
    }

    const result = await auth.verifyEmail({ code, token });

    if (result.cookies) {
      await setCookies(result.cookies);
    }

    return result;
  } catch (_error) {
    return {
      success: false,
      code: AuthErrorCode.UNKNOWN_ERROR,
      message: errorMessages[AuthErrorCode.UNKNOWN_ERROR],
    };
  }
}

export async function resendAuthCode(email: string): Promise<EmailAuthResult> {
  const envVars = env();
  const unkeyRootKey = envVars.UNKEY_ROOT_KEY;
  if (!unkeyRootKey) {
    return {
      success: false,
      code: AuthErrorCode.UNKNOWN_ERROR,
      message: "Service temporarily unavailable. Please try again later.",
    };
  }

  const rl = new Ratelimit({
    namespace: "resend_code",
    duration: "5m",
    limit: 5,
    rootKey: unkeyRootKey,
    onError: (_err: Error) => {
      return { success: true, limit: 0, remaining: 1, reset: 1 };
    },
  });

  const { success } = await rl.limit(email);

  if (!success) {
    return {
      success: false,
      code: AuthErrorCode.RATE_ERROR,
      message: "Sorry we can't send another code. Please contact support",
    };
  }

  if (!email.trim()) {
    return {
      success: false,
      code: AuthErrorCode.INVALID_EMAIL,
      message: "Email address is required.",
    };
  }
  return await auth.resendAuthCode(email);
}

export async function signIntoWorkspace(orgId: string): Promise<VerificationResult> {
  const pendingToken = (await cookies()).get(PENDING_SESSION_COOKIE)?.value;

  if (!pendingToken) {
    return {
      success: false,
      code: AuthErrorCode.UNKNOWN_ERROR,
      message: "No pending authentication found",
    };
  }

  try {
    const result = await auth.completeOrgSelection({
      orgId,
      pendingAuthToken: pendingToken,
    });

    if (result.success) {
      await setCookies(result.cookies);
      await deleteCookie(PENDING_SESSION_COOKIE);
      redirect(result.redirectTo);
    }

    return result;
  } catch (error) {
    return {
      success: false,
      code: AuthErrorCode.UNKNOWN_ERROR,
      message: error instanceof Error ? error.message : "Unknown error occurred",
    };
  }
}

// OAuth
export async function signInViaOAuth(options: SignInViaOAuthOptions): Promise<string> {
  return await auth.signInViaOAuth(options);
}

export async function completeOAuthSignIn(request: Request): Promise<OAuthResult> {
  try {
    const result = await auth.completeOAuthSignIn(request);

    if (result.success) {
      await setCookies(result.cookies);
      redirect(result.redirectTo);
    }

    return result;
  } catch (error) {
    return {
      success: false,
      code: AuthErrorCode.UNKNOWN_ERROR,
      message: error instanceof Error ? error.message : "Unknown error occurred",
    };
  }
}

// Organization Selection
export async function completeOrgSelection(
  orgId: string,
): Promise<NavigationResponse | AuthErrorResponse> {
  const tempSession = (await cookies()).get(PENDING_SESSION_COOKIE);
  if (!tempSession) {
    return {
      success: false,
      code: AuthErrorCode.PENDING_SESSION_EXPIRED,
      message: "Your session has expired. Please sign in again.",
    };
  }

  // Call auth provider with token and orgId
  const result = await auth.completeOrgSelection({
    pendingAuthToken: tempSession.value,
    orgId,
  });

  if (result.success) {
    (await cookies()).delete(PENDING_SESSION_COOKIE);
    for (const cookie of result.cookies) {
      (await cookies()).set(cookie.name, cookie.value, cookie.options);
    }
    // Store the last used organization ID in a cookie for auto-selection on next login
    try {
      await setLastUsedOrgCookie({ orgId });
    } catch (_error) {
      // Ignore cookie setting errors
    }
  } else {
    // Clear pending session on error to prevent stale token issues
    (await cookies()).delete(PENDING_SESSION_COOKIE);
  }

  return result;
}

// Server-accessible switch org function vs client-side trpc
// Used in route handlers, like join
export async function switchOrg(orgId: string): Promise<{ success: boolean; error?: string }> {
  try {
    const { newToken, expiresAt } = await auth.switchOrg(orgId);
    if (!newToken || !expiresAt) {
      throw new Error("Invalid session data returned from auth provider");
    }
    await setSessionCookie({ token: newToken, expiresAt });

    // Store the last used organization ID in a cookie for auto-selection on next login
    try {
      await setLastUsedOrgCookie({ orgId });
    } catch (_error) {
      // Ignore cookie setting errors
    }

    return { success: true };
  } catch (error) {
    return {
      success: false,
      error: error instanceof Error ? error.message : "Failed to switch organization",
    };
  }
}

/**
 * Accept an invitation and switch to the organization in one secure server action
 * This replaces the inline HTML approach with proper server-side cookie handling
 */
export async function acceptInvitationAndJoin(
  invitationId: string,
  organizationId: string,
): Promise<{ success: boolean; error?: string }> {
  try {
    // Accept invitation first
    await auth.acceptInvitation(invitationId);

    // Switch organization and get the new session token
    const { newToken, expiresAt } = await auth.switchOrg(organizationId);

    if (!newToken || !expiresAt) {
      throw new Error("Invalid session data returned from auth provider");
    }

    // Set the session cookie securely on the server side
    await setSessionCookie({ token: newToken, expiresAt });
    try {
      await setLastUsedOrgCookie({ orgId: organizationId });
    } catch (_error) {
      // Ignore cookie setting errors
    }

    return { success: true };
  } catch (error) {
    return {
      success: false,
      error: error instanceof Error ? error.message : "Failed to join organization",
    };
  }
}

/**
 * Verify Turnstile token and retry original auth operation
 */
export async function verifyTurnstileAndRetry(params: {
  turnstileToken: string;
  email: string;
  challengeParams: PendingTurnstileResponse["challengeParams"];
  userData?: { firstName: string; lastName: string }; // Only for sign-up
}): Promise<EmailAuthResult> {
  const { turnstileToken, email, challengeParams, userData } = params;

  // Verify Turnstile token
  const isValidToken = await verifyTurnstileToken(turnstileToken);

  if (!isValidToken) {
    return {
      success: false,
      code: AuthErrorCode.UNKNOWN_ERROR,
      message: "Verification failed. Please try again.",
    };
  }

  // Retry original auth operation based on action
  const metadata = getRequestMetadata();

  if (challengeParams.action === "sign-up" && userData) {
    // For sign-up, we need the user data
    return await auth.signUpViaEmail({
      ...userData,
      email,
      ...metadata,
      bypassRadar: true,
    });
  }
  if (challengeParams.action === "sign-in") {
    // For sign-in, we just need the email
    return await auth.signInViaEmail({
      email,
      ...metadata,
      bypassRadar: true,
    });
  }

  return {
    success: false,
    code: AuthErrorCode.UNKNOWN_ERROR,
    message: "Invalid challenge parameters.",
  };
}

/**
 * Check if a pending session exists (for workspace selection flow)
 * This is needed because PENDING_SESSION_COOKIE is HttpOnly and not accessible from client
 */
export async function hasPendingSession(): Promise<boolean> {
  const pendingToken = (await cookies()).get(PENDING_SESSION_COOKIE);
  return !!pendingToken?.value;
}
