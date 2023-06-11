"use client";

import * as React from "react";
import { useRouter } from "next/navigation";
import { useSignIn } from "@clerk/nextjs";

import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { useToast } from "@/components/ui/use-toast";
import { Loading } from "@/components/loading";

export function EmailSignIn(props) {
  const [isLoading, setIsLoading] = React.useState(false);
  const { signIn, isLoaded: signInLoaded, setActive } = useSignIn();
  const router = useRouter();
  const { toast } = useToast();

  const signInWithCode = async (e: React.FormEvent<HTMLFormElement>) => {
    e.preventDefault();
    const email = new FormData(e.currentTarget).get("email");
    if (!signInLoaded || typeof email !== "string") return null;
    setIsLoading(true);
    await signIn
      .create({
        identifier: email,
      })
      .catch((error) => {
        console.log("sign-in error", JSON.stringify(error));
      });

    const firstFactor = signIn.supportedFirstFactors.find(
      (f) => f.strategy === "email_code",
    ) as { emailAddressId: string } | undefined;

    if (firstFactor) {
      await signIn.prepareFirstFactor({
        strategy: "email_code",
        emailAddressId: firstFactor.emailAddressId,
      });

      setIsLoading(false);
      // set verification to true so we can show the code input
      props.verification(true)
    }
  };

  return (
    <form className="grid gap-2" onSubmit={signInWithCode}>
      <div className="grid gap-1">
        <Input
          name="email"
          placeholder="name@example.com"
          type="email"
          autoCapitalize="none"
          autoComplete="email"
          autoCorrect="off"
          className="bg-background"
        />
      </div>
      <Button disabled={isLoading}>
        {isLoading && <Loading className="mr-2 h-4 w-4 animate-spin" />}
        Sign In with Email
      </Button>
    </form>
  );
}
