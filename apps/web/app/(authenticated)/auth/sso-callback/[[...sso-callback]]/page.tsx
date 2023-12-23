"use client";

import { Loading } from "@/components/dashboard/loading";
import { AuthenticateWithRedirectCallback } from "@clerk/nextjs";

export default function SSOCallback() {
  return (
    <div className="flex h-screen items-center justify-center ">
      <Loading className="h-12 w-12" />
      <AuthenticateWithRedirectCallback afterSignInUrl="/app/apis" afterSignUpUrl="/new" />
    </div>
  );
}
