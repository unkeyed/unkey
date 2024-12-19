"use client";
import { FadeIn } from "@/components/landing/fade-in";
import { MoveRight } from "lucide-react";
import Link from "next/link";
import * as React from "react";
import { ErrorBanner, WarnBanner } from "../../banners";
import { EmailCode } from "../email-code";
import { EmailSignIn } from "../email-signin";
import { OAuthSignIn } from "../oauth-signin";

export default function AuthenticationPage() {
  const [verify, setVerify] = React.useState(false);
  const [accountNotFound, setAccountNotFound] = React.useState(false);
  const [error, setError] = React.useState<string | null>(null);
  const [email, setEmail] = React.useState("");

  return (
    <div className="flex flex-col gap-10">
      {accountNotFound ? (
        <WarnBanner>
          <div className="flex items-center justify-between w-full gap-2">
            <p className="text-xs">Account not found, did you mean to sign up?</p>

            <Link href={`/auth/sign-up?email=${encodeURIComponent(email)}`}>
              <div className="border text-center text-xs border-transparent hover:border-[#FFD55D]/50  text-[#FFD55D]  duration-200  p-1 rounded-lg ">
                <MoveRight className="w-4 h-4" />
              </div>
            </Link>
          </div>
        </WarnBanner>
      ) : null}
      {error ? <ErrorBanner>{error}</ErrorBanner> : null}

      {verify ? (
        <FadeIn>
          <EmailCode setError={setError} />
        </FadeIn>
      ) : (
        <>
          <div className="flex flex-col ">
            <h1 className="text-4xl text-white">Sign In</h1>
            <p className="mt-4 text-sm text-md text-white/50 ">
              New to Unkey?{" "}
              <Link href="/auth/sign-up" className="ml-2 text-white hover:underline">
                Create new account
              </Link>
            </p>
          </div>
          <div className="grid w-full gap-6">
            <OAuthSignIn />
            <div className="relative">
              <div className="absolute inset-0 flex items-center">
                <span className="w-full border-t border-white/20" />
              </div>
              <div className="relative flex justify-center text-sm">
                <span className="px-2 bg-black text-white/40">or continue using email</span>
              </div>
            </div>
            <div className="w-full">
              {/* <EmailSignIn
                setError={setError}
                verification={setVerify}
                setAccountNotFound={setAccountNotFound}
                email={setEmail}
                emailValue={email}
              /> */}
            </div>
          </div>
        </>
      )}
    </div>
  );
}
