import { TurnstileChallenge } from "@/components/auth/turnstile-challenge";
import type { PendingTurnstileResponse } from "@/lib/auth/types";
import { FormInput, Loading } from "@unkey/ui";
import { type FormEvent, useEffect, useState } from "react";
import { useSignIn } from "../hooks";
import { LastUsed, useLastUsed } from "./last_used";

export function EmailSignIn() {
  const { handleSignInViaEmail, email, handleTurnstileVerification, isPendingTurnstileChallenge } =
    useSignIn();
  const [isLoading, setIsLoading] = useState(false);
  const [lastUsed, setLastUsed] = useLastUsed();
  const [clientReady, setClientReady] = useState(false);
  const [turnstileChallenge, setTurnstileChallenge] = useState<PendingTurnstileResponse | null>(
    null,
  );
  const [isTurnstileLoading, setIsTurnstileLoading] = useState(false);
  const [currentEmail, setCurrentEmail] = useState(email || "");

  // Set clientReady to true after hydration is complete
  useEffect(() => {
    setClientReady(true);
  }, []);

  // Validate email format
  const isValidEmail = (email: string) => {
    return email.length > 0 && /^[^\s@]+@[^\s@]+\.[^\s@]+$/.test(email);
  };

  const isFormValid = isValidEmail(currentEmail);

  const handleSubmit = async (e: FormEvent<HTMLFormElement>) => {
    e.preventDefault();
    const formEmail = new FormData(e.currentTarget).get("email");
    if (typeof formEmail !== "string") {
      return;
    }

    setIsLoading(true);
    try {
      const result = await handleSignInViaEmail(formEmail);

      // Check if we got a Turnstile challenge
      if (result && isPendingTurnstileChallenge(result)) {
        setTurnstileChallenge(result);
        setIsLoading(false);
        return;
      }

      setLastUsed("email");
    } catch (_error) {
      // Error handling is done in the hook
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
      await handleTurnstileVerification(token, turnstileChallenge);
      setTurnstileChallenge(null);
      setLastUsed("email");
    } catch (_error) {
      // Error handling is done in the hook
    } finally {
      setIsTurnstileLoading(false);
    }
  };

  const handleTurnstileError = () => {
    setTurnstileChallenge(null);
  };

  // Show Turnstile challenge if needed
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
          Try a different email
        </button>
      </div>
    );
  }

  return (
    <form className="grid gap-16" onSubmit={handleSubmit}>
      <div className="grid gap-6">
        <div className="flex flex-col items-start gap-2">
          <FormInput
            label="Email"
            name="email"
            placeholder="name@example.com"
            type="email"
            defaultValue={email}
            autoCapitalize="none"
            autoComplete="email"
            autoCorrect="off"
            className="h-10 dark !bg-black w-full"
            onChange={(e) => setCurrentEmail(e.target.value)}
          />
        </div>
      </div>
      <button
        type="submit"
        className="relative flex items-center justify-center h-10 gap-2 px-4 text-sm font-semibold text-black duration-200 bg-white border border-white rounded-lg hover:border-white/30 hover:bg-black hover:text-white disabled:opacity-50 disabled:cursor-not-allowed disabled:hover:bg-white disabled:hover:text-black"
        disabled={isLoading || !isFormValid}
      >
        {clientReady && isLoading ? (
          <Loading className="w-4 h-4 animate-spin" />
        ) : (
          "Sign In with Email"
        )}
        {clientReady && lastUsed === "email" && <LastUsed />}
      </button>
    </form>
  );
}
