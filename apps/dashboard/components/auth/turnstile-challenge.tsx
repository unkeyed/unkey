"use client";

import { Turnstile } from "@marsidev/react-turnstile";
import { useState } from "react";

interface TurnstileChallengeProps {
  email: string;
  onSuccess: (token: string) => void;
  onError: (error?: Error | string) => void;
  isLoading?: boolean;
}

export function TurnstileChallenge({ email, onSuccess, onError }: TurnstileChallengeProps) {
  const [isWidgetLoading, setIsWidgetLoading] = useState(true);
  const siteKey = process.env.NEXT_PUBLIC_CLOUDFLARE_TURNSTILE_SITE_KEY;

  if (!siteKey) {
    if (onError) {
      onError(new Error("Turnstile not configured"));
    }
    return (
      <div className="text-center p-4">
        <p className="text-red-500 text-sm">Turnstile is not configured. Please contact support.</p>
      </div>
    );
  }

  return (
    <div className="space-y-4">
      <div className="text-center">
        <h3 className="text-lg font-semibold text-white">Security Verification</h3>
        <p className="text-sm text-gray-400 mt-2">
          Please complete the verification challenge to continue with{" "}
          <span className="font-medium">{email}</span>
        </p>
      </div>

      <div className="flex justify-center">
        <div className="relative w-[300px] h-[65px]">
          {isWidgetLoading && (
            <div className="absolute inset-0 flex items-center justify-center bg-gray-800/90 backdrop-blur-sm rounded border border-gray-600 z-10">
              <div className="flex items-center space-x-2">
                <div className="animate-spin h-4 w-4 border-2 border-gray-400 border-t-white rounded-full" />
                <span className="text-sm text-gray-300">Loading verification...</span>
              </div>
            </div>
          )}
          <div className="w-full h-full">
            <Turnstile
              siteKey={siteKey}
              onWidgetLoad={() => {
                setIsWidgetLoading(false);
              }}
              onSuccess={(token) => {
                onSuccess(token);
              }}
              onError={(_error) => {
                setIsWidgetLoading(false);
                onError(new Error("Turnstile verification failed"));
              }}
              options={{
                theme: "dark",
                size: "normal",
              }}
            />
          </div>
        </div>
      </div>
    </div>
  );
}
