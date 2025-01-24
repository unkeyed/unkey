import { useContext } from "react";
import { SignInContext } from "../context/signin-context";
import { AuthErrorCode, errorMessages } from "../types";
import { signInViaEmail, verifyAuthCode, resendAuthCode } from '../actions';

export function useSignIn() {
  const context = useContext(SignInContext);
  if (!context) throw new Error("useSignIn must be used within SignInProvider");

  const {
    setError,
    setIsVerifying,
    setEmail,
    setAccountNotFound,
  } = context;

  const handleSignInViaEmail = async (email: string) => {
    try {
      setEmail(email);
      setError(null);
      await signInViaEmail(email);
      setIsVerifying(true);
    } catch (err: any) {
      if (err.errors?.[0]?.code === AuthErrorCode.ACCOUNT_NOT_FOUND) {
        setAccountNotFound(true);
        setEmail(email);
      } else {
        setError(errorMessages[AuthErrorCode.UNKNOWN_ERROR]);
      }
    }
  };

  const handleVerification = async (code: string): Promise<void> => {
    try {
      await verifyAuthCode({
        email: context.email,
        code
      });
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
    handleResendCode
  };
}