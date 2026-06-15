"use client";

import { CopyButton, Loading } from "@unkey/ui";
import { useSearchParams } from "next/navigation";
import { useEffect, useRef, useState } from "react";
import { beginAuthMfaEnrollment, completeAuthMfaChallenge } from "../actions";
import { ErrorBanner } from "../banners";
import { CodeInput } from "./code-input";
import { applyVerificationResult } from "./handle-result";

type Enrollment = {
  qrCode: string;
  secret: string;
  challengeId: string;
};

export function MfaEnroll() {
  const searchParams = useSearchParams();
  const redirectParam = searchParams?.get("redirect");
  const [enrollment, setEnrollment] = useState<Enrollment | null>(null);
  const [otp, setOtp] = useState("");
  const [isLoading, setIsLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const hasStarted = useRef(false);

  useEffect(() => {
    if (hasStarted.current) {
      return;
    }
    hasStarted.current = true;

    beginAuthMfaEnrollment()
      .then((result) => {
        if (result.success) {
          setEnrollment({
            qrCode: result.qrCode,
            secret: result.secret,
            challengeId: result.challengeId,
          });
        } else {
          setError(result.message);
        }
      })
      .catch(() => {
        setError("Failed to start two-factor enrollment. Please sign in again.");
      });
  }, []);

  const verifyCode = async (code: string) => {
    if (!code || !enrollment || isLoading) {
      return;
    }
    setIsLoading(true);
    setError(null);
    try {
      const result = await completeAuthMfaChallenge({
        code,
        challengeId: enrollment.challengeId,
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
        Set up two-factor authentication
      </h1>
      <p className="mt-4 text-sm text-white/40">
        Scan the QR code with your authenticator app, then enter the 6 digit code to finish signing
        in.
      </p>

      {error && (
        <div className="mt-4">
          <ErrorBanner>{error}</ErrorBanner>
        </div>
      )}

      {/* The QR box keeps a fixed footprint whether or not the code has
          loaded, so the form below doesn't jump when the QR arrives. */}
      <div className="flex justify-center mt-8">
        {enrollment ? (
          <img
            src={enrollment.qrCode}
            alt="QR code for authenticator app"
            className="size-44 rounded-lg bg-white p-2"
          />
        ) : (
          <div className="flex size-44 items-center justify-center rounded-lg border border-white/10 bg-white/5">
            <Loading type="spinner" className="text-gray-6" />
          </div>
        )}
      </div>

      <div className="mt-4 flex flex-col gap-2">
        <p className="text-xs text-white/40">Can't scan it? Enter this code manually:</p>
        {/* The code sits on its own line so it never wraps mid-string, with
            one-click copy. Fixed single-line height keeps the form from
            shifting when the code resolves. */}
        <div className="flex items-center justify-between gap-2 rounded-lg border border-white/10 bg-white/5 px-3 py-2">
          <code className="font-mono text-sm text-white/80 whitespace-nowrap overflow-x-auto">
            {enrollment?.secret ?? "Generating…"}
          </code>
          <CopyButton
            value={enrollment?.secret ?? ""}
            variant="ghost"
            disabled={!enrollment}
            toastMessage="Copied setup code"
            className="shrink-0 text-white/60 hover:text-white"
          />
        </div>
      </div>

      <form
        className="flex flex-col gap-12 mt-8"
        onSubmit={(e) => {
          e.preventDefault();
          verifyCode(otp);
        }}
      >
        <CodeInput
          value={otp}
          onChange={setOtp}
          onComplete={verifyCode}
          disabled={isLoading || !enrollment}
        />

        <button
          type="submit"
          className="flex items-center cursor-pointer disabled:cursor-not-allowed justify-center h-10 gap-2 px-4 text-sm font-semibold text-black duration-200 bg-white border border-white rounded-lg hover:border-white/30 hover:bg-black hover:text-white"
          disabled={isLoading || !enrollment || otp.length !== 6}
        >
          {isLoading ? <Loading className="w-4 h-4 mr-2 animate-spin" /> : null}
          Continue
        </button>
      </form>
    </div>
  );
}
