"use client";
import { FadeIn } from "@/components/landing/fade-in";
import { useAuth } from "@clerk/nextjs";
import * as React from "react";
import { EmailCode } from "../email-code";
import { EmailSignUp } from "../email-signup";
import { OAuthSignUp } from "../oauth-signup";

export default function AuthenticationPage() {
  const { isLoaded } = useAuth();

  if (!isLoaded) {
    return null;
  }

  const [verify, setVerify] = React.useState(false);
  return (
    <div className="mx-auto flex w-full flex-col justify-center space-y-6 px-2 sm:w-[500px] md:px-0">
      {!verify && (
        <>
          <div className="flex flex-col space-y-2 text-center">
            <h1 className="text-3xl font-semibold tracking-tight">Sign Up to Unkey</h1>
            <p className="text-md text-content-subtle">Enter your email below to sign up</p>
          </div>
          <div className="grid gap-6">
            <EmailSignUp verification={setVerify} />

            <div className="relative">
              <div className="absolute inset-0 flex items-center">
                <span className="w-full border-t" />
              </div>
              <div className="relative flex justify-center text-xs uppercase">
                <span className="bg-background text-content-subtle px-2">Or continue with</span>
              </div>
            </div>
            <OAuthSignUp />
          </div>
          <div className="relative flex justify-center text-xs uppercase">
            <span className="bg-background text-content-subtle px-2">
              Already been here before? Just{" "}
              <a className="text-black" href="/auth/sign-in">
                Sign In
              </a>
            </span>
          </div>
        </>
      )}
      {verify && (
        <FadeIn>
          <div className="flex flex-col space-y-2 text-center">
            <h1 className="text-3xl font-semibold tracking-tight">Enter your email code</h1>
            <p className="text-md text-content-subtle">We sent you a 6 digit code to your email</p>
            <EmailCode />
          </div>
        </FadeIn>
      )}
    </div>
  );
}
