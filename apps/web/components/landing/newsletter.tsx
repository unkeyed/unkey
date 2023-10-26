"use client";

import { addEmail } from "@/app/actions/addEmail";
import { useToast } from "@/components/ui/use-toast";
import { useRef } from "react";
// @ts-expect-error
import { experimental_useFormStatus as useFormStatus } from "react-dom";
import { Loading } from "../dashboard/loading";
import { Button } from "../ui/button";
// biome-ignore lint/suspicious/noExplicitAny: it's tailwindui's code
function ArrowIcon(props: any) {
  return (
    <svg viewBox="0 0 16 6" aria-hidden="true" {...props}>
      <title>Arrow</title>
      <path
        fill="currentColor"
        fillRule="evenodd"
        clipRule="evenodd"
        d="M16 3 10 .5v2H0v1h10v2L16 3Z"
      />
    </svg>
  );
}

const SubmitButton = () => {
  const { pending } = useFormStatus();
  return (
    <Button
      size="icon"
      disabled={pending}
      className="flex items-center justify-center h-full text-white transition aspect-square rounded-xl bg-gray-950 hover:bg-gray-800 disabled:bg-gray-500"
    >
      {pending ? <Loading className="w-4 h-4" /> : <ArrowIcon className="w-4" />}
    </Button>
  );
};

export function NewsletterForm() {
  const formRef = useRef<HTMLFormElement>(null);
  const { toast } = useToast();
  return (
    <form
      ref={formRef}
      className="max-w-md"
      action={async (formData) => {
        const test = await addEmail(formData);
        formRef.current?.reset();
        if (test.success) {
          toast({
            title: "Success",
            description: "Thanks for signing up!",
            variant: "default",
          });
        } else {
          toast({
            title: "Error",
            description: "Something went wrong, please try again later.",
            variant: "alert",
          });
        }
      }}
    >
      <h2 className="text-sm font-semibold tracking-wider font-display text-gray-950">
        Sign up for our newsletter
      </h2>
      <p className="mt-4 text-sm text-gray-700">Subscribe to get the latest Unkey news</p>
      <div className="relative mt-6">
        <input
          type="email"
          name="email"
          id="email"
          placeholder="Email address"
          autoComplete="email"
          aria-label="Email address"
          required
          className="block w-full py-4 pl-6 pr-20 transition bg-transparent border border-gray-300 rounded-2xl text-base/6 text-gray-950 ring-4 ring-transparent placeholder:text-gray-500 focus:border-gray-950 focus:outline-none focus:ring-gray-950/5"
        />
        <div className="absolute flex justify-end inset-y-1 right-1">
          <SubmitButton />
        </div>
      </div>
    </form>
  );
}
