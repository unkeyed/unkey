"use client";

import { Loading } from "@/components/dashboard/loading";
import { AuthenticateWithRedirectCallback } from "@clerk/nextjs";

export const runtime = "edge";

export default function SSOCallback() {
  return (
    <div className="flex items-center justify-center h-screen ">
      <Loading className="w-16 h-16" />
      <AuthenticateWithRedirectCallback afterSignInUrl="/app/apis" afterSignUpUrl="/onboarding" />
    </div>
  );
}
