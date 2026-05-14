"use client";

import * as Sentry from "@sentry/nextjs";
import { Button, Empty } from "@unkey/ui";
import { useEffect } from "react";

/**
 * Error boundary for the authenticated app shell. Render errors that escape a
 * page (failed dynamic imports, runtime exceptions during hydration, etc.)
 * land here. Without this, errors here would bubble all the way up to
 * `global-error.tsx` and the user would lose the sidebar/chrome.
 */
export default function AppError({
  error,
  reset,
}: {
  error: Error & { digest?: string };
  reset: () => void;
}) {
  useEffect(() => {
    Sentry.captureException(error, {
      tags: { boundary: "app_authenticated" },
      extra: { digest: error.digest },
    });
  }, [error]);

  return (
    <div className="flex items-center justify-center w-full min-h-[60vh] px-4">
      <Empty>
        <Empty.Title>Something went wrong</Empty.Title>
        <Empty.Description>
          Our team has been notified. You can try again, or refresh the page. If this error
          continues you can reach out to{" "}
          <a href="mailto:support@unkey.com" className="underline">
            support@unkey.com
          </a>
          .
        </Empty.Description>
        <Empty.Actions>
          <Button variant="default" onClick={reset}>
            Try again
          </Button>
        </Empty.Actions>
      </Empty>
    </div>
  );
}
