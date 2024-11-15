"use client";

import { useSignUp } from "@clerk/nextjs";
import * as React from "react";

import { Loading } from "@/components/dashboard/loading";
import { FadeInStagger } from "@/components/landing/fade-in";
import { Input } from "@/components/ui/input";
import { toast } from "@/components/ui/toaster";
import { useRouter } from "next/navigation";

type Props = {
  setError: (e: string | null) => void;
  setVerification: (b: boolean) => void;
};

export const EmailSignUp: React.FC<Props> = ({ setError, setVerification }) => {
  const { signUp, isLoaded: signUpLoaded, setActive } = useSignUp();

  const [isLoading, setIsLoading] = React.useState(false);
  const [_transferLoading, setTransferLoading] = React.useState(true);
  const router = useRouter();

  // biome-ignore lint/correctness/useExhaustiveDependencies: effect must be called once if sign-up is loaded
  React.useEffect(() => {
    const signUpFromParams = async () => {
      if (!signUpLoaded) {
        return;
      }

      const ticket = new URL(window.location.href).searchParams.get("__clerk_ticket");
      const emailParam = new URL(window.location.href).searchParams.get("email");
      if (!ticket && !emailParam) {
        return;
      }
      if (ticket) {
        await signUp
          ?.create({
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
            setTransferLoading(false);
            setError((err as Error).message);
            console.error(err);
          });
      }

      if (emailParam) {
        setVerification(true);
        await signUp
          ?.create({
            emailAddress: emailParam,
          })
          .then(async () => {
            await signUp.prepareEmailAddressVerification();
            // set verification to true so we can show the code input
            setVerification(true);
            setTransferLoading(false);
          })
          .catch((err) => {
            setTransferLoading(false);
            if (err.errors[0].code === "form_identifier_exists") {
              toast.error("It looks like you have an account. Please use sign in");
            } else {
            }
          });
      }
    };

    signUpFromParams();
    setTransferLoading(false);
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
      setIsLoading(true);
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
          setVerification(true);
        })
        .catch((err) => {
          setIsLoading(false);
          if (err.errors[0].code === "form_identifier_exists") {
            toast.error("It looks like you have an account. Please use sign in");
          } else {
            toast.error("We couldn't sign you up. Please try again later");
          }
        });
    } catch (error) {
      setIsLoading(false);
      console.error(error);
    }
  };

  return (
    <FadeInStagger>
      <form className="grid gap-2" onSubmit={signUpWithCode}>
        <div className="grid gap-4">
          <div className="flex flex-row gap-3 ">
            <div className="flex flex-col items-start w-1/2 gap-2">
              <label htmlFor="first" className="text-xs text-white/50">
                First Name
              </label>
              <Input
                name="first"
                placeholder="Bruce"
                type="text"
                required
                autoCapitalize="none"
                autoCorrect="off"
                className="h-10 text-white duration-500 bg-transparent focus:text-black border-white/20 focus:bg-white focus:border-white hover:bg-white/20 hover:border-white/40 placeholder:white/20 "
              />
            </div>
            <div className="flex flex-col items-start w-1/2 gap-2">
              <label htmlFor="last" className="text-xs text-white/50">
                Last Name
              </label>
              <Input
                name="last"
                placeholder="Banner"
                type="text"
                required
                autoCapitalize="none"
                autoCorrect="off"
                className="h-10 text-white duration-500 bg-transparent focus:text-black border-white/20 focus:bg-white focus:border-white hover:bg-white/20 hover:border-white/40 placeholder:white/20 "
              />
            </div>
          </div>
          <div className="flex flex-col items-start gap-2">
            <label htmlFor="email" className="text-xs text-white/50">
              Email
            </label>
            <Input
              name="email"
              placeholder="name@example.com"
              type="email"
              autoCapitalize="none"
              autoComplete="email"
              autoCorrect="off"
              required
              className="h-10 text-white duration-500 bg-transparent focus:text-black border-white/20 focus:bg-white focus:border-white hover:bg-white/20 hover:border-white/40 placeholder:white/20 "
            />
          </div>
        </div>
        <button
          type="submit"
          className="flex items-center justify-center h-10 gap-2 px-4 mt-8 text-sm font-semibold text-black duration-200 bg-white border border-white rounded-lg hover:border-white/30 hover:bg-black hover:text-white"
          disabled={isLoading}
        >
          {isLoading ? <Loading className="w-4 h-4 animate-spin" /> : "Sign Up with Email"}
        </button>
      </form>
    </FadeInStagger>
  );
};
