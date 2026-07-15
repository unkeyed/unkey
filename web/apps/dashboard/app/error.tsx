"use client";

import * as Sentry from "@sentry/nextjs";
import { Button, Empty } from "@unkey/ui";
import { useRouter } from "next/navigation";
import { useEffect } from "react";

export default function ErrorPage({
  error,
  reset,
}: {
  error: Error & { digest?: string };
  reset: () => void;
}) {
  const router = useRouter();

  useEffect(() => {
    // Report the error that triggered this boundary. global-error.tsx only
    // catches failures in the root layout, so this is where most route-level
    // render/data errors are captured.
    Sentry.captureException(error);
  }, [error]);

  return (
    <div className="flex items-center justify-center w-full min-h-[60vh] px-4">
      <Empty>
        <Empty.Title>Something went wrong</Empty.Title>
        <Empty.Description>
          An unexpected error occurred while loading this page. Our team has been notified. Try
          again, and if the problem persists, contact support
          {error.digest ? (
            <>
              {" "}
              with reference <span className="font-mono text-gray-12">{error.digest}</span>
            </>
          ) : null}
          .
        </Empty.Description>
        <Empty.Actions>
          <Button variant="primary" onClick={() => reset()}>
            Try again
          </Button>
          <Button variant="outline" onClick={() => router.push("/")}>
            Go to dashboard
          </Button>
        </Empty.Actions>
      </Empty>
    </div>
  );
}
