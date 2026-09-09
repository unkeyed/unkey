"use client";

import { match } from "@unkey/match";
import { AlertBanner, AlertBannerDescription, AlertBannerTitle } from "@unkey/ui";
import type { Route } from "next";
import Link from "next/link";
import type { ReactNode } from "react";
import { SUPPORT_MAILTO } from "@/lib/support";
import type { BreachKey } from "./limit-groups";

export function BreachBanner({
  breached,
  billingHref,
}: {
  breached: BreachKey[];
  billingHref: Route;
}) {
  return match(breached)
    .when(
      (keys) => keys.includes("domains"),
      () => (
        <Banner>
          Your workspace is at its custom domain limit. Remove a domain that you do not need, or{" "}
          <BannerLink href={billingHref}>upgrade your plan</BannerLink>.
        </Banner>
      ),
    )
    .when(
      (keys) => keys.includes("compute") && keys.includes("api"),
      () => (
        <Banner>
          Scale down or remove a deployment to free compute capacity. To raise your API operations
          limit, <BannerLink href={billingHref}>upgrade your plan</BannerLink>.
        </Banner>
      ),
    )
    .when(
      (keys) => keys.includes("compute"),
      () => (
        <Banner>
          Scale down or remove a deployment to free capacity. To request higher capacity limits,{" "}
          <BannerLink href={SUPPORT_MAILTO}>contact us</BannerLink>.
        </Banner>
      ),
    )
    .otherwise(() => (
      <Banner>
        You're over your plan's allowed usage. To continue,{" "}
        <BannerLink href={billingHref}>upgrade your plan</BannerLink>.
      </Banner>
    ));
}

function Banner({ children }: { children: ReactNode }) {
  return (
    <AlertBanner variant="error">
      <AlertBannerTitle>You've reached a limit</AlertBannerTitle>
      <AlertBannerDescription>{children}</AlertBannerDescription>
    </AlertBanner>
  );
}

function BannerLink({ href, children }: { href: Route; children: ReactNode }) {
  return (
    <Link href={href} className="underline underline-offset-2 hover:opacity-80">
      {children}
    </Link>
  );
}
