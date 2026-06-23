"use client";

import { useWorkspaceNavigation } from "@/hooks/use-workspace-navigation";
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
  { segment: "security", label: "Security", getHref: routes.settings.security },
] as const;

export default function SettingsLayout({ children }: { children: ReactNode }) {
  const workspace = useWorkspaceNavigation();
  const segments = useSelectedLayoutSegments();
  const active = segments[0] ?? "general";

  return (
    <div className="flex flex-col md:flex-row w-full flex-1 min-h-0">
      <SecondaryNav aria-label="Settings">
        <SecondaryNavTitle>Settings</SecondaryNavTitle>
        <SecondaryNavGroup>
          {ITEMS.map((item) => (
            <SecondaryNavItem key={item.segment} asChild active={active === item.segment}>
              <Link href={item.getHref({ workspaceSlug: workspace.slug })}>{item.label}</Link>
            </SecondaryNavItem>
          ))}
        </SecondaryNavGroup>
      </SecondaryNav>
      <div className="flex-1 min-w-0">{children}</div>
    </div>
  );
}
