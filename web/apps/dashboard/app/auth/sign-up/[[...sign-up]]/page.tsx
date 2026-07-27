"use client";
import { Button, Loading } from "@unkey/ui";
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
  const [showEmail, setShowEmail] = useState(false);
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
      <div className="flex flex-col justify-center gap-8">
        {isLoading && <Loading />}
        {verify ? (
          <EmailCode invitationToken={invitationToken || undefined} />
        ) : (
          <>
            <h1 className="text-3xl font-semibold tracking-tight text-center leading-tight text-gray-12">
              Go from zero to deployed in minutes.
            </h1>
            {showEmail ? (
              <div className="flex flex-col gap-4">
                <EmailSignUp setVerification={setVerify} />
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
                <OAuthSignUp />
                <Button
                  variant="ghost"
                  size="xlg"
                  className="w-full rounded-lg"
                  onClick={() => setShowEmail(true)}
                >
                  Continue with Email &rarr;
                </Button>
              </div>
            )}
          </>
        )}
      </div>
    </SignUpProvider>
  );
}
