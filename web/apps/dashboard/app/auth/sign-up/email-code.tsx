"use client";

import * as React from "react";

import { AuthErrorCode, errorMessages } from "@/lib/auth/types";
import { cn } from "@/lib/utils";
import { Button, toast } from "@unkey/ui";
import { OTPInput, type SlotProps } from "input-otp";
import { applyVerificationResult } from "../challenge/handle-result";
import { useSignUp } from "../hooks/useSignUp";

export function EmailCode({ invitationToken }: { invitationToken?: string }) {
  const { handleCodeVerification, handleResendCode } = useSignUp();
  const [isLoading, setIsLoading] = React.useState(false);
  const [timeLeft, setTimeLeft] = React.useState(10); // Start with 10 seconds
  const [clientReady, setClientReady] = React.useState(false);
  const [otp, setOtp] = React.useState("");
  const timerRef = React.useRef<NodeJS.Timeout | null>(null);

  // Function to start or restart the countdown timer
  const startCountdown = React.useCallback(() => {
    // Clear any existing timer first
    if (timerRef.current) {
      clearInterval(timerRef.current);
    }

    // Set initial time
    setTimeLeft(10);

    // Start a new timer
    timerRef.current = setInterval(() => {
      setTimeLeft((prevTime) => {
        if (prevTime <= 1) {
          if (timerRef.current) {
            clearInterval(timerRef.current);
          }
          return 0;
        }
        return prevTime - 1;
      });
    }, 1000);
  }, []);

  React.useEffect(() => {
    setClientReady(true);
    startCountdown();

    // Clean up timer when component unmounts
    return () => {
      if (timerRef.current) {
        clearInterval(timerRef.current);
      }
    };
  }, [startCountdown]);

  const verifyCode = async (otp: string) => {
    if (typeof otp !== "string") {
      return null;
    }
    setIsLoading(true);
    try {
      const result = await handleCodeVerification(otp, invitationToken);
      const message = applyVerificationResult(result);
      if (message) {
        setIsLoading(false);
        toast.error(message);
      }
    } catch (err) {
      setIsLoading(false);
      const errorCode = (err as Error).message as AuthErrorCode;
      toast.error(errorMessages[errorCode] || errorMessages[AuthErrorCode.UNKNOWN_ERROR]);
    }
  };

  const resendCode = async () => {
    try {
      // Reset the timer when resending code
      setTimeLeft(10);

      const p = handleResendCode();
      toast.promise(p, {
        loading: "Sending new code ...",
        success: "A new code has been sent to your email",
      });
      await p;
    } catch (_error) {
      setIsLoading(false);
    }
  };

  return (
    <div className="flex flex-col w-full text-left">
      <h1 className="text-2xl font-semibold tracking-tight text-gray-12">Security code sent!</h1>
      <p className="mt-4 text-sm text-gray-11">
        To continue, please enter the 6 digit verification code sent to the provided email.
      </p>

      {/* Only show resend option after countdown reaches zero */}
      {timeLeft === 0 && (
        <p className="mt-2 text-sm text-gray-11">
          Didn't receive the code?{" "}
          <button type="button" className="text-gray-12" onClick={resendCode}>
            Resend
          </button>
        </p>
      )}

      <form
        className="flex flex-col gap-12 mt-10"
        onSubmit={(e) => {
          e.preventDefault();
          if (isLoading) {
            return;
          }
          verifyCode(otp);
        }}
      >
        <OTPInput
          data-1p-ignore
          autoFocus
          value={otp}
          onChange={setOtp}
          onComplete={(value) => {
            if (isLoading) {
              return;
            }
            verifyCode(value);
          }}
          disabled={isLoading}
          maxLength={6}
          render={({ slots }) => (
            <div className="flex items-center justify-between">
              {slots.slice(0, 6).map((slot, idx) => (
                // biome-ignore lint/suspicious/noArrayIndexKey: I have nothing better
                <Slot key={idx} {...slot} />
              ))}
            </div>
          )}
        />

        <Button
          type="submit"
          variant="primary"
          size="xlg"
          className="w-full rounded-lg mt-8"
          disabled={isLoading || otp.length !== 6}
          loading={clientReady && isLoading}
        >
          Continue
        </Button>
      </form>
    </div>
  );
}

const Slot: React.FC<SlotProps> = (props) => (
  <div
    className={cn(
      "relative w-10 h-12 border rounded-lg font-light text-base border-gray-6 text-gray-12",
      "flex items-center justify-center",
      "transition-all duration-300",
      "group-hover:border-gray-8 group-focus-within:border-gray-8",
      "outline-solid outline-0 outline-gray-12",
      { "outline-1": props.isActive },
    )}
  >
    {props.char !== null && <div>{props.char}</div>}
  </div>
);
