import { Loading } from "@/components/dashboard/loading";
import { Input } from "@/components/ui/input";
import { useSignIn } from "@/lib/auth/hooks";
import { type FormEvent, useState } from "react";
import { LastUsed, useLastUsed } from "./last_used";

export function EmailSignIn() {
  const { handleSignInViaEmail, email } = useSignIn();
  const [isLoading, setIsLoading] = useState(false);
  const [lastUsed, setLastUsed] = useLastUsed();

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
    <form className="grid gap-2" onSubmit={handleSubmit}>
      <div className="grid gap-1">
        <Input
          name="email"
          placeholder="name@example.com"
          type="email"
          defaultValue={email}
          autoCapitalize="none"
          autoComplete="email"
          autoCorrect="off"
          required
          className="h-10 text-white duration-500 bg-transparent focus:text-black border-white/20 focus:bg-white focus:border-white hover:bg-white/20 hover:border-white/40 placeholder:white/20"
        />
      </div>
      <button
        type="submit"
        className="relative flex items-center justify-center h-10 gap-2 px-4 text-sm font-semibold text-black duration-200 bg-white border border-white rounded-lg hover:border-white/30 hover:bg-black hover:text-white"
        disabled={isLoading}
      >
        {isLoading ? <Loading className="w-4 h-4 animate-spin" /> : "Sign In with Email"}
        {lastUsed === "email" && <LastUsed />}
      </button>
    </form>
  );
}
