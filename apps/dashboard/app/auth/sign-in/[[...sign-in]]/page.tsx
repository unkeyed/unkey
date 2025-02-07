"use client";

import { FadeIn } from "@/components/landing/fade-in";
import { MoveRight } from "lucide-react";
import Link from "next/link";
import { ErrorBanner, WarnBanner } from "../../banners";
import { EmailCode } from "../email-code";
import { EmailSignIn } from "../email-signin";
import { OAuthSignIn } from "../oauth-signin";
import { SignInProvider } from "@/lib/auth/context/signin-context";
import { useSignIn } from "@/lib/auth/hooks";
import { OrgSelector } from "../org-selector";
import { Dialog, DialogContent, DialogDescription, DialogHeader } from "@/components/ui/dialog";

function SignInContent() {
  const { isVerifying, accountNotFound, error, email, hasPendingAuth, orgs } = useSignIn();

  return (
    <div className="flex flex-col gap-10">

      {hasPendingAuth &&
          <OrgSelector organizations={orgs} />
      }

      {accountNotFound && (
        <WarnBanner>
          <div className="flex items-center justify-between w-full gap-2">
            <p className="text-xs">Account not found, did you mean to sign up?</p>
            <Link href={`/auth/sign-up?email=${encodeURIComponent(email)}`}>
              <div className="border text-center text-xs border-transparent hover:border-[#FFD55D]/50 text-[#FFD55D] duration-200 p-1 rounded-lg">
                <MoveRight className="w-4 h-4" />
              </div>
            </Link>
          </div>
        </WarnBanner>
      )}
      {error && <ErrorBanner>{error}</ErrorBanner>}

      {isVerifying ? (
        <FadeIn>
          <EmailCode />
        </FadeIn>
      ) : (
        <>
          <div className="flex flex-col">
            <h1 className="text-4xl text-white">Sign In</h1>
            <p className="mt-4 text-sm text-md text-white/50">
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
            <EmailSignIn />
          </div>
        </>
      )}
    </div>
  );
}

export default function AuthenticationPage() {
  return (
    <SignInProvider>
      <SignInContent />
    </SignInProvider>
  );
}