"use client";

import { SUPPORT_MAILTO } from "@/lib/support";
import { AlertBanner, AlertBannerDescription, AlertBannerTitle } from "@unkey/ui";
import type { Route } from "next";
import Link from "next/link";
import type { ReactNode } from "react";
import type { GroupKey } from "./limit-groups";

export function BreachBanner({
  breached,
  billingHref,
}: {
  breached: GroupKey[];
  billingHref: Route;
}) {
  const compute = breached.includes("compute");
  const api = breached.includes("api");
  const domains = breached.includes("domains");

  if (domains && !compute && !api) {
    return (
      <Banner>
        Your workspace is at its custom domain limit. Remove a domain that you do not need, or{" "}
        <BannerLink href={billingHref}>upgrade your plan</BannerLink>.
      </Banner>
    );
  }

  if (compute && api) {
    return (
      <Banner>
        Scale down or remove a deployment to free compute capacity. To raise your API operations
        limit, <BannerLink href={billingHref}>upgrade your plan</BannerLink>.
      </Banner>
    );
  }
  if (compute) {
    return (
      <Banner>
        Scale down or remove a deployment to free capacity. To request higher capacity limits,{" "}
        <BannerLink href={SUPPORT_MAILTO}>contact us</BannerLink>.
      </Banner>
    );
  }
  return (
    <Banner>
      You're over your plan's allowed usage. To continue,{" "}
      <BannerLink href={billingHref}>upgrade your plan</BannerLink>.
    </Banner>
  );
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
