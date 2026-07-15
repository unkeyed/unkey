"use client";

import * as React from "react";

import { AuthErrorCode, errorMessages } from "@/lib/auth/types";
import { cn } from "@/lib/utils";
import { Button, toast } from "@unkey/ui";
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
      <h1 className="text-2xl font-semibold tracking-tight text-gray-12">Security code sent!</h1>
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

        <Button
          type="submit"
          variant="primary"
          size="xlg"
          className="w-full rounded-lg"
          disabled={isLoading || otp.length !== 6}
          loading={clientReady && isLoading}
          onClick={() => verifyEmail(otp)}
        >
          Continue
        </Button>
      </form>
    </div>
  );
};

const Slot: React.FC<SlotProps> = (props) => (
  <div
    className={cn(
      "relative w-10 h-12 border rounded-lg font-light text-base border-gray-6 text-gray-12",
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
