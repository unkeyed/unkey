"use client";

import { FadeIn } from "@/components/landing/fade-in";
import { ArrowRight } from "@unkey/icons";
import { Empty, Loading } from "@unkey/ui";
import Link from "next/link";
import { useSearchParams } from "next/navigation";
import { useEffect, useRef, useState } from "react";
import { completeOrgSelection } from "../../actions";
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

  // Add clientReady state to handle hydration
  const [clientReady, setClientReady] = useState(false);
  const hasAttemptedSignIn = useRef(false);
  const hasAttemptedAutoOrgSelection = useRef(false);
  const [isLoading, setIsLoading] = useState(false);

  // Helper function to get cookie value on client side
  const getCookie = (name: string): string | null => {
    if (typeof document === "undefined") {
      return null;
    }
    const value = `; ${document.cookie}`;
    const parts = value.split(`; ${name}=`);
    if (parts.length === 2) {
      return parts.pop()?.split(";").shift() || null;
    }
    return null;
  };

  // Set clientReady to true after hydration
  useEffect(() => {
    setClientReady(true);
  }, []);

  // Handle auto org selection when returning from OAuth
  useEffect(() => {
    if (!clientReady || !hasPendingAuth || hasAttemptedAutoOrgSelection.current) {
      return;
    }

    const lastUsedOrgId = getCookie("unkey_last_org_used");
    if (lastUsedOrgId) {
      hasAttemptedAutoOrgSelection.current = true;
      setIsLoading(true);

      completeOrgSelection(lastUsedOrgId)
        .then((result) => {
          if (!result.success) {
            setError(result.message);
            setIsLoading(false);
            return;
          }
          // On success, redirect to the dashboard
          window.location.href = result.redirectTo;
        })
        .catch((err) => {
          console.error("Auto org selection failed:", err);
          setError("Failed to automatically sign in. Please select your workspace.");
          setIsLoading(false);
        });
    }
  }, [clientReady, hasPendingAuth, setError]);

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

  // Show a loading indicator when auto-selecting org
  if (isLoading && clientReady) {
    return (
      <Empty>
        <Loading type="spinner" />
        <p className="text-sm text-white/60 mt-4">Signing you in...</p>
      </Empty>
    );
  }
  return hasPendingAuth ? (
    <OrgSelector organizations={orgs} lastOrgId={getCookie("unkey_last_org_used") || undefined} />
  ) : (
    <div className="flex flex-col gap-10">
      {accountNotFound && (
        <WarnBanner>
          <div className="flex items-center justify-between w-full gap-2">
            <p className="text-xs">Account not found, did you mean to sign up?</p>
            <Link href={`/auth/sign-up?email=${encodeURIComponent(email)}`}>
              <div className="border text-center text-xs border-transparent hover:border-[#FFD55D]/50 text-[#FFD55D] duration-200 p-1 rounded-lg">
                <ArrowRight iconSize="md-regular" />
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
