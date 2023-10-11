"use client";

import { useSignUp } from "@clerk/nextjs";
import * as React from "react";

import { Loading } from "@/components/dashboard/loading";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { useToast } from "@/components/ui/use-toast";
import { useRouter } from "next/navigation";
import { FadeIn } from "@/components/landing/fade-in";
import { set } from "zod";

export function EmailSignUp(props: { verification: (value: boolean) => void }) {
  const { signUp, isLoaded: signUpLoaded, setActive } = useSignUp();
  const { toast } = useToast();
  
  const [isLoading, setIsLoading] = React.useState(false);
  const router = useRouter();

  React.useEffect(() => {
    setIsLoading(true);
    const signUpOrgUser = async () => {
      const ticket = new URL(window.location.href).searchParams.get("__clerk_ticket");
      if (!signUpLoaded) {
        return;
      }
      if (!ticket) {
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
              router.push("/app/apis");
            });
          }
        })
        .catch((err) => {
          setIsLoading(false);
          console.error(err);
        });
    };
    const addUserEmail = async () => {
      const emailParam = new URL(window.location.href).searchParams.get("email");
      if (!signUpLoaded) {
        return;
      }
      if (!emailParam) {
        return;
      }
      await signUp
        .create({
          emailAddress: emailParam,
        })
        .then(async () => {
          await signUp.prepareEmailAddressVerification();
          // set verification to true so we can show the code input
          props.verification(true);
          setIsLoading(false);
        })
        .catch((err) => {
          setIsLoading(false);
          if (err.errors[0].code === "form_identifier_exists") {
            toast({
              title: "Error",
              description: "Sorry, it looks like you have an account. Please use sign in",
              variant: "alert",
            });
          } else {
            console.log("Supress error")
          }
        });
    }
    addUserEmail();
    signUpOrgUser();
  }, [signUpLoaded]);

  const signUpWithCode = async (e: React.FormEvent<HTMLFormElement>) => {
    e.preventDefault();
    const email = new FormData(e.currentTarget).get("email");
    const first = new FormData(e.currentTarget).get("first");
    const last = new FormData(e.currentTarget).get("last");

    if (
      !signUpLoaded ||
      typeof email !== "string" ||
      typeof first !== "string" ||
      typeof last !== "string"
    ) {
      return null;
    }
    
    try {
      await signUp
        .create({
          emailAddress: email,
          firstName: first,
          lastName: last,
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
              variant: "alert",
            });
          } else {
            toast({
              title: "Error",
              description: "Sorry, We couldn't sign you up. Please try again later",
              variant: "alert",
            });
          }
        });
    } catch (error) {
      setIsLoading(false);
      console.error(error);
    }
  };

  return (
    <FadeIn>
    {!isLoading && (
    <form className="grid gap-2" onSubmit={signUpWithCode}>
      <div className="grid gap-1">
        <div className="flex flex-row gap-1 ">
          <Input
            name="first"
            placeholder="Bruce"
            type="text"
            required
            autoCapitalize="none"
            autoCorrect="off"
            className="bg-background"
          />
          <Input
            name="last"
            placeholder="Banner"
            type="text"
            required
            autoCapitalize="none"
            autoCorrect="off"
            className="bg-background"
          />
        </div>
        <Input
          name="email"
          placeholder="name@example.com"
          type="email"
          autoCapitalize="none"
          autoComplete="email"
          autoCorrect="off"
          required
          className="bg-background"
        />
      </div>
      <Button disabled={isLoading}>
        {isLoading && <Loading className="w-4 h-4 mr-2 animate-spin" />}
        Sign Up with Email
      </Button>
    </form>
    )}
    </FadeIn>
  );
}
