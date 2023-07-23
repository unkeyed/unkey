"use client";

import { useSignUp } from "@clerk/nextjs";
import * as React from "react";

import { Loading } from "@/components/dashboard/loading";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { useToast } from "@/components/ui/use-toast";
import { useRouter } from "next/navigation";

export function EmailSignUp(props: { verification: (value: boolean) => void }) {
  const { signUp, isLoaded: signUpLoaded, setActive } = useSignUp();
  const { toast } = useToast();
  const param = "__clerk_ticket";
  const ticket = new URL(window.location.href).searchParams.get(param);
  const [isLoading, setIsLoading] = React.useState(false);
  const router = useRouter();

  React.useEffect(() => {
    const signUpOrgUser = async () => {
      if (!ticket) {
        return;
      }
      if (!signUpLoaded) {
        return;
      }
      await signUp
        .create({
          strategy: "ticket",
          ticket,
        })
        .then((result) => {
          if (result.status === "complete" && result.createdSessionId) {
            setActive({ session: result.createdSessionId }).then(() => {
              router.push("/onboarding");
            });
          }
        })
        .catch((err) => {
          setIsLoading(false);
          console.error(err);
        });
    };
    signUpOrgUser();
  }, [ticket, signUpLoaded]);

  const signUpWithCode = async (e: React.FormEvent<HTMLFormElement>) => {
    e.preventDefault();
    const email = new FormData(e.currentTarget).get("email");
    if (!signUpLoaded || typeof email !== "string") {
      return null;
    }
    setIsLoading(true);
    try {
      await signUp
        .create({
          emailAddress: email,
        })
        .then(async () => {
          await signUp.prepareEmailAddressVerification();
          setIsLoading(false);
          // set verification to true so we can show the code input
          props.verification(true);
        })
        .catch((err) => {
          setIsLoading(false);
          if (err.errors[0].code === "form_identifier_exists") {
            toast({
              title: "Error",
              description: "Sorry, it looks like you have an account. Please use sign in",
              variant: "destructive",
            });
          } else {
            toast({
              title: "Error",
              description: "Sorry, We couldn't sign you up. Please try again later",
              variant: "destructive",
            });
          }
        });
    } catch (error) {
      setIsLoading(false);
      console.error(error);
    }
  };

  return (
    <form className="grid gap-2" onSubmit={signUpWithCode}>
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
        {isLoading && <Loading className="w-4 h-4 mr-2 animate-spin" />}
        Sign Up with Email
      </Button>
    </form>
  );
}
