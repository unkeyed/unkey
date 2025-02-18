"use client";

import { Loading } from "@/components/dashboard/loading";
import { Input } from "@/components/ui/input";
import { useSignIn } from "@clerk/nextjs";
import { useRouter } from "next/navigation";
import * as React from "react";
import { LastUsed, useLastUsed } from "./last_used";

export function EmailSignIn(props: {
  setError: (err: string) => void;
  verification: (value: boolean) => void;
  setAccountNotFound: (value: boolean) => void;
  email: (value: string) => void;
  emailValue: string;
}) {
  const { signIn, isLoaded: signInLoaded, setActive } = useSignIn();
  const param = "__clerk_ticket";
  const [isLoading, setIsLoading] = React.useState(false);
  const router = useRouter();
  const [lastUsed, setLastUsed] = useLastUsed();

  // biome-ignore lint/correctness/useExhaustiveDependencies: effect must be called once if sign-in is loaded
  React.useEffect(() => {
    const signUpOrgUser = async () => {
      if (!signInLoaded) {
        return;
      }

      const ticket = new URL(window.location.href).searchParams.get(param);
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
              router.push("/apis");
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
        setLastUsed("email");
      })
      .catch((err) => {
        setIsLoading(false);
        if (err.errors[0].code === "form_identifier_not_found") {
          props.setAccountNotFound(true);
          props.email(email);
        } else {
          props.setError("We couldn't sign you in. Please try again later");
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
            required
            className="h-10 text-white duration-500 bg-transparent focus:text-black border-white/20 focus:bg-white focus:border-white hover:bg-white/20 hover:border-white/40 placeholder:white/20 "
          />
        </div>
        <button
          type="submit"
          className="relative flex items-center justify-center h-10 gap-2 px-4 text-sm font-semibold text-black duration-200 bg-white border border-white rounded-lg hover:border-white/30 hover:bg-black hover:text-white"
          disabled={isLoading}
        >
          {isLoading ? <Loading className="w-4 h-4 animate-spin" /> : "Sign In with Email"}
          {lastUsed === "email" ? <LastUsed /> : null}
        </button>
      </form>
    </>
  );
}
