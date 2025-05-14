import { Loading } from "@/components/dashboard/loading";
import { type FormEvent, useEffect, useState } from "react";
import { useSignIn } from "../hooks";
import { LastUsed, useLastUsed } from "./last_used";
import {  FormInput,  } from "@unkey/ui";

export function EmailSignIn() {
  const { handleSignInViaEmail, email } = useSignIn();
  const [isLoading, setIsLoading] = useState(false);
  const [lastUsed, setLastUsed] = useLastUsed();
  const [clientReady, setClientReady] = useState(false);

  // Set clientReady to true after hydration is complete
  useEffect(() => {
    setClientReady(true);
  }, []);

  const handleSubmit = async (e: FormEvent<HTMLFormElement>) => {
    e.preventDefault();
    const formEmail = new FormData(e.currentTarget).get("email");
    if (typeof formEmail !== "string") {
      return;
    }

    setIsLoading(true);
    await handleSignInViaEmail(formEmail);
    setLastUsed("email");
    setIsLoading(false);
  };

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
          />
        </div>
      </div>
      <button
        type="submit"
        className="flex items-center justify-center h-10 gap-2 px-4 text-sm font-semibold text-black duration-200 bg-white border border-white rounded-lg hover:border-white/30 hover:bg-black hover:text-white"
        disabled={isLoading}
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
