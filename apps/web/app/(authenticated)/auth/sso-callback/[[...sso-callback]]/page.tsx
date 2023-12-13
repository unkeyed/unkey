"use client";

import { Loading } from "@/components/dashboard/loading";
import { AuthenticateWithRedirectCallback } from "@clerk/nextjs";

export default function SSOCallback() {
  return (
    <div className="flex h-screen items-center justify-center ">
      <Loading className="h-32 w-32" />
      <AuthenticateWithRedirectCallback afterSignInUrl="/app/apis" afterSignUpUrl="/new" />
    </div>
  );
}
