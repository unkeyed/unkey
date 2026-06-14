"use server";

import { getCookie, setCookies, setLastUsedOrgCookie, setSessionCookie } from "@/lib/auth/cookies";
import { auth } from "@/lib/auth/server";
import {
  AUTH_CHALLENGE_COOKIE,
  type AuthChallengeCookieData,
  type AuthChallengeType,
  AuthErrorCode,
  type AuthErrorResponse,
  type EmailAuthResult,
  type Invitation,
  type NavigationResponse,
  type OAuthResult,
  PENDING_SESSION_COOKIE,
  type PendingAuthChallengeResponse,
  RADAR_ATTEMPT_COOKIE,
  type SignInViaOAuthOptions,
  UNKEY_LAST_ORG_COOKIE,
  type UserData,
  type VerificationResult,
  errorMessages,
} from "@/lib/auth/types";
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

// Reads the in-flight challenge state set when an authentication attempt was
// interrupted by an MFA or Radar challenge.
async function getChallengeCookieData(): Promise<AuthChallengeCookieData | null> {
  const raw = (await cookies()).get(AUTH_CHALLENGE_COOKIE)?.value;
  if (!raw) {
    return null;
  }
  try {
    return JSON.parse(raw) as AuthChallengeCookieData;
  } catch {
    return null;
  }
}

function pendingSessionExpired(): AuthErrorResponse {
  return {
    success: false,
    code: AuthErrorCode.PENDING_SESSION_EXPIRED,
    message: errorMessages[AuthErrorCode.PENDING_SESSION_EXPIRED],
  };
}

// Authentication Actions
export async function signUpViaEmail(params: UserData): Promise<EmailAuthResult> {
  const metadata = await getRequestMetadata();
  const result = await auth.signUpViaEmail({ ...params, ...metadata });
  if (result.success && result.cookies) {
    await setCookies(result.cookies);
  }
  return result;
}

export async function signInViaEmail(email: string): Promise<EmailAuthResult> {
  const metadata = await getRequestMetadata();
  const result = await auth.signInViaEmail({ email, ...metadata });
  if (result.success && result.cookies) {
    await setCookies(result.cookies);
  }
  return result;
}

