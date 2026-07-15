"use client";

import { Button } from "@unkey/ui";
import { useSearchParams } from "next/navigation";
import { type FormEvent, useState } from "react";
import { completeAuthRadarSmsChallenge, sendAuthRadarSmsCode } from "../actions";
import { ErrorBanner } from "../banners";
import { CodeInput } from "./code-input";
import { applyVerificationResult } from "./handle-result";
import { PhoneInput } from "./phone-input";

type SmsVerification = {
  verificationId: string;
  phoneNumber: string;
};

export function RadarSmsChallenge() {
  const searchParams = useSearchParams();
  const redirectParam = searchParams?.get("redirect");
  const [verification, setVerification] = useState<SmsVerification | null>(null);
  const [otp, setOtp] = useState("");
  const [isLoading, setIsLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);
  // E.164 phone number from the PhoneInput; empty until a valid number is entered.
  const [phoneNumber, setPhoneNumber] = useState("");

  const sendCode = async (e: FormEvent<HTMLFormElement>) => {
    e.preventDefault();
    if (!phoneNumber || isLoading) {
      return;
    }

    setIsLoading(true);
    setError(null);
    try {
      const result = await sendAuthRadarSmsCode({ phoneNumber });
      if (result.success) {
        setVerification({
          verificationId: result.verificationId,
          phoneNumber: result.phoneNumber,
        });
      } else {
        setError(result.message);
      }
    } catch (_error) {
      setError("Failed to send the SMS code. Please try again.");
    } finally {
      setIsLoading(false);
    }
  };

  const verifyCode = async (code: string) => {
    if (!code || !verification || isLoading) {
      return;
    }
    setIsLoading(true);
    setError(null);
    try {
      const result = await completeAuthRadarSmsChallenge({
        code,
        verificationId: verification.verificationId,
        phoneNumber: verification.phoneNumber,
      });
      const message = applyVerificationResult(result, redirectParam);
      if (message) {
        setError(message);
        setIsLoading(false);
      }
      // On success the browser is navigating away; keep the loading state up
      // so the button doesn't pop back to idle mid-transition.
    } catch (_error) {
      setError("Something went wrong. Please try again.");
      setIsLoading(false);
    }
  };

  return (
    <div className="flex flex-col w-full text-left">
      <h1 className="text-2xl font-semibold tracking-tight text-gray-12">Verify it's you</h1>
      <p className="mt-4 text-sm text-gray-11">
        We noticed something unusual about this sign-in.{" "}
        {verification
          ? `Enter the 6 digit code we sent to ${verification.phoneNumber}.`
          : "Please verify your phone number to continue."}
      </p>

      {error && <ErrorBanner>{error}</ErrorBanner>}

      {verification ? (
        <form
          className="flex flex-col gap-12 mt-10"
          onSubmit={(e) => {
            e.preventDefault();
            verifyCode(otp);
          }}
        >
          <CodeInput value={otp} onChange={setOtp} onComplete={verifyCode} disabled={isLoading} />

          <Button
            type="submit"
            variant="primary"
            size="xlg"
            className="w-full rounded-lg"
            disabled={isLoading || otp.length !== 6}
            loading={isLoading}
          >
            Continue
          </Button>
        </form>
      ) : (
        <form className="flex flex-col gap-12 mt-10" onSubmit={sendCode}>
          <PhoneInput onChange={(e164) => setPhoneNumber(e164)} disabled={isLoading} />

          <Button
            type="submit"
            variant="primary"
            size="xlg"
            className="w-full rounded-lg"
            disabled={isLoading || !phoneNumber}
            loading={isLoading}
          >
            Send code
          </Button>
        </form>
      )}
    </div>
  );
}
