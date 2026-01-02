"use client";

import * as React from "react";

import { TurnstileChallenge } from "@/components/auth/turnstile-challenge";
import { AuthErrorCode, type PendingTurnstileResponse, errorMessages } from "@/lib/auth/types";
import { FormInput, Loading, toast } from "@unkey/ui";
import { useSearchParams } from "next/navigation";
import { useSignUp } from "../hooks/useSignUp";

interface Props {
  setVerification: (value: boolean) => void;
}

export const EmailSignUp: React.FC<Props> = ({ setVerification }) => {
  const { handleSignUpViaEmail, handleTurnstileVerification, isPendingTurnstileChallenge } =
    useSignUp();
  const [isLoading, setIsLoading] = React.useState(false);
  const [isTurnstileLoading, setIsTurnstileLoading] = React.useState(false);
  const [turnstileChallenge, setTurnstileChallenge] =
    React.useState<PendingTurnstileResponse | null>(null);
  const [validationError, setValidationError] = React.useState<string>("");
  const searchParams = useSearchParams();
  const emailFromParams = searchParams?.get("email") || "";
  const [firstName, setFirstName] = React.useState("");
  const [lastName, setLastName] = React.useState("");
  const [email, setEmail] = React.useState(emailFromParams);

  //fix hydration error with the loading state
  const [clientLoaded, setClientLoaded] = React.useState(false);

  React.useEffect(() => {
    setClientLoaded(true);
  }, []);

  // Validate form fields
  const isValidEmail = (email: string) => {
    return email.length > 0 && /^[^\s@]+@[^\s@]+\.[^\s@]+$/.test(email);
  };

  const isFormValid =
    firstName.trim().length > 0 && lastName.trim().length > 0 && isValidEmail(email);

  const signUpWithCode = async (e: React.FormEvent<HTMLFormElement>) => {
    e.preventDefault();
    setValidationError(""); // Clear any previous errors

    const formData = new FormData(e.currentTarget);
    const email = formData.get("email");
    const first = formData.get("first");
    const last = formData.get("last");

    // Validate required fields and convert to strings
    const missingFields: string[] = [];
    if (typeof email !== "string" || !email.trim()) {
      missingFields.push("Email");
    }
    if (typeof first !== "string" || !first.trim()) {
      missingFields.push("First Name");
    }
    if (typeof last !== "string" || !last.trim()) {
      missingFields.push("Last Name");
    }

    if (missingFields.length > 0) {
      setValidationError(
        `Please fill in the following required fields: ${missingFields.join(", ")}`,
      );
      return;
    }

    try {
      setIsLoading(true);
      const result = await handleSignUpViaEmail({
        email: email as string,
        firstName: first as string,
        lastName: last as string,
      });

      // Check if we got a Turnstile challenge
      if (result && isPendingTurnstileChallenge(result)) {
        setTurnstileChallenge(result);
        setIsLoading(false);
        return;
      }

      // If successful, proceed to verification
      if (result?.success) {
        setVerification(true);
      }
    } catch (err: unknown) {
      const errorCode =
        err !== null &&
        typeof err === "object" &&
        "message" in err &&
        typeof (err as { message: string }).message === "string"
          ? ((err as { message: string }).message as AuthErrorCode)
          : AuthErrorCode.UNKNOWN_ERROR;
      toast.error(errorMessages[errorCode] || errorMessages[AuthErrorCode.UNKNOWN_ERROR]);
    } finally {
      setIsLoading(false);
    }
  };

  const handleTurnstileSuccess = async (token: string) => {
    if (!turnstileChallenge) {
      return;
    }
    setIsTurnstileLoading(true);
    try {
      const result = await handleTurnstileVerification(token, turnstileChallenge);

      if (result?.success) {
        setTurnstileChallenge(null);
        setVerification(true);
      } else {
        toast.error("Verification failed. Please try again.");
      }
    } catch (_error) {
      toast.error("Verification failed. Please try again.");
    } finally {
      setIsTurnstileLoading(false);
    }
  };

  const handleTurnstileError = () => {
    setTurnstileChallenge(null);
    toast.error("Verification failed. Please try again.");
  };
  if (turnstileChallenge) {
    return (
      <div className="grid gap-6">
        <TurnstileChallenge
          email={turnstileChallenge.email}
          onSuccess={handleTurnstileSuccess}
          onError={handleTurnstileError}
          isLoading={isTurnstileLoading}
        />
        <button
          type="button"
          onClick={() => setTurnstileChallenge(null)}
          className="text-sm text-gray-400 hover:text-white underline"
        >
          Try different information
        </button>
      </div>
    );
  }

  return (
    <form className="grid gap-16" onSubmit={signUpWithCode}>
      <div className="grid gap-10">
        {validationError && (
          <div
            className="p-3 text-sm text-red-400 bg-red-900/20 border border-red-800 rounded-lg"
            role="alert"
            aria-live="polite"
          >
            {validationError}
          </div>
        )}
        <div className="flex flex-row gap-3">
          <div className="flex flex-col items-start w-1/2 gap-2">
            <FormInput
              label="First Name"
              name="first"
              placeholder="Bruce"
              type="text"
              autoCapitalize="none"
              autoCorrect="off"
              className="h-10 dark !bg-black"
              onChange={(e) => {
                setFirstName(e.target.value);
                validationError && setValidationError("");
              }}
            />
          </div>
          <div className="flex flex-col items-start w-1/2 gap-2">
            <FormInput
              label="Last Name"
              name="last"
              placeholder="Banner"
              type="text"
              autoCapitalize="none"
              autoCorrect="off"
              className="h-10 dark !bg-black"
              onChange={(e) => {
                setLastName(e.target.value);
                validationError && setValidationError("");
              }}
            />
          </div>
        </div>
        <div className="flex flex-col items-start gap-2">
          <FormInput
            label="Email"
            name="email"
            defaultValue={emailFromParams}
            placeholder="name@example.com"
            type="email"
            autoCapitalize="none"
            autoComplete="email"
            autoCorrect="off"
            className="h-10 dark !bg-black w-full"
            onChange={(e) => {
              setEmail(e.target.value);
              validationError && setValidationError("");
            }}
          />
        </div>
      </div>
      <button
        type="submit"
        className="flex items-center justify-center h-10 gap-2 px-4 mt-8 text-sm font-semibold text-black duration-200 bg-white border border-white rounded-lg hover:border-white/30 hover:bg-black hover:text-white disabled:opacity-50 disabled:cursor-not-allowed disabled:hover:bg-white disabled:hover:text-black"
        disabled={isLoading || !isFormValid}
      >
        {clientLoaded && isLoading ? (
          <Loading className="w-4 h-4 animate-spin" />
        ) : (
          "Sign Up with Email"
        )}
      </button>
    </form>
  );
};
