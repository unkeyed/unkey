"use client";
import { useRouter } from "next/navigation";
import { useEffect } from "react";

export function SuccessClient({ workSpaceSlug }: { workSpaceSlug?: string }) {
  const router = useRouter();

  useEffect(() => {
    if (workSpaceSlug) {
      router.push(`/${workSpaceSlug}/settings/billing`);
    } else {
      // Redirect to root when no workspace slug is available
      // This will typically redirect to workspace selection or onboarding
      router.push("/");
    }
  }, [router, workSpaceSlug]);

  return <></>;
}
