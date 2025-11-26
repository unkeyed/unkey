"use client";

import { FadeIn } from "@/components/landing/fade-in";
import { getCookie } from "@/lib/auth/cookies";
import { PENDING_SESSION_COOKIE, UNKEY_LAST_ORG_COOKIE } from "@/lib/auth/types";
import { ArrowRight } from "@unkey/icons";
import { Empty, Loading } from "@unkey/ui";
import Link from "next/link";
import { useRouter, useSearchParams } from "next/navigation";
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
  const router = useRouter();
  const searchParams = useSearchParams();
  const verifyParam = searchParams?.get("verify");
  const invitationToken = searchParams?.get("invitation_token");
  const invitationEmail = searchParams?.get("email");
  const [lastUsedOrgId, setLastUsedOrgId] = useState<string | undefined>(undefined);
  // Add clientReady state to handle hydration
  const [clientReady, setClientReady] = useState(false);
  const hasAttemptedSignIn = useRef(false);
  const hasAttemptedAutoOrgSelection = useRef(false);
  const [isLoading, setIsLoading] = useState(true);
  const [isAutoSelecting, setIsAutoSelecting] = useState(false);

  // Set clientReady to true after hydration
  useEffect(() => {
    getCookie(UNKEY_LAST_ORG_COOKIE)
      .then((value) => {
        if (value) {
          setLastUsedOrgId(value);
        }
      })
      .catch((_error) => {
        // Ignore cookie read errors
      })
      .finally(() => {
        setClientReady(true);
        // Only set isLoading to false if we don't have pending auth
        // (if we do, the auto-selection effect will handle loading state)
        if (!hasPendingAuth) {
          setIsLoading(false);
        }
      });
  }, [hasPendingAuth]);

  // Handle auto org selection when returning from OAuth
  useEffect(() => {
    if (!clientReady || !hasPendingAuth || hasAttemptedAutoOrgSelection.current) {
      return;
    }

    if (lastUsedOrgId) {
      hasAttemptedAutoOrgSelection.current = true;
      setIsLoading(true);
      setIsAutoSelecting(true);

      completeOrgSelection(lastUsedOrgId)
        .then((result) => {
          if (!result.success) {
            setError(result.message);
            setIsLoading(false);
            setIsAutoSelecting(false);
            return;
          }
          // On success, redirect to the dashboard
          router.push(result.redirectTo);
        })
        .catch((_err) => {
          setError("Failed to automatically sign in. Please select your workspace.");
          setIsLoading(false);
          setIsAutoSelecting(false);
        });
    } else {
      // No lastUsedOrgId, so we need to show the org selector manually
      hasAttemptedAutoOrgSelection.current = true;
      setIsLoading(false);
    }
  }, [clientReady, hasPendingAuth, setError, lastUsedOrgId, router]);

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
        } catch (_err) {
          // Ignore auto sign-in errors
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

  // Check for session expiration when org selector is shown
  useEffect(() => {
    if (!clientReady || !hasPendingAuth) {
      return;
    }

    const checkSessionValidity = async () => {
      const pendingSession = await getCookie(PENDING_SESSION_COOKIE);
      if (!pendingSession) {
        setError("Your session has expired. Please sign in again.");
        // Clear the orgs query parameter to reset to sign-in form
        router.push("/auth/sign-in");
      }
    };

    // Check immediately when org selector is shown
    checkSessionValidity();

    // Then check periodically (every 30 seconds)
    const interval = setInterval(checkSessionValidity, 30000);

    return () => clearInterval(interval);
  }, [clientReady, hasPendingAuth, router, setError]);

  if (isAutoSelecting || isLoading) {
    let message = "Loading...";
    if (isLoading) {
      message = "Loading last workspace...";
    }
    return (
      <Empty>
        <Loading type="spinner" className="text-gray-6" />
        <p className="text-sm text-white/60 mt-4">{message}</p>
      </Empty>
    );
  }

  const handleOrgSelectorClose = () => {
    // When user closes the org selector, navigate back to clean sign-in page
    router.push("/auth/sign-in");
  };

  // Only show org selector if we have pending auth and we're not actively auto-selecting
  return hasPendingAuth && !isAutoSelecting ? (
    <OrgSelector organizations={orgs} lastOrgId={lastUsedOrgId} onClose={handleOrgSelectorClose} />
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
