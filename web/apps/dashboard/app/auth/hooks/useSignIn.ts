import { getCookie } from "@/lib/auth/cookies";
import {
  AuthErrorCode,
  type AuthErrorResponse,
  type EmailAuthResult,
  type Organization,
  PENDING_SESSION_COOKIE,
  type PendingOrgSelectionResponse,
  type PendingTurnstileResponse,
  SIGN_IN_URL,
  type VerificationResult,
  errorMessages,
} from "@/lib/auth/types";
import { useSearchParams } from "next/navigation";
import { useContext, useEffect, useState } from "react";
import {
  resendAuthCode,
  signInViaEmail,
  verifyAuthCode,
  verifyTurnstileAndRetry,
} from "../actions";
import { SignInContext } from "../context/signin-context";

function isAuthErrorResponse(result: VerificationResult): result is AuthErrorResponse {
  return !result.success && "message" in result;
}

function isPendingOrgSelection(result: VerificationResult): result is PendingOrgSelectionResponse {
  return (
    !result.success &&
    result.code === AuthErrorCode.ORGANIZATION_SELECTION_REQUIRED &&
    "organizations" in result &&
    Array.isArray(result.organizations)
  );
}

function isPendingTurnstileChallenge(result: EmailAuthResult): result is PendingTurnstileResponse {
  return (
    !result.success &&
    result.code === AuthErrorCode.RADAR_CHALLENGE_REQUIRED &&
    "email" in result &&
    "challengeParams" in result
  );
}

export function useSignIn() {
  const context = useContext(SignInContext);
  if (!context) {
    throw new Error("useSignIn must be used within SignInProvider");
  }

  const searchParams = useSearchParams();
  const [orgs, setOrgs] = useState<Organization[]>([]);
  const [hasPendingAuth, setHasPendingAuth] = useState<boolean>(false);
  const [loading, setLoading] = useState(true);

  const { setError, setIsVerifying, setEmail, setAccountNotFound } = context;

  useEffect(() => {
    const checkAuthStatus = async () => {
      try {
        // Try to get organizations from URL parameters
        const orgsParam = searchParams?.get("orgs");
        let parsedOrgs: Organization[] = [];

        if (orgsParam) {
          try {
            parsedOrgs = JSON.parse(decodeURIComponent(orgsParam));
            if (Array.isArray(parsedOrgs)) {
              setOrgs(parsedOrgs);
            } else {
              // Invalid format, clear orgs
              setOrgs([]);
            }
          } catch (_err) {
            // Invalid JSON, clear orgs and don't show error
            setOrgs([]);
          }
        } else {
          // No orgs param, clear orgs
          setOrgs([]);
        }

        // Check for pending session cookie
        const hasTempSession = await getCookie(PENDING_SESSION_COOKIE);
        setHasPendingAuth(Boolean(parsedOrgs.length && hasTempSession));
      } catch (_err) {
        // Ignore auth status check errors
      } finally {
        setLoading(false);
      }
    };

    checkAuthStatus();
  }, [searchParams]);

  const handleSignInViaEmail = async (email: string) => {
    try {
      setEmail(email);
      setError(null);
      const result = await signInViaEmail(email);

      // Return result for Turnstile challenge handling
      if (isPendingTurnstileChallenge(result)) {
        return result;
      }

      // Check if the operation was successful
      if (result.success) {
        setIsVerifying(true);
        return result;
      }

      // Handle error case - only set error message if we have an error response
      if (isAuthErrorResponse(result)) {
        if (result.code === AuthErrorCode.ACCOUNT_NOT_FOUND) {
          setAccountNotFound(true);
          setEmail(email);
        } else {
          setError(result.message);
        }
      } else {
        setError(errorMessages[AuthErrorCode.UNKNOWN_ERROR]);
      }

      return result;
    } catch (error) {
      // This catches any unexpected errors that weren't handled by the API
      setError(errorMessages[AuthErrorCode.UNKNOWN_ERROR]);
      throw error;
    }
  };

  const handleVerification = async (code: string, invitationToken?: string): Promise<void> => {
    try {
      const result = await verifyAuthCode({
        email: context.email,
        code,
        invitationToken,
      });

      // Determine where to redirect based on the verification result
      const redirectUrl = (() => {
        // If we have an invitation token, the verifyAuthCode should have handled
        // the org selection automatically, so we should get a success response
        if (invitationToken && result.success) {
          return result.redirectTo;
        }

        // Only show org selector if we don't have an invitation token
        if (!invitationToken && isPendingOrgSelection(result)) {
          const orgsParam = encodeURIComponent(JSON.stringify(result.organizations));
          return `${SIGN_IN_URL}?orgs=${orgsParam}`;
        }

        if (result.success) {
          return result.redirectTo;
        }

        return null;
      })();

      // If we have a redirect URL, navigate to it
      if (redirectUrl) {
        window.location.href = redirectUrl;
        return;
      }

      // Handle error case - only set error message if we have an error response
      if (isAuthErrorResponse(result)) {
        setError(result.message);
      } else {
        setError(errorMessages[AuthErrorCode.UNKNOWN_ERROR]);
      }
    } catch (error) {
      setError(errorMessages[AuthErrorCode.UNKNOWN_ERROR]);
      throw error;
    }
  };

  const handleResendCode = async (): Promise<void> => {
    try {
      const result = await resendAuthCode(context.email);
      if (!result.success) {
        setError(errorMessages[AuthErrorCode.UNKNOWN_ERROR]);
        return;
      }
    } catch (error) {
      setError(errorMessages[AuthErrorCode.UNKNOWN_ERROR]);
      throw error;
    }
  };

  const handleTurnstileVerification = async (
    turnstileToken: string,
    challengeData: PendingTurnstileResponse,
    userData?: { firstName: string; lastName: string },
  ): Promise<void> => {
    try {
      setError(null);
      const result = await verifyTurnstileAndRetry({
        turnstileToken,
        email: challengeData.email,
        challengeParams: challengeData.challengeParams,
        userData,
      });

      if (result.success) {
        setIsVerifying(true);
        return;
      }

      if (isAuthErrorResponse(result)) {
        if (result.code === AuthErrorCode.ACCOUNT_NOT_FOUND) {
          setAccountNotFound(true);
          setEmail(challengeData.email);
        } else {
          setError(result.message);
        }
      } else {
        setError(errorMessages[AuthErrorCode.UNKNOWN_ERROR]);
      }
    } catch (error) {
      setError(errorMessages[AuthErrorCode.UNKNOWN_ERROR]);
      throw error;
    }
  };

  return {
    ...context,
    handleSignInViaEmail,
    handleVerification,
    handleResendCode,
    handleTurnstileVerification,
    isPendingTurnstileChallenge,
    orgs,
    loading,
    hasPendingAuth,
  };
}
