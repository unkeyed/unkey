"use client";

import { useWorkspaceNavigation } from "@/hooks/use-workspace-navigation";
import { useBillingUIUpgrades } from "@/lib/flags/use-billing-ui-upgrades";
import { routes } from "@/lib/navigation/routes";
import { SecondaryNav, SecondaryNavGroup, SecondaryNavItem, SecondaryNavTitle } from "@unkey/ui";
import Link from "next/link";
import { useSelectedLayoutSegments } from "next/navigation";
import type { ReactNode } from "react";

const ITEMS = [
  { segment: "general", label: "General", getHref: routes.settings.general },
  { segment: "team", label: "Team", getHref: routes.settings.team },
  { segment: "root-keys", label: "Root Keys", getHref: routes.settings.rootKeys },
  { segment: "billing", label: "Billing", getHref: routes.settings.billing },
  { segment: "account", label: "Account", getHref: routes.settings.account },
  { segment: "usage", label: "Usage", getHref: routes.settings.usage },
  { segment: "limits", label: "Limits", getHref: routes.settings.limits },
  { segment: "security", label: "Security", getHref: routes.settings.security },
] as const;

const BILLING_UPGRADE_SEGMENTS: ReadonlySet<string> = new Set(["usage", "limits"]);

export default function SettingsLayout({ children }: { children: ReactNode }) {
  const workspace = useWorkspaceNavigation();
  const segments = useSelectedLayoutSegments();
  const active = segments[0] ?? "general";
  const billingUpgrades = useBillingUIUpgrades();
  const items = ITEMS.filter(
    (item) => billingUpgrades || !BILLING_UPGRADE_SEGMENTS.has(item.segment),
  );

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
