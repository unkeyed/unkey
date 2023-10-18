"use client";

import { useSignIn } from "@clerk/nextjs";
import * as React from "react";

import { Loading } from "@/components/dashboard/loading";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { useToast } from "@/components/ui/use-toast";
import { useRouter } from "next/navigation";
import { Dialog, DialogContent, DialogTitle } from "@/components/ui/dialog";

export function EmailSignIn(props: { verification: (value: boolean) => void, dialog: (value: boolean) => void, email: (value: string) => void, emailValue: string }) {
  const { signIn, isLoaded: signInLoaded, setActive } = useSignIn();
  const { toast } = useToast();
  const param = "__clerk_ticket";
  const [isLoading, setIsLoading] = React.useState(false);
  const router = useRouter();
  React.useEffect(() => {
    const signUpOrgUser = async () => {
      const ticket = new URL(window.location.href).searchParams.get(param);
      if (!signInLoaded) {
        return;
      }
      if (!ticket) {
        return;
      }
      setIsLoading(true);
      await signIn
        .create({
          strategy: "ticket",
          ticket,
        })
        .then((result) => {
          if (result.status === "complete" && result.createdSessionId) {
            setActive({ session: result.createdSessionId }).then(() => {
              router.push("/app");
            });
          }
        })
        .catch((err) => {
          setIsLoading(false);
          console.error(err);
        });
    };
    signUpOrgUser();
  }, [signInLoaded]);

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
          props.dialog(true);
          props.email(email);
        } else {
          toast({
            title: "Error",
            description: "Sorry, We couldn't sign you in. Please try again later",
            variant: "alert",
          });
        }
      });
  };
  return (
    <>
      <form className="grid gap-2" onSubmit={signInWithCode}>
        <div className="grid gap-1">
          <Input
            name="email"
            placeholder="name@example.com"
            type="email"
            defaultValue={props.emailValue}
            autoCapitalize="none"
            autoComplete="email"
            autoCorrect="off"
            className="bg-background"
          />
        </div>
        <Button disabled={isLoading}>
          {isLoading && <Loading className="w-4 h-4 mr-2 animate-spin" />}
          Sign In with Email
        </Button>
      </form>
    </>
  )
}
