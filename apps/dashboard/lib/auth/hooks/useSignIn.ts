import { useContext, useEffect, useState } from "react";
import { SignInContext } from "../context/signin-context";
import { AuthErrorCode, AuthErrorResponse, errorMessages, Organization, PENDING_SESSION_COOKIE, PendingOrgSelectionResponse, SIGN_IN_URL, VerificationResult } from "../types";
import { signInViaEmail, verifyAuthCode, resendAuthCode } from '../actions';
import { useSearchParams } from "next/navigation";
import { getCookie } from "../cookies";

function isAuthErrorResponse(
  result: VerificationResult
): result is AuthErrorResponse {
  return !result.success && 'message' in result;
}

function isPendingOrgSelection(
  result: VerificationResult
): result is PendingOrgSelectionResponse {
  return (
    !result.success &&
    result.code === AuthErrorCode.ORGANIZATION_SELECTION_REQUIRED &&
    'organizations' in result &&
    Array.isArray(result.organizations)
  );
}

export function useSignIn() {
  const context = useContext(SignInContext);
  if (!context) throw new Error("useSignIn must be used within SignInProvider");

  const searchParams = useSearchParams();
  const [orgs, setOrgs] = useState<Organization[]>([]);

  const [loading, setLoading] = useState(true);

  const {
    setError,
    setIsVerifying,
    setEmail,
    setAccountNotFound,
  } = context;

  useEffect(() => {
    // Try to get organizations from URL parameters
    const orgsParam = searchParams?.get('orgs');
    
    if (orgsParam) {
      try {
        // Parse the organizations from the URL
        const organizations = JSON.parse(decodeURIComponent(orgsParam));
        setOrgs(organizations);
      } catch (err) {
        setError('Failed to load organizations');
      }
    }
    setLoading(false);
  }, [searchParams]);

  const hasPendingAuth = async () => {
    const hasTempSession = await getCookie(PENDING_SESSION_COOKIE); 
    return orgs.length && hasTempSession;
  }

  const handleSignInViaEmail = async (email: string) => {
    try {
      setEmail(email);
      setError(null);
      await signInViaEmail(email);
      setIsVerifying(true);
    } catch (err: any) {
      if (err.code === AuthErrorCode.ACCOUNT_NOT_FOUND) {
        setAccountNotFound(true);
        setEmail(email);
      } else {
        setError(errorMessages[AuthErrorCode.UNKNOWN_ERROR]);
      }
    }
  };

  const handleVerification = async (code: string): Promise<void> => {
  try {
    const result = await verifyAuthCode({
      email: context.email,
      code
    });

    // Determine where to redirect based on the verification result
    const redirectUrl = (() => {
      if (isPendingOrgSelection(result)) {
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
      await resendAuthCode(context.email);
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
    orgs,
    loading,
    hasPendingAuth
  };
}