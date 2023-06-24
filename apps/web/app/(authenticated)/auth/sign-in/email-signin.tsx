"use client";

import { useSignIn } from "@clerk/nextjs";
import * as React from "react";

import { Loading } from "@/components/loading";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { useToast } from "@/components/ui/use-toast";

export function EmailSignIn(props: { verification: (value: boolean) => void }) {
  const { signIn, isLoaded: signInLoaded } = useSignIn();
  const { toast } = useToast();

  const [isLoading, setIsLoading] = React.useState(false);

  const signInWithCode = async (e: React.FormEvent<HTMLFormElement>) => {
    e.preventDefault();
    const email = new FormData(e.currentTarget).get("email");
    if (!signInLoaded || typeof email !== "string") {
      return null;
    }
    setIsLoading(true);
    await signIn
      .create({
        identifier: email,
      })
      .then(async () => {
        const firstFactor = signIn.supportedFirstFactors.find((f) => f.strategy === "email_code") as
          | { emailAddressId: string }
          | undefined;

        if (firstFactor) {
          await signIn.prepareFirstFactor({
            strategy: "email_code",
            emailAddressId: firstFactor.emailAddressId,
          });

          setIsLoading(false);
          props.verification(true);
        }
      })
      .catch((err) => {
        setIsLoading(false);
        if (err.errors[0].code === "form_identifier_not_found") {
          toast({
            title: "Error",
            description: "Sorry, We couldn't find your account. Please use sign up",
            variant: "destructive",
          });
        } else {
          toast({
            title: "Error",
            description: "Sorry, We couldn't sign you in. Please try again later",
            variant: "destructive",
          });
        }
      });
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
