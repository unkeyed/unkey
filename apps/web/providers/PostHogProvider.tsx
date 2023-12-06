// app/providers.tsx
"use client";
import { UserResource } from "@clerk/types";
import { usePathname, useSearchParams } from "next/navigation";
import posthog from "posthog-js";
import { PostHogProvider } from "posthog-js/react";
import { useEffect } from "react";

if (typeof window !== "undefined") {
  posthog.init(process.env.NEXT_PUBLIC_POSTHOG_KEY!, {
    api_host: process.env.NEXT_PUBLIC_POSTHOG_HOST || "https://app.posthog.com",
    capture_pageview: false,
    disable_session_recording: false,
  });
}

export function PostHogPageview(): JSX.Element {
  const pathname = usePathname();
  const searchParams = useSearchParams();
  useEffect(() => {
    if (pathname) {
      let url = window.origin + pathname;
      if (searchParams?.toString()) {
        url = url + `?${searchParams.toString()}`;
      }
      posthog.capture("$pageview", {
        $current_url: url,
      });
    }
  }, [pathname, searchParams]);

  return <></>;
}

export const PostHogIdentify = ({ user }: { user: UserResource }) => {
  posthog.identify(user.id, {
    email: user.primaryEmailAddress?.emailAddress,
    userName: user.username,
    firstName: user.firstName,
    lastName: user.lastName,
    createdAt: user.createdAt?.toISOString(),
  });
};

/*
 * This function can be used to send events to PostHog from anywhere in the app.
 * Example:
 * import { PostHogEvent } from "@/providers/PostHogProvider";
 * PostHogEvent({ name: "plan_upgrade", properties: { plan: "pro"} })
 *
 */

export const PostHogEvent = ({
  name,
  properties,
}: { name: string; properties?: Record<string, any> }) => {
  posthog.capture(name, properties);
};

export function PHProvider({ children }: { children: React.ReactNode }) {
  return <PostHogProvider client={posthog}>{children}</PostHogProvider>;
}
