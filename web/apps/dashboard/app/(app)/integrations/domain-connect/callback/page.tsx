"use client";

import { isSafeRedirectPath } from "@/app/auth/sign-in/redirect-utils";
import { LoadingState } from "@/components/loading-state";
import { Empty } from "@unkey/ui";
import { useRouter, useSearchParams } from "next/navigation";
import { useEffect } from "react";

// Public callback that DNS providers redirect to after a Domain Connect approval.
// Lives at a public path so the cross-site return navigation isn't blocked when
// the SameSite=Strict session cookie is dropped. The same-site router.replace
// below restores the cookie on the next request.
export default function Page() {
  const router = useRouter();
  const searchParams = useSearchParams();
  const to = searchParams?.get("to") ?? null;

  useEffect(() => {
    if (to && isSafeRedirectPath(to)) {
      router.replace(to);
    }
  }, [router, to]);

  if (!to || !isSafeRedirectPath(to)) {
    return (
      <div className="w-full min-h-[60vh] flex justify-center items-center">
        <Empty>
          <Empty.Title>Invalid callback target</Empty.Title>
          <Empty.Description>Missing or invalid redirect destination.</Empty.Description>
        </Empty>
      </div>
    );
  }

  return <LoadingState message="Finishing domain setup..." />;
}
