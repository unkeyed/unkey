"use client";

import * as React from "react";

import { Loading } from "@/components/dashboard/loading";
import { FadeInStagger } from "@/components/landing/fade-in";
import { Input } from "@/components/ui/input";
import { toast } from "@/components/ui/toaster";
import { useSignUp } from "@/lib/auth/hooks/useSignUp";
import { AuthErrorCode, errorMessages } from "@/lib/auth/types";

interface Props {
  setVerification: (value: boolean) => void;
}

export const EmailSignUp: React.FC<Props> = ({ setVerification }) => {
  const { handleSignUpViaEmail } = useSignUp();
  const [isLoading, setIsLoading] = React.useState(false);

  const signUpWithCode = async (e: React.FormEvent<HTMLFormElement>) => {
    e.preventDefault();
    const email = new FormData(e.currentTarget).get("email");
    const first = new FormData(e.currentTarget).get("first");
    const last = new FormData(e.currentTarget).get("last");

    if (typeof email !== "string" || typeof first !== "string" || typeof last !== "string") {
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
      toast.error(errorMessages[errorCode] || errorMessages[AuthErrorCode.UNKNOWN_ERROR]);
      console.error(err);
    } finally {
      setIsLoading(false);
    }
  };

  return (
    <FadeInStagger>
      <form className="grid gap-2" onSubmit={signUpWithCode}>
        <div className="grid gap-4">
          <div className="flex flex-row gap-3 ">
            <div className="flex flex-col items-start w-1/2 gap-2">
              <label htmlFor="first" className="text-xs text-white/50">
                First Name
              </label>
              <Input
                name="first"
                placeholder="Bruce"
                type="text"
                required
                autoCapitalize="none"
                autoCorrect="off"
                className="h-10 text-white duration-500 bg-transparent focus:text-black border-white/20 focus:bg-white focus:border-white hover:bg-white/20 hover:border-white/40 placeholder:white/20 "
              />
            </div>
            <div className="flex flex-col items-start w-1/2 gap-2">
              <label htmlFor="last" className="text-xs text-white/50">
                Last Name
              </label>
              <Input
                name="last"
                placeholder="Banner"
                type="text"
                required
                autoCapitalize="none"
                autoCorrect="off"
                className="h-10 text-white duration-500 bg-transparent focus:text-black border-white/20 focus:bg-white focus:border-white hover:bg-white/20 hover:border-white/40 placeholder:white/20 "
              />
            </div>
          </div>
          <div className="flex flex-col items-start gap-2">
            <label htmlFor="email" className="text-xs text-white/50">
              Email
            </label>
            <Input
              name="email"
              placeholder="name@example.com"
              type="email"
              autoCapitalize="none"
              autoComplete="email"
              autoCorrect="off"
              required
              className="h-10 text-white duration-500 bg-transparent focus:text-black border-white/20 focus:bg-white focus:border-white hover:bg-white/20 hover:border-white/40 placeholder:white/20 "
            />
          </div>
        </div>
        <button
          type="submit"
          className="flex items-center justify-center h-10 gap-2 px-4 mt-8 text-sm font-semibold text-black duration-200 bg-white border border-white rounded-lg hover:border-white/30 hover:bg-black hover:text-white"
          disabled={isLoading}
        >
          {isLoading ? <Loading className="w-4 h-4 animate-spin" /> : "Sign Up with Email"}
        </button>
      </form>
    </FadeInStagger>
  );
};
