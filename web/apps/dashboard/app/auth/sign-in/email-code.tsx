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
    if (!code) {
      return;
    }
    setIsLoading(true);
    try {
      await handleVerification(code, invitationToken);
      toast.success("Signed in", {
        description: "redirecting...",
      });
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
    <div className="flex flex-col max-w-sm mx-auto text-left">
      <h1 className="text-4xl text-transparent bg-clip-text bg-linear-to-r from-white to-white/30">
        Security code sent!
      </h1>
      <p className="mt-4 text-sm text-white/40">
        To continue, please enter the 6 digit verification code sent to the provided email.
      </p>

      {/* Only show resend option after countdown reaches zero */}
      {timeLeft === 0 && (
        <p className="mt-2 text-sm text-white/40">
          Didn't receive the code?{" "}
          <button type="button" className="text-white" onClick={resendCode}>
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
          value={otp}
          onChange={setOtp}
          onComplete={() => verifyCode(otp)}
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
          className="flex items-center justify-center h-10 gap-2 px-4 text-sm font-semibold text-black duration-200 bg-white border border-white rounded-lg hover:border-white/30 hover:bg-black hover:text-white"
          disabled={isLoading}
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
      "relative w-10 h-12 text-[2rem] border border-white/20 rounded-lg text-white font-light text-base",
      "flex items-center justify-center",
      "transition-all duration-300",
      "group-hover:border-white/50 group-focus-within:border-white/50",
      "outline-solid outline-0 outline-white",
      { "outline-1 ": props.isActive },
    )}
  >
    {props.char !== null && <div>{props.char}</div>}
  </div>
);
