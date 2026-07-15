"use client";

import * as React from "react";

import { AuthErrorCode, errorMessages } from "@/lib/auth/types";
import { cn } from "@/lib/utils";
import { Loading, toast } from "@unkey/ui";
import { OTPInput, type SlotProps } from "input-otp";
import { applyVerificationResult } from "../challenge/handle-result";
import { useSignUp } from "../hooks/useSignUp";

export const EmailVerify: React.FC = () => {
  const { handleEmailVerification } = useSignUp();
  const [isLoading, setIsLoading] = React.useState(false);
  const [_timeLeft, _setTimeLeft] = React.useState(0);
  const [clientReady, setClientReady] = React.useState(false);
  const [otp, setOtp] = React.useState("");

  // Set clientReady to true after hydration is complete
  React.useEffect(() => {
    setClientReady(true);
  }, []);

  const verifyEmail = async (otp: string) => {
    if (typeof otp !== "string" || isLoading) {
      return null;
    }
    setIsLoading(true);
    try {
      const result = await handleEmailVerification(otp);
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

  return (
    <div className="flex flex-col w-full text-left">
      <h1 className="text-4xl text-transparent bg-clip-text bg-linear-to-r from-gray-12 to-gray-12/40">
        Security code sent!
      </h1>
      <p className="mt-4 text-sm text-gray-11">
        To continue, please enter the 6 digit verification code sent to the provided email.
      </p>

      <form className="flex flex-col gap-12 mt-10" onSubmit={() => verifyEmail(otp)}>
        <OTPInput
          data-1p-ignore
          autoFocus
          value={otp}
          onChange={setOtp}
          onComplete={(value) => verifyEmail(value)}
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
          className="flex items-center justify-center cursor-pointer disabled:cursor-not-allowed disabled:opacity-50 h-10 gap-2 px-4 text-sm font-semibold rounded-lg border duration-200 text-gray-1 bg-gray-12 border-gray-12 hover:bg-gray-12/90"
          disabled={isLoading || otp.length !== 6}
          onClick={() => verifyEmail(otp)}
        >
          {clientReady && isLoading ? <Loading className="w-4 h-4 mr-2 animate-spin" /> : null}
          Continue
        </button>
      </form>
    </div>
  );
};

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
