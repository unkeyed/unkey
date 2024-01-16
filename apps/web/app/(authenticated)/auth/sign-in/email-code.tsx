"use client";

import { useSignIn } from "@clerk/nextjs";
import { useRouter } from "next/navigation";
import * as React from "react";

import { Loading } from "@/components/dashboard/loading";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { toast } from "@/components/ui/toaster";

export function EmailCode() {
  const router = useRouter();
  const { signIn, isLoaded: signInLoaded, setActive } = useSignIn();

  const [isLoading, setIsLoading] = React.useState(false);
  const [timeLeft, setTimeLeft] = React.useState(0);

  const verifyCode = async (e: React.FormEvent<HTMLFormElement>) => {
    e.preventDefault();
    const emailCode = new FormData(e.currentTarget).get("code");
    if (!signInLoaded || typeof emailCode !== "string") {
      return null;
    }
    setIsLoading(true);
    await signIn
      .attemptFirstFactor({
        strategy: "email_code",
        code: emailCode,
      })
      .then(async (result) => {
        if (result.status === "complete" && result.createdSessionId) {
          await setActive({ session: result.createdSessionId });
          router.push("/app");
        }
      })
      .catch((err) => {
        setIsLoading(false);
        if (err.errors[0].code === "form_code_incorrect") {
          toast.error("Please check the 6 digit code, the one you entered is incorrect");
        }
      });
  };

  const resendCode = async (e: React.FormEvent) => {
    e.preventDefault();
    try {
      const firstFactor = signIn?.supportedFirstFactors.find((f) => f.strategy === "email_code") as
        | { emailAddressId: string }
        | undefined;
      if (!firstFactor) {
        return null;
      }
      await signIn?.prepareFirstFactor({
        strategy: "email_code",
        emailAddressId: firstFactor.emailAddressId,
      });
      setTimeLeft(30);
      toast.success("A new code has been sent to your email");
      const _interval = setInterval(() => {
        setTimeLeft((time) => {
          if (time === 0) {
            clearInterval(_interval);
            return 0;
          }
          return time - 1;
        });
      }, 1000);
    } catch (error) {
      setIsLoading(false);
      console.error(error);
    }
  };

  return (
    <>
      <form className="grid gap-2" onSubmit={verifyCode}>
        <div className="grid gap-1">
          <Input
            name="code"
            placeholder="123456"
            type="number"
            autoCapitalize="none"
            autoComplete="none"
            maxLength={6}
            autoCorrect="off"
            className="bg-background"
          />
        </div>
        <Button disabled={isLoading}>
          {isLoading && <Loading className="w-4 h-4 mr-2 animate-spin" />}
          Verify Code
        </Button>
        <Button disabled={isLoading || timeLeft > 0} variant="ghost" onClick={resendCode}>
          Resend Code {timeLeft > 0 && `(${timeLeft})`}
        </Button>
      </form>
    </>
  );
}
