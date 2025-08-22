"use server";

import { deleteCookie, getCookie, setCookies, setSessionCookie } from "@/lib/auth/cookies";
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

    if (result.cookies) {
      await setCookies(result.cookies);
    }

    return result;
  } catch (error) {
    console.error(error);
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
    console.error(error);
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
    onError: (err: Error, identifier: string) => {
      console.error(`Error occurred while rate limiting ${identifier}: ${err.message}`);
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
  const pendingToken = cookies().get("sess-temp")?.value;

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
      await deleteCookie("sess-temp");
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
  }

  return result;
}

// Server-accessible switch org function vs client-side trpc
// Used in route handlers, like join
export async function switchOrg(orgId: string): Promise<{ success: boolean; error?: string }> {
  try {
    const { newToken, expiresAt } = await auth.switchOrg(orgId);

    await setSessionCookie({ token: newToken, expiresAt });

    return { success: true };
  } catch (error) {
    console.error("Organization switch failed:", error);
    return {
      success: false,
      error: error instanceof Error ? error.message : "Failed to switch organization",
    };
  }
}
