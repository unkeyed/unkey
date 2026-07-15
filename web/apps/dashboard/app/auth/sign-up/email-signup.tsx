"use client";

import * as React from "react";

import { AuthErrorCode, errorMessages } from "@/lib/auth/types";
import { Button, FormInput, toast } from "@unkey/ui";
import { useSearchParams } from "next/navigation";
import { useSignUp } from "../hooks/useSignUp";

interface Props {
  setVerification: (value: boolean) => void;
}

export const EmailSignUp: React.FC<Props> = ({ setVerification }) => {
  const { handleSignUpViaEmail } = useSignUp();
  const [isLoading, setIsLoading] = React.useState(false);
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

      // If successful, proceed to verification
      if (result?.success) {
        setVerification(true);
      } else if (result && !result.success) {
        toast.error(result.message || errorMessages[AuthErrorCode.UNKNOWN_ERROR]);
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

  return (
    <form className="flex flex-col gap-4" onSubmit={signUpWithCode}>
      {validationError && (
        <div
          className="p-3 text-sm rounded-lg border text-error-11 bg-error-3 border-error-6"
          role="alert"
          aria-live="polite"
        >
          {validationError}
        </div>
      )}
      <div className="flex flex-row gap-3">
        <FormInput
          label="First Name"
          name="first"
          placeholder="Bruce"
          type="text"
          autoCapitalize="none"
          autoCorrect="off"
          className="w-1/2 [&_label]:text-sm [&_input]:text-sm"
          onChange={(e) => {
            setFirstName(e.target.value);
            validationError && setValidationError("");
          }}
        />
        <FormInput
          label="Last Name"
          name="last"
          placeholder="Banner"
          type="text"
          autoCapitalize="none"
          autoCorrect="off"
          className="w-1/2 [&_label]:text-sm [&_input]:text-sm"
          onChange={(e) => {
            setLastName(e.target.value);
            validationError && setValidationError("");
          }}
        />
      </div>
      <FormInput
        label="Email"
        name="email"
        defaultValue={emailFromParams}
        placeholder="name@example.com"
        type="email"
        autoCapitalize="none"
        autoComplete="email"
        autoCorrect="off"
        className="w-full [&_label]:text-sm [&_input]:text-sm"
        onChange={(e) => {
          setEmail(e.target.value);
          validationError && setValidationError("");
        }}
      />
      <Button
        type="submit"
        variant="primary"
        size="xlg"
        className="w-full rounded-lg"
        disabled={isLoading || !isFormValid}
        loading={clientLoaded && isLoading}
      >
        Continue with Email
      </Button>
    </form>
  );
};
