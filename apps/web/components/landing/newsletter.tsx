"use client";

import { experimental_useFormStatus as useFormStatus } from "react-dom";
import { useToast } from "@/components/ui/use-toast";
import { Icons } from "@/components/ui/icons";
import { addEmail } from "@/app/actions/addEmail";
import { useRef } from "react";
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
    <button
      disabled={pending}
      className="flex aspect-square h-full items-center justify-center rounded-xl bg-neutral-950 text-white transition hover:bg-neutral-800 disabled:bg-neutral-500"
    >
      {pending ? <Icons.spinner className="w-4 h-4 animate-spin" /> : <ArrowIcon className="w-4" />}
    </button>
  );
};

export function NewsletterForm() {
  const formRef = useRef<HTMLFormElement>(null);
  const { toast } = useToast();
  console.log(formRef);
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
            variant: "destructive",
          });
        }
      }}
    >
      <h2 className="font-display text-sm font-semibold tracking-wider text-neutral-950">
        Sign up for our newsletter
      </h2>
      <p className="mt-4 text-sm text-neutral-700">Subscribe to get the latest Unkey news</p>
      <div className="relative mt-6">
        <input
          type="email"
          name="email"
          id='email'
          placeholder="Email address"
          autoComplete="email"
          aria-label="Email address"
          required
          className="block w-full rounded-2xl border border-neutral-300 bg-transparent py-4 pl-6 pr-20 text-base/6 text-neutral-950 ring-4 ring-transparent transition placeholder:text-neutral-500 focus:border-neutral-950 focus:outline-none focus:ring-neutral-950/5"
        />
        <div className="absolute inset-y-1 right-1 flex justify-end">
          <SubmitButton />
        </div>
      </div>
    </form>
  );
}
