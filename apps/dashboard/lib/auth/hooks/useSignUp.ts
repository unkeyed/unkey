"use client";

import { resendAuthCode, signUpViaEmail, verifyAuthCode } from "../actions";
import { useSignUpContext } from "../context/signup-context";
import type { UserData } from "../types";

export function useSignUp() {
  const { userData, updateUserData, clearUserData } = useSignUpContext();

  const handleSignUpViaEmail = async ({ firstName, lastName, email }: UserData): Promise<void> => {
    updateUserData({ email, firstName, lastName });

    try {
      await signUpViaEmail({ email, firstName, lastName });
    } catch (error) {
      console.error("Sign up failed:", error);
      throw error;
    }
  };

  const handleVerification = async (code: string): Promise<void> => {
    try {
      await verifyAuthCode({
        email: userData.email,
        code,
      });
    } catch (error) {
      console.error("Verification error:", error);
    }
  };

  const handleResendCode = async (): Promise<void> => {
    try {
      await resendAuthCode(userData.email);
    } catch (error) {
      throw new Error(
        `Failed to resend authentication code to ${userData.email}: ${
          error instanceof Error ? error.message : "Unknown error occurred"
        }`,
      );
    }
  };

  return {
    userData,
    updateUserData,
    clearUserData,
    handleVerification,
    handleResendCode,
    handleSignUpViaEmail,
  };
}
