"use client";

import { experimental_useFormStatus as useFormStatus } from "react-dom";
import { useToast } from "@/components/ui/use-toast";
import { Icons } from "@/components/ui/icons";
import { addEmail } from "@/app/actions/addEmail";
import { useRef } from "react";
// rome-ignore lint/suspicious/noExplicitAny: it's tailwindui's code
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
      className="flex items-center justify-center h-full text-white transition aspect-square rounded-xl bg-neutral-950 hover:bg-neutral-800 disabled:bg-neutral-500"
    >
      {pending ? <Icons.spinner className="w-4 h-4 animate-spin" /> : <ArrowIcon className="w-4" />}
    </button>
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
            variant: "destructive",
          });
        }
      }}
    >
      <h2 className="text-sm font-semibold tracking-wider font-display text-neutral-950">
        Sign up for our newsletter
      </h2>
      <p className="mt-4 text-sm text-neutral-700">Subscribe to get the latest Unkey news</p>
      <div className="relative mt-6">
        <input
          type="email"
          name="email"
          id="email"
          placeholder="Email address"
          autoComplete="email"
          aria-label="Email address"
          required
          className="block w-full py-4 pl-6 pr-20 transition bg-transparent border rounded-2xl border-neutral-300 text-base/6 text-neutral-950 ring-4 ring-transparent placeholder:text-neutral-500 focus:border-neutral-950 focus:outline-none focus:ring-neutral-950/5"
        />
        <div className="absolute flex justify-end inset-y-1 right-1">
          <SubmitButton />
        </div>
      </div>
    </form>
  );
}
