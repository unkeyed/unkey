"use client";

import { cn } from "@/lib/utils";
import { Loading, toast } from "@unkey/ui";
import { OTPInput, type SlotProps } from "input-otp";
import { useCallback, useEffect, useRef, useState } from "react";
import { useSignIn } from "../hooks";

export function EmailCode({ invitationToken }: { invitationToken?: string }) {
  const { handleVerification, handleResendCode, setError } = useSignIn();
  const [timeLeft, setTimeLeft] = useState(10); // Start with 10 seconds
  const [isLoading, setIsLoading] = useState(false);
  const [otp, setOtp] = useState("");
  const [clientReady, setClientReady] = useState(false);
  const timerRef = useRef<NodeJS.Timeout | null>(null);

  // Function to start or restart the countdown timer
  const startCountdown = useCallback(() => {
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

  // Set clientReady to true after hydration is complete
  useEffect(() => {
    setClientReady(true);
    startCountdown();

    // Start countdown timer only on client side
    const timer = setInterval(() => {
      setTimeLeft((prevTime) => {
        if (prevTime <= 1) {
          clearInterval(timer);
          return 0;
        }
        return prevTime - 1;
      });
    }, 1000);

    // Clean up timer when component unmounts
    return () => {
      if (timerRef.current) {
        clearInterval(timerRef.current);
      }
    };
  }, [startCountdown]);

  const verifyCode = async (code: string) => {
    if (!code || isLoading) {
      return;
    }
    setIsLoading(true);
    try {
      const isNavigating = await handleVerification(code, invitationToken);
      if (isNavigating) {
        toast.success("Signed in", {
          description: "redirecting...",
        });
        // Keep the button in its loading state until the browser leaves the
        // page, otherwise it pops back to "Continue" mid-navigation.
        return;
      }
    } catch (err) {
      setError((err as Error).message);
    }
    setIsLoading(false);
  };

  const resendCode = async () => {
    try {
      // Reset the timer when resending code
      startCountdown();

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
      <h1 className="text-4xl text-transparent bg-clip-text bg-linear-to-r from-gray-12 to-gray-12/40">
        Security code sent!
      </h1>
      <p className="mt-4 text-sm text-gray-11">
        To continue, please enter the 6 digit verification code sent to the provided email.
      </p>

      {/* Only show resend option after countdown reaches zero */}
      {timeLeft === 0 && (
        <p className="mt-2 text-sm text-gray-11">
          Didn't receive the code?{" "}
          <button type="button" className="cursor-pointer text-gray-12" onClick={resendCode}>
            Resend
          </button>
        </p>
      )}

      <form
        className="flex flex-col gap-12 mt-10"
        onSubmit={(e) => {
          e.preventDefault();
          verifyCode(otp);
        }}
      >
        <OTPInput
          data-1p-ignore
          autoFocus
          className="[&_input]:text-gray-12!"
          value={otp}
          onChange={setOtp}
          onComplete={(value) => verifyCode(value)}
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

        <button
          type="submit"
          className="flex items-center cursor-pointer disabled:cursor-not-allowed disabled:opacity-50 justify-center h-10 gap-2 px-4 text-sm font-semibold rounded-lg border duration-200 text-gray-1 bg-gray-12 border-gray-12 hover:bg-gray-12/90"
          disabled={isLoading || otp.length !== 6}
          onClick={() => verifyCode(otp)}
        >
          {clientReady && isLoading ? <Loading className="w-4 h-4 mr-2 animate-spin" /> : null}
          Continue
        </button>
      </form>
    </div>
  );
}

const Slot: React.FC<SlotProps> = (props) => (
  <div
    className={cn(
      "relative w-10 h-12 text-[2rem] border rounded-lg font-light text-base border-gray-6 text-gray-12",
      "flex items-center justify-center",
      "transition-all duration-300",
      "group-hover:border-gray-8 group-focus-within:border-gray-8",
      "outline-solid outline-0 outline-gray-12",
      { "outline-1 ": props.isActive },
    )}
  >
    {props.char !== null && <div>{props.char}</div>}
  </div>
);
