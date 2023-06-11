"use client";

import * as React from "react";
import { useRouter } from "next/navigation";
import { useSignUp } from "@clerk/nextjs";

import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Loading } from "@/components/loading";

export function EmailCode() {
  const [isLoading, setIsLoading] = React.useState(false);
  const { signUp, isLoaded: signUpLoaded, setActive } = useSignUp();
  const router = useRouter();

  const verifyCode = async (e: React.FormEvent<HTMLFormElement>) => {
    e.preventDefault();
    const emailCode = new FormData(e.currentTarget).get("code");
    if (!signUpLoaded || typeof emailCode !== "string") {
      return null;
    }
    setIsLoading(true);
    const verify = await signUp.attemptEmailAddressVerification({
      code: emailCode,
    });

    if (verify.status === "complete" && verify.createdSessionId) {
      await setActive({ session: verify.createdSessionId });
      router.push("/1/overview");
    }
  };

  return (
    <form className="grid gap-2" onSubmit={verifyCode}>
      <div className="grid gap-1">
        <Input
          name="code"
          placeholder="1234567"
          type="text"
          autoCapitalize="none"
          autoComplete="none"
          autoCorrect="off"
          className="bg-background"
        />
      </div>
      <Button disabled={isLoading}>
        {isLoading && <Loading className="mr-2 h-4 w-4 animate-spin" />}
        Verify Code
      </Button>
    </form>
  );
}
