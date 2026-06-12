"use client";

import { FormInput, Loading } from "@unkey/ui";
import { useSearchParams } from "next/navigation";
import { type FormEvent, useState } from "react";
import { completeAuthRadarSmsChallenge, sendAuthRadarSmsCode } from "../actions";
import { ErrorBanner } from "../banners";
import { CodeInput } from "./code-input";
import { applyVerificationResult } from "./handle-result";

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

  const sendCode = async (e: FormEvent<HTMLFormElement>) => {
    e.preventDefault();
    const phoneNumber = new FormData(e.currentTarget).get("phoneNumber");
    if (typeof phoneNumber !== "string" || !phoneNumber.trim() || isLoading) {
      return;
    }

    setIsLoading(true);
    setError(null);
    try {
      const result = await sendAuthRadarSmsCode({ phoneNumber: phoneNumber.trim() });
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
    <div className="flex flex-col max-w-sm mx-auto text-left">
      <h1 className="text-4xl text-transparent bg-clip-text bg-linear-to-r from-white to-white/30">
        Verify it's you
      </h1>
      <p className="mt-4 text-sm text-white/40">
        We noticed something unusual about this sign-in.{" "}
        {verification
          ? `Enter the 6 digit code we sent to ${verification.phoneNumber}.`
          : "Please verify your phone number to continue."}
      </p>

      {error && (
        <div className="mt-4">
          <ErrorBanner>{error}</ErrorBanner>
        </div>
      )}

      {verification ? (
        <form
          className="flex flex-col gap-12 mt-10"
          onSubmit={(e) => {
            e.preventDefault();
            verifyCode(otp);
          }}
        >
          <CodeInput value={otp} onChange={setOtp} onComplete={verifyCode} disabled={isLoading} />

          <button
            type="submit"
            className="flex items-center cursor-pointer disabled:cursor-not-allowed justify-center h-10 gap-2 px-4 text-sm font-semibold text-black duration-200 bg-white border border-white rounded-lg hover:border-white/30 hover:bg-black hover:text-white"
            disabled={isLoading}
          >
            {isLoading ? <Loading className="w-4 h-4 mr-2 animate-spin" /> : null}
            Continue
          </button>
        </form>
      ) : (
        <form className="flex flex-col gap-12 mt-10" onSubmit={sendCode}>
          <FormInput
            label="Phone number"
            name="phoneNumber"
            placeholder="+1 555 555 5555"
            type="tel"
            autoComplete="tel"
            className="h-10 dark bg-black! w-full [&_input]:text-white!"
          />

          <button
            type="submit"
            className="flex items-center cursor-pointer disabled:cursor-not-allowed justify-center h-10 gap-2 px-4 text-sm font-semibold text-black duration-200 bg-white border border-white rounded-lg hover:border-white/30 hover:bg-black hover:text-white"
            disabled={isLoading}
          >
            {isLoading ? <Loading className="w-4 h-4 mr-2 animate-spin" /> : null}
            Send code
          </button>
        </form>
      )}
    </div>
  );
}
