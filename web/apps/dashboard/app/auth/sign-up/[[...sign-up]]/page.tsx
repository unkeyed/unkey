"use client";
import { FadeIn } from "@/components/landing/fade-in";
import { Loading } from "@unkey/ui";
import Link from "next/link";
import { useSearchParams } from "next/navigation";
import { useEffect, useRef, useState } from "react";
import { SignUpProvider } from "../../context/signup-context";
import { useSignUp } from "../../hooks";
import { EmailCode } from "../email-code";
import { EmailSignUp } from "../email-signup";
import { OAuthSignUp } from "../oauth-signup";

export default function AuthenticationPage() {
  const [verify, setVerify] = useState(false);
  const { handleSignUpViaEmail } = useSignUp();
  const searchParams = useSearchParams();
  const invitationToken = searchParams?.get("invitation_token");
  const invitationEmail = searchParams?.get("email");
  const [isLoading, setIsLoading] = useState(false);
  const hasAttemptedSignUp = useRef(false);

  // Handle auto sign-up with invitation token and email
  useEffect(() => {
    const attemptAutoSignUp = async () => {
      // Only proceed if we have the required data and haven't attempted sign-up yet
      if (invitationToken && invitationEmail && !hasAttemptedSignUp.current) {
        // Mark that we've attempted sign-up to prevent multiple attempts
        hasAttemptedSignUp.current = true;

        // Set loading state to true
        setIsLoading(true);

        try {
          // Attempt sign-in with the provided email
          await handleSignUpViaEmail({
            firstName: "", // they can set their first and
            lastName: "", // last name later
            email: invitationEmail,
          });
        } catch (err) {
          // Log auto sign-up errors for debugging
          console.error("Auto sign-up failed:", err);
        } finally {
          // Reset loading state
          setIsLoading(false);
        }
      }
    };

    attemptAutoSignUp();
  }, [invitationToken, invitationEmail, handleSignUpViaEmail]);

  return (
    <SignUpProvider>
      <div className="flex flex-col justify-center space-y-6">
        {isLoading && <Loading />}
        {verify ? (
          <FadeIn>
            <EmailCode invitationToken={invitationToken || undefined} />
          </FadeIn>
        ) : (
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
        )}
      </div>
    </SignUpProvider>
  );
}
