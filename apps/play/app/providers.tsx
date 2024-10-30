"use client";
import posthog from "posthog-js";
import { PostHogProvider } from "posthog-js/react";

if (typeof window !== "undefined") {
  posthog.init(process.env.NEXT_PUBLIC_POSTHOG_KEY!, {
    api_host: process.env.NEXT_PUBLIC_POSTHOG_HOST,
  });
}
export function CSPostHogProvider({
  children,
}: Readonly<{
  children: React.ReactNode;
}>) {
  return <PostHogProvider client={posthog}> {children} </PostHogProvider>;
}
