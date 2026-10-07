"use client";

import Link from "next/link";
import { useEffect, useRef } from "react";

export function AuthenticationErrorContent() {
  const headingRef = useRef<HTMLHeadingElement>(null);

  useEffect(() => {
    headingRef.current?.focus();
  }, []);

  return (
    <div className="space-y-6 text-white">
      <div className="space-y-2">
        <h1 ref={headingRef} tabIndex={-1} className="font-semibold text-xl outline-none">
          We could not sign you in
        </h1>
        <p className="text-sm text-white/70">
          Your sign-in attempt could not be verified. Start a new sign-in attempt and try again.
        </p>
      </div>
      <div className="flex flex-col gap-3">
        <Link
          href="/auth/sign-in"
          className="inline-flex h-9 items-center justify-center rounded-md bg-white px-4 font-medium text-black text-sm hover:bg-white/90"
        >
          Sign in again
        </Link>
        <p className="text-xs text-white/60">
          If the problem continues, contact{" "}
          <Link className="underline" href="mailto:support@unkey.com">
            support@unkey.com
          </Link>
          .
        </p>
      </div>
    </div>
  );
}
