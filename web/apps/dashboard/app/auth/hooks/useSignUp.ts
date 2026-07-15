"use client";

import type { EmailAuthResult, UserData, VerificationResult } from "@/lib/auth/types";
import { resendAuthCode, signUpViaEmail, verifyAuthCode, verifyEmail } from "../actions";
import { useSignUpContext } from "../context/signup-context";
import { useRadarSignals } from "../radar/radar-signals";

export function useSignUp() {
  const { userData, updateUserData, clearUserData } = useSignUpContext();
  const { getToken } = useRadarSignals();

  const handleSignUpViaEmail = async ({
    firstName,
    lastName,
    email,
  }: UserData): Promise<EmailAuthResult> => {
    updateUserData({ email, firstName, lastName });

    const signalsId = await getToken();
    const result = await signUpViaEmail({ email, firstName, lastName }, signalsId);
    return result;
  };

  const handleCodeVerification = async (
    code: string,
    invitationToken?: string,
  ): Promise<VerificationResult> => {
    // Validate userData exists and has email
    if (!userData || !userData.email) {
      throw new Error("User email is required for code verification.");
    }

    const signalsId = await getToken();
    return await verifyAuthCode({
      email: userData.email,
      code,
      invitationToken,
      signalsId,
    });
  };

  const handleEmailVerification = async (code: string): Promise<VerificationResult> => {
    return await verifyEmail(code);
  };

  const handleResendCode = async (): Promise<EmailAuthResult> => {
    // Validate userData exists and has email
    if (!userData || !userData.email) {
      throw new Error("User email is required to resend authentication code.");
    }

    try {
      const signalsId = await getToken();
      return await resendAuthCode(userData.email, signalsId);
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
    handleCodeVerification,
    handleEmailVerification,
    handleResendCode,
    handleSignUpViaEmail,
  };
}
