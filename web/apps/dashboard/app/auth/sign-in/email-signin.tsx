import { Button, FormInput } from "@unkey/ui";
import { type FormEvent, useEffect, useState } from "react";
import { useSignIn } from "../hooks";
import { LastUsed, useLastUsed } from "./last_used";

export function EmailSignIn() {
  const { handleSignInViaEmail, email } = useSignIn();
  const [isLoading, setIsLoading] = useState(false);
  const [lastUsed, setLastUsed] = useLastUsed();
  const [clientReady, setClientReady] = useState(false);
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
      await handleSignInViaEmail(formEmail);
      setLastUsed("email");
    } catch (_error) {
      // Error handling is done in the hook
    } finally {
      setIsLoading(false);
    }
  };

  return (
    <form className="flex flex-col gap-4" onSubmit={handleSubmit}>
      <FormInput
        label="Email"
        name="email"
        placeholder="name@example.com"
        type="email"
        defaultValue={email}
        autoCapitalize="none"
        autoComplete="email"
        autoCorrect="off"
        className="w-full"
        onChange={(e) => setCurrentEmail(e.target.value)}
      />
      <Button
        type="submit"
        variant="primary"
        size="xlg"
        className="relative w-full rounded-lg"
        disabled={isLoading || !isFormValid}
        loading={clientReady && isLoading}
      >
        Continue with Email
        {clientReady && lastUsed === "email" && <LastUsed />}
      </Button>
    </form>
  );
}
