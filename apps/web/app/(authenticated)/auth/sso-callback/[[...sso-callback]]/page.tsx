"use client";

import { AuthenticateWithRedirectCallback } from "@clerk/nextjs";
import { Loading } from "@/components/dashboard/loading";

export const runtime = "edge";

export default function SSOCallback() {
  return (
    <div className="flex items-center justify-center h-screen ">
      <Loading className="w-16 h-16" />
      <AuthenticateWithRedirectCallback afterSignInUrl="/app/apis" afterSignUpUrl="/onboarding" />
    </div>
  );
}
