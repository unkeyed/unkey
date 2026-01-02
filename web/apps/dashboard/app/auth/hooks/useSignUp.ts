"use client";

import type {
  EmailAuthResult,
  PendingTurnstileResponse,
  UserData,
  VerificationResult,
} from "@/lib/auth/types";
import { AuthErrorCode } from "@/lib/auth/types";
import {
  resendAuthCode,
  signUpViaEmail,
  verifyAuthCode,
  verifyEmail,
  verifyTurnstileAndRetry,
} from "../actions";
import { useSignUpContext } from "../context/signup-context";

export function useSignUp() {
  const { userData, updateUserData, clearUserData } = useSignUpContext();

  const isPendingTurnstileChallenge = (
    result: EmailAuthResult,
  ): result is PendingTurnstileResponse => {
    return (
      !result.success &&
      result.code === AuthErrorCode.RADAR_CHALLENGE_REQUIRED &&
      "email" in result &&
      "challengeParams" in result
    );
  };

  const handleSignUpViaEmail = async ({
    firstName,
    lastName,
    email,
  }: UserData): Promise<EmailAuthResult> => {
    updateUserData({ email, firstName, lastName });

    const result = await signUpViaEmail({ email, firstName, lastName });
    return result;
  };

  const handleTurnstileVerification = async (
    turnstileToken: string,
    challengeData: PendingTurnstileResponse,
  ): Promise<EmailAuthResult> => {
    // Validate userData exists and has required properties
    if (!userData || !userData.firstName || !userData.lastName) {
      throw new Error("User data is incomplete. First name and last name are required.");
    }

    const result = await verifyTurnstileAndRetry({
      turnstileToken,
      email: challengeData.email,
      challengeParams: challengeData.challengeParams,
      userData: {
        firstName: userData.firstName,
        lastName: userData.lastName,
      },
    });
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

    return await verifyAuthCode({
      email: userData.email,
      code,
      invitationToken,
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
      return await resendAuthCode(userData.email);
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
    handleTurnstileVerification,
    isPendingTurnstileChallenge,
  };
}
