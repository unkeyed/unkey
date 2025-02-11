"use client";

import * as React from "react";

import { Loading } from "@/components/dashboard/loading";
import { toast } from "@/components/ui/toaster";
import { useSignUp } from "@/lib/auth/hooks/useSignUp";
import { AuthErrorCode, errorMessages } from "@/lib/auth/types";
import { cn } from "@/lib/utils";
import { OTPInput, type SlotProps } from "input-otp";

export const EmailCode: React.FC = () => {
  const { handleVerification, handleResendCode } = useSignUp();
  const [isLoading, setIsLoading] = React.useState(false);
  const [_timeLeft, _setTimeLeft] = React.useState(0);

  const verifyCode = async (otp: string) => {
    if (typeof otp !== "string") {
      return null;
    }
    setIsLoading(true);
    await handleVerification(otp).catch((err) => {
      setIsLoading(false);
      const errorCode = err.message as AuthErrorCode;
      toast.error(errorMessages[errorCode] || errorMessages[AuthErrorCode.UNKNOWN_ERROR]);
    });
  };

  const resendCode = async () => {
    try {
      const p = handleResendCode();
      toast.promise(p, {
        loading: "Sending new code ...",
        success: "A new code has been sent to your email",
      });
      await p;
    } catch (error) {
      setIsLoading(false);
      console.error(error);
    }
  };

  const [otp, setOtp] = React.useState("");

  return (
    <div className="flex flex-col max-w-sm mx-auto text-left">
      <h1 className="text-4xl text-transparent bg-clip-text bg-gradient-to-r from-white to-white/30">
        Security code sent!
      </h1>
      <p className="mt-4 text-sm text-white/40">
        To continue, please enter the 6 digit verification code sent to the provided email.
      </p>

      <p className="mt-2 text-sm text-white/40">
        Didn't receive the code?{" "}
        <button type="button" className="text-white" onClick={resendCode}>
          Resend
        </button>
      </p>
      <form className="flex flex-col gap-12 mt-10" onSubmit={() => verifyCode(otp)}>
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
          {isLoading ? <Loading className="w-4 h-4 mr-2 animate-spin" /> : null}
          Continue
        </button>
      </form>
    </div>
  );
};

const Slot: React.FC<SlotProps> = (props) => (
  <div
    className={cn(
      "relative w-10 h-12 text-[2rem] border border-white/20 rounded-lg text-white font-light text-base",
      "flex items-center justify-center",
      "transition-all duration-300",
      "group-hover:border-white/50 group-focus-within:border-white/50",
      "outline outline-0 outline-white",
      { "outline-1 ": props.isActive },
    )}
  >
    {props.char !== null && <div>{props.char}</div>}
  </div>
);
