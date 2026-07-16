"use client";

import { Button } from "@unkey/ui";
import { useRouter, useSearchParams } from "next/navigation";
import { Suspense, useEffect, useState } from "react";

const REDIRECT_DELAY_SECONDS = 3;

function JoinSuccessContent() {
  const router = useRouter();
  const searchParams = useSearchParams();
  const [countdown, setCountdown] = useState(REDIRECT_DELAY_SECONDS);

  // Get organization info from URL params if available
  const orgName = searchParams?.get("org_name");
  const fromInvite = searchParams?.get("from_invite") === "true";
  // The invitation was accepted but the org could not be made active for this
  // session, so the dashboard would still show the previous workspace. The
  // user has to pick the new one from the switcher, which means this page must
  // not bounce them onward before they have read that.
  const switchRequired = searchParams?.get("switch_required") === "true";

  useEffect(() => {
    if (switchRequired) {
      return;
    }

    const timer = setInterval(() => {
      setCountdown((prev) => {
        if (prev <= 1) {
          router.push("/apis");
          return 0;
        }
        return prev - 1;
      });
    }, 1000);

    return () => clearInterval(timer);
  }, [router, switchRequired]);

  const message = switchRequired
    ? `You've been added to ${
        orgName ?? "the workspace"
      }, but we couldn't open it for you. Select it from the workspace switcher to continue.`
    : fromInvite && orgName
      ? `You've been added to ${orgName}. Redirecting to your dashboard...`
      : fromInvite
        ? "You've successfully joined the workspace. Redirecting to your dashboard..."
        : "You'll be redirected to your dashboard in a moment...";

  return (
    <div className="min-h-screen bg-black text-white flex items-center justify-center">
      <div className="text-center max-w-md mx-auto px-4">
        {!switchRequired && (
          <div className="mb-6">
            <div className="w-16 h-16 border-4 border-gray-800 border-t-white rounded-full animate-spin mx-auto" />
          </div>
        )}

        <div className="mb-6">
          <h1 className="text-2xl font-semibold mb-3">
            {fromInvite ? "Successfully joined!" : "Authentication complete"}
          </h1>

          <p className="text-gray-400 mb-4">{message}</p>
        </div>

        {!switchRequired && (
          <div className="text-sm text-gray-500 mb-4">Redirecting in {countdown} seconds</div>
        )}

        <Button onClick={() => router.push("/apis")} variant="primary">
          {switchRequired ? "Go to dashboard" : "Continue manually"}
        </Button>
      </div>
    </div>
  );
}

export default function JoinSuccessPage() {
  return (
    <Suspense
      fallback={
        <div className="min-h-screen bg-black text-white flex items-center justify-center">
          <div className="text-center max-w-md mx-auto px-4">
            <div className="mb-6">
              <div className="w-16 h-16 border-4 border-gray-800 border-t-white rounded-full animate-spin mx-auto" />
            </div>
            <h1 className="text-2xl font-semibold mb-3">Loading...</h1>
          </div>
        </div>
      }
    >
      <JoinSuccessContent />
    </Suspense>
  );
}
