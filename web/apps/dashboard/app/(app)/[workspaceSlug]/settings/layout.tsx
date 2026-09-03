"use client";

import { useWorkspaceNavigation } from "@/hooks/use-workspace-navigation";
import { useFlag } from "@/lib/flags/provider";
import { useBillingUIUpgrades } from "@/lib/flags/use-billing-ui-upgrades";
import { routes } from "@/lib/navigation/routes";
import { trpc } from "@/lib/trpc/client";
import { SecondaryNav, SecondaryNavGroup, SecondaryNavItem, SecondaryNavTitle } from "@unkey/ui";
import Link from "next/link";
import { useSelectedLayoutSegments } from "next/navigation";
import type { ReactNode } from "react";

const ITEMS = [
  { segment: "general", label: "General", getHref: routes.settings.general },
  { segment: "team", label: "Team", getHref: routes.settings.team },
  { segment: "root-keys", label: "Root Keys", getHref: routes.settings.rootKeys },
  { segment: "logdrains", label: "Log Drains", getHref: routes.settings.logdrains.list },
  { segment: "billing", label: "Billing", getHref: routes.settings.billing },
  { segment: "account", label: "Account", getHref: routes.settings.account },
  { segment: "usage", label: "Usage", getHref: routes.settings.usage },
  { segment: "limits", label: "Limits", getHref: routes.settings.limits },
] as const;

const BILLING_UPGRADE_SEGMENTS: ReadonlySet<string> = new Set(["usage", "limits"]);

export default function SettingsLayout({ children }: { children: ReactNode }) {
  const workspace = useWorkspaceNavigation();
  const segments = useSelectedLayoutSegments();
  const active = segments[0] ?? "general";
  const billingUpgrades = useBillingUIUpgrades();
  const logdrainsEnabled = useFlag("logdrains");
  const { data: currentUser } = trpc.user.getCurrentUser.useQuery();
  const items = ITEMS.filter(
    (item) => billingUpgrades || !BILLING_UPGRADE_SEGMENTS.has(item.segment),
  )
    .filter((item) => logdrainsEnabled || item.segment !== "logdrains")
    .filter((item) => currentUser?.role === "admin" || item.segment !== "root-keys");
  const isLogdrainCreation = segments[0] === "logdrains" && segments[1] === "new";

  if (isLogdrainCreation) {
    return <div className="flex-1 min-w-0">{children}</div>;
  }

  return (
    <div className="flex flex-col md:flex-row w-full flex-1 min-h-0">
      <SecondaryNav aria-label="Settings">
        <SecondaryNavTitle>Settings</SecondaryNavTitle>
        <SecondaryNavGroup>
          {items.map((item) => (
            <SecondaryNavItem
              key={item.segment}
              active={active === item.segment}
              render={<Link href={item.getHref({ workspaceSlug: workspace.slug })} />}
            >
              {item.label}
            </SecondaryNavItem>
          ))}
        </SecondaryNavGroup>
      </SecondaryNav>
      <div className="flex-1 min-w-0">{children}</div>
    </div>
  );
}
