"use client";
import { FadeIn } from "@/components/landing/fade-in";
import * as React from "react";
import { ErrorBanner } from "../../banners";
import { EmailCode } from "../email-code";
import { EmailSignUp } from "../email-signup";
import { OAuthSignUp } from "../oauth-signup";

import Link from "next/link";
import { SignUpProvider } from "@/lib/auth/context/signup-context";

export default function AuthenticationPage() {
  const [verify, setVerify] = React.useState(false);

  return (
    <SignUpProvider>
    <div className="flex flex-col justify-center space-y-6">
      {!verify ? (
        <>
          <div className="flex flex-col">
            <h1 className="text-4xl text-white">Create new account</h1>
            <p className="mt-4 text-sm text-md text-white/50">
              Sign up to Unkey or?
              <Link href="/auth/sign-in" className="ml-2 text-white hover:underline">
                Sign in
              </Link>
            </p>
          </div>
          <div className="grid gap-10 mt-4">
            <OAuthSignUp />
            <div className="relative">
              <div className="absolute inset-0 flex items-center">
                <span className="w-full border-t border-white/20" />
              </div>
              <div className="relative flex justify-center text-sm">
                <span className="px-2 bg-black text-white/50">or continue using email</span>
              </div>
            </div>
            <EmailSignUp setVerification={setVerify} />
          </div>
        </>
      ) : (
        <FadeIn>
          <EmailCode />
        </FadeIn>
      )}
    </div>
    </SignUpProvider>
  );
}
