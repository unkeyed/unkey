"use client";
import posthog from "posthog-js";
import { PostHogProvider } from "posthog-js/react";

if (typeof window !== "undefined") {
  posthog.init(process.env.NEXT_PUBLIC_POSTHOG_KEY!, {
    api_host: process.env.NEXT_PUBLIC_POSTHOG_HOST,
    capture_pageview: false,
    disable_session_recording: true,
  });
}

export function PHProvider({
  children,
}: {
  children: React.ReactNode;
}) {
  return <PostHogProvider client={posthog}>{children}</PostHogProvider>;
}
