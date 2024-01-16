"use client";

import { addEmail } from "@/app/actions/addEmail";
import { toast } from "@/components/ui/toaster";
import { useRef } from "react";
import { useFormStatus } from "react-dom";
import { Loading } from "../dashboard/loading";
import { Button } from "../ui/button";

function ArrowIcon(props: any) {
  return (
    <svg viewBox="0 0 16 6" aria-hidden="true" {...props}>
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
      aria-label="Submit"
      size="icon"
      disabled={pending}
      className="flex aspect-square h-full items-center justify-center rounded-xl bg-gray-950 text-white transition hover:bg-gray-800 disabled:bg-gray-500"
    >
      {pending ? <Loading className="h-4 w-4" /> : <ArrowIcon className="w-4" />}
    </Button>
  );
};

export function NewsletterForm() {
  const formRef = useRef<HTMLFormElement>(null);
  return (
    <form
      ref={formRef}
      className="max-w-md"
      action={async (formData) => {
        const test = await addEmail(formData);
        formRef.current?.reset();
        if (test.success) {
          toast.success("Thanks for signing up!");
        } else {
          toast.error("Something went wrong, please try again later.");
        }
      }}
    >
      <h2 className="font-display text-sm font-semibold tracking-wider text-gray-950">
        Sign up for our newsletter
      </h2>
      <p className="mt-4 text-sm text-gray-700">Subscribe to get the latest Unkey news</p>
      <label htmlFor="email" className="sr-only">
        Enter email address to subscribe:
      </label>
      <div className="relative mt-6">
        <input
          type="email"
          name="email"
          id="email"
          placeholder="Email address"
          autoComplete="email"
          aria-label="Email address"
          required
          className="block w-full rounded-2xl border border-gray-300 bg-transparent py-4 pl-6 pr-20 text-base/6 text-gray-950 ring-4 ring-transparent transition placeholder:text-gray-500 focus:border-gray-950 focus:outline-none focus:ring-gray-950/5"
        />

        <div className="absolute inset-y-1 right-1 flex justify-end">
          <SubmitButton />
        </div>
      </div>
    </form>
  );
}
