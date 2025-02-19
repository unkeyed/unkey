"use client";

import { useConsentManager } from "@c15t/react";
import { usePathname, useSearchParams } from "next/navigation";
import { usePostHog } from "posthog-js/react";
import { useEffect } from "react";

export default function PostHogPageView(): null {
  const pathname = usePathname();
  const searchParams = useSearchParams();
  const { consents } = useConsentManager();
  const posthog = usePostHog();

  // callbacks.onConsentChange((consent) => {
  //   if (consent.measurement) {
  //     posthog.capture("$pageleave");
  //   }
  // });
  useEffect(() => {
    if (pathname && posthog && consents.measurement) {
      posthog.capture("$pageleave");
      let url = window.origin + pathname;
      if (searchParams.toString()) {
        url = `${url}?${searchParams.toString()}`;
      }
      posthog.capture("$pageview", {
        $current_url: url,
      });
    }
  }, [pathname, searchParams, consents.measurement, posthog]);

  return null;
}
