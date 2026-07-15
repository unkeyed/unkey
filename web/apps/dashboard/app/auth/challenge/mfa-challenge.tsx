"use client";

import { Button } from "@unkey/ui";
import { useSearchParams } from "next/navigation";
import { useState } from "react";
import { completeAuthMfaChallenge } from "../actions";
import { ErrorBanner } from "../banners";
import { CodeInput } from "./code-input";
import { applyVerificationResult } from "./handle-result";

export function MfaChallenge() {
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
      const result = await completeAuthMfaChallenge({ code });
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
      <h1 className="text-2xl font-semibold tracking-tight text-gray-12">
        Two-factor authentication
      </h1>
      <p className="mt-4 text-sm text-gray-11">
        Enter the 6 digit code from your authenticator app to continue.
      </p>

      {error && <ErrorBanner>{error}</ErrorBanner>}

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
    </div>
  );
}
