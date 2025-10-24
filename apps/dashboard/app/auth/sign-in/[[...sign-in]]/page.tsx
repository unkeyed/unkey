"use client";

import { FadeIn } from "@/components/landing/fade-in";
import { ArrowRight } from "@unkey/icons";
import { Loading } from "@unkey/ui";
import Link from "next/link";
import { useSearchParams } from "next/navigation";
import { useEffect, useRef, useState } from "react";
import { ErrorBanner, WarnBanner } from "../../banners";
import { SignInProvider } from "../../context/signin-context";
import { useSignIn } from "../../hooks";
import { EmailCode } from "../email-code";
import { EmailSignIn } from "../email-signin";
import { EmailVerify } from "../email-verify";
import { OAuthSignIn } from "../oauth-signin";
import { OrgSelector } from "../org-selector";

function SignInContent() {
  const {
    isVerifying,
    accountNotFound,
    error,
    email,
    hasPendingAuth,
    orgs,
    handleSignInViaEmail,
    setError,
  } = useSignIn();
  const searchParams = useSearchParams();
  const verifyParam = searchParams?.get("verify");
  const invitationToken = searchParams?.get("invitation_token");
  const invitationEmail = searchParams?.get("email");

  // Initialize isLoading as false
  const [isLoading, setIsLoading] = useState(false);

  // Add clientReady state to handle hydration
  const [clientReady, setClientReady] = useState(false);
  const hasAttemptedSignIn = useRef(false);

  // Set clientReady to true after hydration
  useEffect(() => {
    setClientReady(true);
  }, []);

  // Handle auto sign-in with invitation token and email
  useEffect(() => {
    // Only run this effect on the client side after hydration
    if (!clientReady) {
      return;
    }

    const attemptAutoSignIn = async () => {
      // Only proceed if we have required data, aren't in other auth states, and haven't attempted sign-in yet
      if (
        invitationToken &&
        invitationEmail &&
        !isVerifying &&
        !hasPendingAuth &&
        !hasAttemptedSignIn.current
      ) {
        // Mark that we've attempted sign-in to prevent multiple attempts
        hasAttemptedSignIn.current = true;

        // Set loading state to true
        setIsLoading(true);

        try {
          // Attempt sign-in with the provided email
          await handleSignInViaEmail(invitationEmail);
        } catch (err) {
          console.error("Auto sign-in failed:", err);
        } finally {
          // Reset loading state
          setIsLoading(false);
        }
      }
    };

    attemptAutoSignIn();
  }, [
    clientReady,
    invitationToken,
    invitationEmail,
    isVerifying,
    hasPendingAuth,
    handleSignInViaEmail,
  ]);

  // Show a loading indicator only when isLoading is true AND client has hydrated
  if (clientReady && isLoading) {
    return <Loading />;
  }

  return (
    <div className="flex flex-col gap-10">
      {hasPendingAuth && <OrgSelector organizations={orgs} onError={setError} />}

      {accountNotFound && (
        <WarnBanner>
          <div className="flex items-center justify-between w-full gap-2">
            <p className="text-xs">Account not found, did you mean to sign up?</p>
            <Link href={`/auth/sign-up?email=${encodeURIComponent(email)}`}>
              <div className="border text-center text-xs border-transparent hover:border-[#FFD55D]/50 text-[#FFD55D] duration-200 p-1 rounded-lg">
                <ArrowRight iconsize="md-regular" />
              </div>
            </Link>
          </div>
        </WarnBanner>
      )}
      {error && <ErrorBanner>{error}</ErrorBanner>}

      {isVerifying ? (
        <FadeIn>
          <EmailCode invitationToken={invitationToken || undefined} />
        </FadeIn>
      ) : verifyParam === "email" ? (
        <FadeIn>
          <EmailVerify />
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