export async function verifyAuthCode(params: {
  email: string;
  code: string;
  invitationToken?: string;
}): Promise<VerificationResult> {
  const { email, code, invitationToken } = params;
  try {
    // Fetch the invitation once up front; it is reused by both the
    // org-selection and the post-verification branches below.
    let invitation: Invitation | null = null;
    if (invitationToken) {
      invitation = await auth.getInvitation(invitationToken).catch(() => null);
      if (invitation?.email !== email) {
        throw new Error("Invalid invitation");
      }
    }

    const metadata = await getRequestMetadata();
    const radarAuthAttemptId = (await cookies()).get(RADAR_ATTEMPT_COOKIE)?.value || undefined;

    const result = await auth.verifyAuthCode({
      email,
      code,
      invitationToken,
      ...metadata,
      radarAuthAttemptId,
    });

    // If we have an invitation token and got organization_selection_required,
    // automatically select the invited organization
    if (
      invitationToken &&
      !result.success &&
      result.code === AuthErrorCode.ORGANIZATION_SELECTION_REQUIRED &&
      "organizations" in result
    ) {
      try {
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

            // The invited org may enforce MFA. Hand the challenge back to the
            // client so the user can enroll/verify — completeOrgSelection has
            // already persisted the challenge cookies. Without this an invited
            // user joining an MFA-required org could never finish signing in.
            if (!orgSelectionResult.success && "challengeType" in orgSelectionResult) {
              return orgSelectionResult;
            }

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

    const metadata = await getRequestMetadata();
    const result = await auth.verifyEmail({ code, token, ...metadata });

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

  const metadata = await getRequestMetadata();
  const radarAuthAttemptId = (await cookies()).get(RADAR_ATTEMPT_COOKIE)?.value || undefined;
  const result = await auth.resendAuthCode({ email, ...metadata, radarAuthAttemptId });
  if (result.success && result.cookies) {
    await setCookies(result.cookies);
  }
  return result;
}

// OAuth
export async function signInViaOAuth(options: SignInViaOAuthOptions): Promise<string> {
  return await auth.signInViaOAuth(options);
}

export async function completeOAuthSignIn(request: Request): Promise<OAuthResult> {
  // `redirect()` works by throwing a NEXT_REDIRECT error that the framework
  // catches at the boundary. Keep it OUTSIDE the try/catch — otherwise the
  // catch swallows the redirect and surfaces the framework's internal
  // "NEXT_REDIRECT;..." digest as an error message.
  let redirectTo: string | null = null;
  try {
    const result = await auth.completeOAuthSignIn(request);

    if (result.success) {
      await setCookies(result.cookies);
      redirectTo = result.redirectTo;
    } else {
      return result;
    }
  } catch (error) {
    return {
      success: false,
      code: AuthErrorCode.UNKNOWN_ERROR,
      message: error instanceof Error ? error.message : "Unknown error occurred",
    };
  }
  redirect(redirectTo);
}

// Organization Selection
export type OrgSelectionSuccess = NavigationResponse & { workspaceSlug?: string };

export async function completeOrgSelection(
  orgId: string,
): Promise<OrgSelectionSuccess | PendingAuthChallengeResponse | AuthErrorResponse> {
  const tempSession = (await cookies()).get(PENDING_SESSION_COOKIE);
  if (!tempSession) {
    return {
      success: false,
      code: AuthErrorCode.PENDING_SESSION_EXPIRED,
      message: errorMessages[AuthErrorCode.PENDING_SESSION_EXPIRED],
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

    // Look up the workspace slug for the selected org so the client can
    // rewrite deep-link URLs that contain a different workspace slug.
    let workspaceSlug: string | undefined;
    try {
      const { db } = await import("@/lib/db");
      const workspace = await db.query.workspaces.findFirst({
        where: (table, { and, eq, isNull }) =>
          and(eq(table.orgId, orgId), isNull(table.deletedAtM)),
        columns: { slug: true },
      });
      workspaceSlug = workspace?.slug ?? undefined;
    } catch {
      // Non-critical — fall back to default redirect
    }

    return { ...result, workspaceSlug };
  }

  // The selected organization may require MFA before it will issue a session
  // (e.g. "Require non-SSO members to be enrolled in MFA"). WorkOS surfaces
  // that as an mfa_challenge / mfa_enrollment during org selection. Persist
  // the challenge cookies — the fresh pending token and challenge state — so
  // the challenge/enrollment UI can finish the sign-in. Without this the new
  // pending token is dropped and the user can never sign in to that org.
  if ("challengeType" in result) {
    await setCookies(result.cookies);
  }

  // Don't clear pending session on error - let user try again or close modal
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
 * Returns the type of in-flight auth challenge (MFA or Radar), if any. The
 * challenge cookies are HttpOnly, so the client uses this to decide which
 * challenge UI to render.
 */
export async function getPendingAuthChallenge(): Promise<AuthChallengeType | null> {
  const pendingToken = (await cookies()).get(PENDING_SESSION_COOKIE)?.value;
  if (!pendingToken) {
    return null;
  }
  const challenge = await getChallengeCookieData();
  return challenge?.type ?? null;
}

/**
 * Creates a TOTP factor for the user who is mid sign-in and must enroll in
 * MFA. Returns the QR code and secret to render, plus the challenge ID the
 * client passes back to completeAuthMfaChallenge with the user's first code.
 */
export async function beginAuthMfaEnrollment(): Promise<
  | { success: true; qrCode: string; secret: string; uri: string; challengeId: string }
  | AuthErrorResponse
> {
  const challenge = await getChallengeCookieData();
  if (challenge?.type !== "mfa-enroll") {
    return pendingSessionExpired();
  }

  try {
    const enrollment = await auth.beginMfaEnrollment({
      userId: challenge.userId,
      email: challenge.email,
    });
    return {
      success: true,
      qrCode: enrollment.qrCode,
      secret: enrollment.secret,
      uri: enrollment.uri,
      challengeId: enrollment.challengeId,
    };
  } catch (_error) {
    return {
      success: false,
      code: AuthErrorCode.UNKNOWN_ERROR,
      message: errorMessages[AuthErrorCode.UNKNOWN_ERROR],
    };
  }
}

/**
 * Completes a pending MFA TOTP challenge. The challenge ID comes from the
 * challenge cookie (existing factor) or from beginAuthMfaEnrollment (new
 * factor).
 */
export async function completeAuthMfaChallenge(params: {
  code: string;
  challengeId?: string;
}): Promise<VerificationResult> {
  const pendingToken = (await cookies()).get(PENDING_SESSION_COOKIE)?.value;
  if (!pendingToken) {
    return pendingSessionExpired();
  }

  const challenge = await getChallengeCookieData();
  const challengeId =
    params.challengeId ?? (challenge?.type === "mfa" ? challenge.challengeId : undefined);
  if (!challengeId) {
    return pendingSessionExpired();
  }

  const metadata = await getRequestMetadata();
  const result = await auth.completeMfaChallenge({
    code: params.code,
    challengeId,
    pendingAuthToken: pendingToken,
    ...metadata,
  });

  if (result.cookies) {
    await setCookies(result.cookies);
  }
  if (result.success) {
    (await cookies()).delete(PENDING_SESSION_COOKIE);
  }
  return result;
}

/**
 * Completes a pending Radar email challenge with the code WorkOS emailed to
 * the user.
 */
export async function completeAuthRadarEmailChallenge(params: {
  code: string;
}): Promise<VerificationResult> {
  const pendingToken = (await cookies()).get(PENDING_SESSION_COOKIE)?.value;
  const challenge = await getChallengeCookieData();
  if (!pendingToken || challenge?.type !== "radar-email") {
    return pendingSessionExpired();
  }

  const metadata = await getRequestMetadata();
  const result = await auth.completeRadarEmailChallenge({
    code: params.code,
    radarChallengeId: challenge.radarChallengeId,
    pendingAuthToken: pendingToken,
    ...metadata,
  });

  if (result.cookies) {
    await setCookies(result.cookies);
  }
  if (result.success) {
    (await cookies()).delete(PENDING_SESSION_COOKIE);
  }
  return result;
}

/**
 * Sends the SMS code for a pending Radar SMS challenge to the given phone
 * number.
 */
export async function sendAuthRadarSmsCode(params: {
  phoneNumber: string;
}): Promise<{ success: true; verificationId: string; phoneNumber: string } | AuthErrorResponse> {
  const pendingToken = (await cookies()).get(PENDING_SESSION_COOKIE)?.value;
  const challenge = await getChallengeCookieData();
  if (!pendingToken || challenge?.type !== "radar-sms") {
    return pendingSessionExpired();
  }

  try {
    const metadata = await getRequestMetadata();
    const response = await auth.sendRadarSmsCode({
      userId: challenge.userId,
      phoneNumber: params.phoneNumber,
      pendingAuthToken: pendingToken,
      ...metadata,
    });
    return { success: true, ...response };
  } catch (_error) {
    return {
      success: false,
      code: AuthErrorCode.UNKNOWN_ERROR,
      message: "Failed to send the SMS code. Please check the phone number and try again.",
    };
  }
}

/**
 * Completes a pending Radar SMS challenge with the code the user received.
 */
export async function completeAuthRadarSmsChallenge(params: {
  code: string;
  verificationId: string;
  phoneNumber: string;
}): Promise<VerificationResult> {
  const pendingToken = (await cookies()).get(PENDING_SESSION_COOKIE)?.value;
  const challenge = await getChallengeCookieData();
  if (!pendingToken || challenge?.type !== "radar-sms") {
    return pendingSessionExpired();
  }

  const metadata = await getRequestMetadata();
  const result = await auth.completeRadarSmsChallenge({
    code: params.code,
    verificationId: params.verificationId,
    phoneNumber: params.phoneNumber,
    pendingAuthToken: pendingToken,
    ...metadata,
  });

  if (result.cookies) {
    await setCookies(result.cookies);
  }
  if (result.success) {
    (await cookies()).delete(PENDING_SESSION_COOKIE);
  }
  return result;
}

/**
 * Clear pending authentication state when user cancels org selection
 */
export async function clearPendingAuth(): Promise<void> {
  (await cookies()).delete(PENDING_SESSION_COOKIE);
  (await cookies()).delete(UNKEY_LAST_ORG_COOKIE);
  (await cookies()).delete(AUTH_CHALLENGE_COOKIE);
}

/**
 * Check if a pending session exists (for workspace selection flow)
 * This is needed because PENDING_SESSION_COOKIE is HttpOnly and not accessible from client
 */
export async function hasPendingSession(): Promise<boolean> {
  const pendingToken = (await cookies()).get(PENDING_SESSION_COOKIE);
  return !!pendingToken?.value;
}
