"use client";

import { Loading } from "@unkey/ui";
import { useSearchParams } from "next/navigation";
import { useState } from "react";
import { completeAuthRadarEmailChallenge } from "../actions";
import { ErrorBanner } from "../banners";
import { CodeInput } from "./code-input";
import { applyVerificationResult } from "./handle-result";

export function RadarEmailChallenge() {
  const searchParams = useSearchParams();
  const redirectParam = searchParams?.get("redirect");
  const [otp, setOtp] = useState("");
  const [isLoading, setIsLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const verifyCode = async (code: string) => {
    if (!code || isLoading) {
      return;
    }
    setIsLoading(true);
    setError(null);
    try {
      const result = await completeAuthRadarEmailChallenge({ code });
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
      <h1 className="text-4xl text-transparent bg-clip-text bg-linear-to-r from-gray-12 to-gray-12/40">
        Verify it's you
      </h1>
      <p className="mt-4 text-sm text-gray-11">
        We noticed something unusual about this sign-in. Please enter the 6 digit verification code
        we sent to your email to continue.
      </p>

      {error && (
        <div className="mt-4">
          <ErrorBanner>{error}</ErrorBanner>
        </div>
      )}

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
          className="flex items-center cursor-pointer disabled:cursor-not-allowed disabled:opacity-50 justify-center h-10 gap-2 px-4 text-sm font-semibold rounded-lg border duration-200 text-gray-1 bg-gray-12 border-gray-12 hover:bg-gray-12/90"
          disabled={isLoading || otp.length !== 6}
        >
          {isLoading ? <Loading className="w-4 h-4 mr-2 animate-spin" /> : null}
          Continue
        </button>
      </form>
    </div>
  );
}
