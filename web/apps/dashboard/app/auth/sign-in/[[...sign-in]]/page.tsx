"use client";

import { getCookie } from "@/lib/auth/cookies-actions";
import {
  AuthErrorCode,
  PENDING_SESSION_COOKIE,
  UNKEY_LAST_ORG_COOKIE,
  errorMessages,
} from "@/lib/auth/types";
import { ArrowRight } from "@unkey/icons";
import { Button, Empty, Loading, toast } from "@unkey/ui";
import type { Route } from "next";
import Link from "next/link";
import { useRouter, useSearchParams } from "next/navigation";
import { useEffect, useRef, useState } from "react";
import { ErrorBanner, WarnBanner } from "../../banners";
import { MfaChallenge } from "../../challenge/mfa-challenge";
import { MfaEnroll } from "../../challenge/mfa-enroll";
import { RadarEmailChallenge } from "../../challenge/radar-email-challenge";
import { RadarSmsChallenge } from "../../challenge/radar-sms-challenge";
import { SignInProvider } from "../../context/signin-context";
import { useSignIn } from "../../hooks";
import { EmailCode } from "../email-code";
import { EmailSignIn } from "../email-signin";
import { EmailVerify } from "../email-verify";
import { LastUsed, useLastUsed } from "../last_used";
import { OAuthSignIn } from "../oauth-signin";
import { OrgSelector } from "../org-selector";
import { saveRedirectUrl } from "../redirect-utils";

function SignInContent() {
  const {
    isVerifying,
    accountNotFound,
    error,
    email,
    hasPendingAuth,
    loading: pendingAuthLoading,
    orgs,
    handleSignInViaEmail,
    setError,
  } = useSignIn();
  const router = useRouter();
  const searchParams = useSearchParams();
  const verifyParam = searchParams?.get("verify");
  const challengeParam = searchParams?.get("challenge");
  const invitationToken = searchParams?.get("invitation_token");
  const invitationEmail = searchParams?.get("email");
  const redirectParam = searchParams?.get("redirect");
  const orgsParam = searchParams?.get("orgs");
  // Error surfaced via redirect (e.g. a Radar block on the OAuth callback,
  // which has no client action to set context error). Map the code to its
  // message, falling back to the generic one for unknown codes.
  const errorParam = searchParams?.get("error");
  const urlError = errorParam
    ? (errorMessages[errorParam as AuthErrorCode] ?? errorMessages[AuthErrorCode.UNKNOWN_ERROR])
    : null;
  const [lastUsedOrgId, setLastUsedOrgId] = useState<string | undefined>(undefined);
  // Add clientReady state to handle hydration
  const [clientReady, setClientReady] = useState(false);
  const [showEmail, setShowEmail] = useState(false);
  const [lastUsed] = useLastUsed();

  // Persist the redirect URL to sessionStorage so it survives the full auth
  // flow (OAuth redirects, org selection) even in browsers like Safari that
  // can lose URL params across redirect chains.
  useEffect(() => {
    if (redirectParam) {
      saveRedirectUrl(redirectParam);
    }
  }, [redirectParam]);

  // Errors arriving via redirect (e.g. a Radar block on the OAuth callback)
  // are surfaced as a toast, matching how every other flow shows them.
  useEffect(() => {
    if (urlError) {
      toast.error(urlError);
    }
  }, [urlError]);
  const hasAttemptedSignIn = useRef(false);
  // Used while the invitation auto sign-in sends its code
  const [isLoading, setIsLoading] = useState(false);

  // Set clientReady to true after hydration. The last-used org is only used
  // to highlight the matching entry in the manual org selector; automatic
  // selection happens server-side in /auth/continue before users land here.
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
      });
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
        setError(errorMessages[AuthErrorCode.PENDING_SESSION_EXPIRED]);
        // Clear the orgs query parameter to reset to sign-in form
        router.push("/auth/sign-in" as Route);
      }
    };

    // Check immediately when org selector is shown
    checkSessionValidity();

    // Then check periodically (every 30 seconds)
    const interval = setInterval(checkSessionValidity, 30000);

    return () => clearInterval(interval);
  }, [clientReady, hasPendingAuth, router, setError]);

  // While the invitation auto sign-in sends its code, or while the pending
  // session is being checked for an org-selection continuation, hold the
  // form back to avoid flashing the wrong step.
  if (isLoading || (orgsParam && pendingAuthLoading)) {
    return (
      <Empty>
        <Loading type="spinner" className="text-gray-6" />
        <p className="text-sm text-gray-11 mt-4">
          {invitationToken ? "Signing you in..." : "Loading..."}
        </p>
      </Empty>
    );
  }

  // Show the org selector when /auth/continue could not auto-select an org
  return hasPendingAuth ? (
    <OrgSelector organizations={orgs} lastOrgId={lastUsedOrgId} />
  ) : (
    <div className="flex flex-col gap-10">
      {accountNotFound && (
        <WarnBanner>
          <div className="flex items-center justify-between w-full gap-2">
            <p className="text-xs">Account not found, did you mean to sign up?</p>
            <Link href={`/auth/sign-up?email=${encodeURIComponent(email)}` as Route}>
              <div className="border text-center text-xs border-transparent hover:border-warning-8 text-warning-11 duration-200 p-1 rounded-lg">
                <ArrowRight iconSize="md-regular" />
              </div>
            </Link>
          </div>
        </WarnBanner>
      )}
      {error && <ErrorBanner>{error}</ErrorBanner>}

      {challengeParam === "mfa" ? (
        <MfaChallenge />
      ) : challengeParam === "mfa-enroll" ? (
        <MfaEnroll />
      ) : challengeParam === "radar-email" ? (
        <RadarEmailChallenge />
      ) : challengeParam === "radar-sms" ? (
        <RadarSmsChallenge />
      ) : isVerifying ? (
        <EmailCode invitationToken={invitationToken || undefined} />
      ) : verifyParam === "email" ? (
        <EmailVerify />
      ) : (
        <div className="flex flex-col gap-8">
          <h1 className="text-3xl font-semibold tracking-tight text-center text-gray-12">
            Log in to Unkey
          </h1>
          {showEmail ? (
            <div className="flex flex-col gap-4">
              <EmailSignIn />
              <button
                type="button"
                onClick={() => setShowEmail(false)}
                className="text-sm text-center cursor-pointer text-gray-11 transition-colors hover:text-gray-12"
              >
                &larr; Other options
              </button>
            </div>
          ) : (
            <div className="flex flex-col gap-2">
              <OAuthSignIn />
              <Button
                variant="ghost"
                size="xlg"
                className="w-full rounded-lg"
                onClick={() => setShowEmail(true)}
              >
                Continue with Email &rarr;
                {clientReady && lastUsed === "email" && <LastUsed />}
              </Button>
            </div>
          )}
        </div>
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
