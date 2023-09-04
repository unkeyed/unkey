"use client";
import { FadeIn } from "@/components/landing/fade-in";
import { useAuth } from "@clerk/nextjs";
import * as React from "react";
import { EmailCode } from "../email-code";
import { EmailSignIn } from "../email-signin";
import { OAuthSignIn } from "../oauth-signin";

export const runtime = "edge";

export default function AuthenticationPage() {
  const [verify, setVerify] = React.useState(false);
  const { isLoaded } = useAuth();

  if (!isLoaded) {
    return null;
  }
  return (
    <div className="mx-auto flex w-full flex-col justify-center space-y-6 px-6 md:px-0 sm:w-[500px]">
      {!verify && (
        <>
          <div className="flex flex-col space-y-2 text-center">
            <h1 className="text-3xl font-semibold tracking-tight">Sign In to Unkey</h1>
            <p className="text-md text-content-subtle">Enter your email below to sign in</p>
          </div>
          <div className="grid gap-6">
            <EmailSignIn verification={setVerify} />

            <div className="relative">
              <div className="absolute inset-0 flex items-center">
                <span className="w-full border-t" />
              </div>
              <div className="relative flex justify-center text-xs uppercase">
                <span className="bg-background px-2 text-content-subtle">Or continue with</span>
              </div>
            </div>
            <OAuthSignIn />
          </div>
          <div className="relative flex justify-center text-xs uppercase">
            <span className="bg-background px-2 text-content-subtle">
              Not been here before? Just{" "}
              <a className="text-black" href="/auth/sign-up">
                Sign Up
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
