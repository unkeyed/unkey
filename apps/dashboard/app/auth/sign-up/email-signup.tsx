"use client";

import * as React from "react";

import { Loading } from "@/components/dashboard/loading";
import { FadeInStagger } from "@/components/landing/fade-in";
import { toast } from "@/components/ui/toaster";
import { AuthErrorCode, errorMessages } from "@/lib/auth/types";
import { useSearchParams } from "next/navigation";
import { useSignUp } from "../hooks/useSignUp";
import { FormInput } from "@unkey/ui";

interface Props {
  setVerification: (value: boolean) => void;
}

export const EmailSignUp: React.FC<Props> = ({ setVerification }) => {
  const { handleSignUpViaEmail } = useSignUp();
  const [isLoading, setIsLoading] = React.useState(false);
  const searchParams = useSearchParams();
  const emailFromParams = searchParams?.get("email") || "";

  //fix hydration error with the loading state
  const [clientLoaded, setClientLoaded] = React.useState(false);

  React.useEffect(() => {
    setClientLoaded(true);
  }, []);

  const signUpWithCode = async (e: React.FormEvent<HTMLFormElement>) => {
    e.preventDefault();
    const email = new FormData(e.currentTarget).get("email");
    const first = new FormData(e.currentTarget).get("first");
    const last = new FormData(e.currentTarget).get("last");

    if (
      typeof email !== "string" ||
      typeof first !== "string" ||
      typeof last !== "string"
    ) {
      return null;
    }

    try {
      setIsLoading(true);
      await handleSignUpViaEmail({
        email: email,
        firstName: first,
        lastName: last,
      }).then(() => {
        setVerification(true);
      });
    } catch (err: any) {
      const errorCode = err.message as AuthErrorCode;
      toast.error(
        errorMessages[errorCode] || errorMessages[AuthErrorCode.UNKNOWN_ERROR]
      );
      console.error(err);
    } finally {
      setIsLoading(false);
    }
  };

  return (
      <form className="grid gap-16" onSubmit={signUpWithCode}>
        <div className="grid gap-10">
          <div className="flex flex-row gap-3">
            <div className="flex flex-col items-start w-1/2 gap-2">
              <FormInput
                label="First Name"
                name="first"
                placeholder="Bruce"
                type="text"
                autoCapitalize="none"
                autoCorrect="off"
                className="h-10 dark !bg-black"
              />
            </div>
            <div className="flex flex-col items-start w-1/2 gap-2">
              <FormInput
                label="Last Name"
                name="last"
                placeholder="Banner"
                type="text"
                autoCapitalize="none"
                autoCorrect="off"
                className="h-10 dark !bg-black"
              />
            </div>
          </div>
          <div className="flex flex-col items-start gap-2">
            <FormInput
              label="Email"
              name="email"
              defaultValue={emailFromParams}
              placeholder="name@example.com"
              type="email"
              autoCapitalize="none"
              autoComplete="email"
              autoCorrect="off"
              className="h-10 dark !bg-black w-full"
            />
          </div>
        </div>
        <button
          type="submit"
          className="flex items-center justify-center h-10 gap-2 px-4 text-sm font-semibold text-black duration-200 bg-white border border-white rounded-lg hover:border-white/30 hover:bg-black hover:text-white"
          disabled={isLoading}
        >
          {!clientLoaded ? (
            "Sign Up with Email"
          ) : isLoading ? (
            <Loading className="w-4 h-4 animate-spin" />
          ) : (
            "Sign Up with Email"
          )}
        </button>
      </form>
  );
};
