"use client";
import { useRouter } from "next/navigation";
import { useEffect } from "react";

export function SuccessClient({ workSpaceSlug }: { workSpaceSlug?: string }) {
  const router = useRouter();

  useEffect(() => {
    if (workSpaceSlug) {
      router.push(`/${workSpaceSlug}/settings/billing`);
    } else {
      router.push("/apis");
    }
  }, [router, workSpaceSlug]);

  return <></>;
}
