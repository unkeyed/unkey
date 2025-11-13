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
  type SignInViaOAuthOptions,
  type UserData,
  type VerificationResult,
  errorMessages,
} from "@/lib/auth/types";
import { requireEmailMatch } from "@/lib/auth/utils";
import { env } from "@/lib/env";
import { Ratelimit } from "@unkey/ratelimit";
import { cookies } from "next/headers";
import { redirect } from "next/navigation";
// Authentication Actions
export async function signUpViaEmail(params: UserData): Promise<EmailAuthResult> {
  return await auth.signUpViaEmail(params);
}

export async function signInViaEmail(email: string): Promise<EmailAuthResult> {
  return await auth.signInViaEmail(email);
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
              } catch (error) {
                // Don't fail the redirect if we can't get org name
                console.warn("Could not fetch organization name for success page:", error);
              }

              return {
                success: true,
                redirectTo: redirectUrl,
                cookies: [],
              };
            }
          }
        }
      } catch (error) {
        console.error("Failed to auto-select invited organization:", {
          error: error instanceof Error ? error.message : "Unknown error",
        });
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
            } catch (acceptError) {
              // Log but don't fail - invitation might already be accepted
              console.warn("Could not accept invitation (might already be accepted):", {
                invitationId: invitation.id,
                error: acceptError instanceof Error ? acceptError.message : "Unknown error",
              });
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
          } catch (error) {
            // Don't fail if we can't get org name, just use join success without org name
            console.warn("Could not fetch organization name for new user success page:", error);
            const params = new URLSearchParams({ from_invite: "true" });
            redirectUrl = `/join/success?${params.toString()}`;
          }

          return {
            success: true,
            redirectTo: redirectUrl,
            cookies: [],
          };
        }
        console.warn("Invalid invitation or missing organization ID:", {
          hasInvitation: !!invitation,
          organizationId: invitation?.organizationId,
        });
      } catch (error) {
        console.error("Failed to process invitation for new user:", {
          error: error instanceof Error ? error.message : "Unknown error",
          invitationToken: `${invitationToken.substring(0, 10)}...`,
        });
        // Fall through to return original result
      }
    }

    return result;
  } catch (error) {
    console.error("Failed to verify auth code:", {
      error: error instanceof Error ? error.message : "Unknown error",
    });
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
      console.error("Pending auth token missing or expired");
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
  } catch (error) {
    console.error("Failed to verify email:", {
      error: error instanceof Error ? error.message : "Unknown error",
    });
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
    console.error("UNKEY_ROOT_KEY environment variable is not set");
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
    onError: (err: Error) => {
      console.error("Rate limiting error:", err.message);
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
  const pendingToken = cookies().get(PENDING_SESSION_COOKIE)?.value;

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
  const tempSession = cookies().get(PENDING_SESSION_COOKIE);
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
    cookies().delete(PENDING_SESSION_COOKIE);
    for (const cookie of result.cookies) {
      cookies().set(cookie.name, cookie.value, cookie.options);
    }
    // Store the last used organization ID in a cookie for auto-selection on next login
    try {
      await setLastUsedOrgCookie({ orgId });
    } catch (error) {
      console.error("Failed to set last used org cookie in completeOrgSelection:", {
        orgId,
        error: error instanceof Error ? error.message : "Unknown error",
      });
    }
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
    } catch (error) {
      console.error("Failed to set last used org cookie in switchOrg:", {
        orgId,
        error: error instanceof Error ? error.message : "Unknown error",
      });
    }

    return { success: true };
  } catch (error) {
    console.error("Organization switch failed:", {
      error: error instanceof Error ? error.message : "Unknown error",
    });
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
    } catch (error) {
      console.error("Failed to set last used org cookie in acceptInvitationAndJoin:", {
        orgId: organizationId,
        error: error instanceof Error ? error.message : "Unknown error",
      });
    }

    return { success: true };
  } catch (error) {
    console.error("Failed to accept invitation and join organization:", {
      error: error instanceof Error ? error.message : "Unknown error",
    });
    return {
      success: false,
      error: error instanceof Error ? error.message : "Failed to join organization",
    };
  }
}

/**
 * Check if a pending session exists (for workspace selection flow)
 * This is needed because PENDING_SESSION_COOKIE is HttpOnly and not accessible from client
 */
export async function hasPendingSession(): Promise<boolean> {
  const pendingToken = cookies().get(PENDING_SESSION_COOKIE);
  return !!pendingToken?.value;
}
