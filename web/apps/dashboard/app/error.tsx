"use client";

import * as Sentry from "@sentry/nextjs";
import { useEffect } from "react";

/**
 * Root-level Next.js error boundary. Without this file, render errors thrown
 * inside `app/layout.tsx`'s children fall through to `global-error.tsx` only
 * after wrecking the whole tree — and many transient errors (failed dynamic
 * imports, hydration failures inside a segment) never reach Sentry at all.
 *
 * `global-error.tsx` already handles the layout-level case; this one captures
 * everything below the root layout so Sentry sees the failure and the user
 * can recover via reset() without a hard reload.
 */
export default function RootError({
  error,
  reset,
}: {
  error: Error & { digest?: string };
  reset: () => void;
}) {
  useEffect(() => {
    Sentry.captureException(error, {
      tags: { boundary: "app_root" },
      extra: { digest: error.digest },
    });
  }, [error]);

  return (
    <div
      style={{
        display: "flex",
        flexDirection: "column",
        alignItems: "center",
        justifyContent: "center",
        minHeight: "60vh",
        padding: "2rem",
        textAlign: "center",
      }}
    >
      <h1 style={{ fontSize: "1.5rem", fontWeight: 600, marginBottom: "0.5rem" }}>
        Something went wrong
      </h1>
      <p style={{ marginBottom: "1.5rem", opacity: 0.7 }}>
        Our team has been notified. You can try again, or refresh the page. If this error continues
        you can reach out to{" "}
        <a href="mailto:support@unkey.com" style={{ textDecoration: "underline" }}>
          support@unkey.com
        </a>
        .
      </p>
      <button
        type="button"
        onClick={reset}
        style={{
          padding: "0.5rem 1rem",
          borderRadius: "0.375rem",
          border: "1px solid currentColor",
          background: "transparent",
          cursor: "pointer",
        }}
      >
        Try again
      </button>
    </div>
  );
}
