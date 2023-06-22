"use client";
import { AuthenticateWithRedirectCallback } from "@clerk/nextjs";

import { Icons } from "@/components/ui/icons";

export const runtime = "edge";

export default function SSOCallback() {
  return (
    <div className="h-screen flex items-center justify-center ">
      <Icons.spinner className="mr-2 h-16 w-16 animate-spin" />
      <AuthenticateWithRedirectCallback afterSignInUrl="/app/apis" afterSignUpUrl="/onboarding" />
    </div>
  );
}
