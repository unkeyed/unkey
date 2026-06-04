"use client";

import { useWorkspaceNavigation } from "@/hooks/use-workspace-navigation";
import { useFlag } from "@/lib/flags/provider";
import { SecondaryNav, SecondaryNavGroup, SecondaryNavItem, SecondaryNavTitle } from "@unkey/ui";
import Link from "next/link";
import { useSelectedLayoutSegments } from "next/navigation";
import type { ReactNode } from "react";

const ITEMS = [
  { segment: "general", label: "General" },
  { segment: "team", label: "Team" },
  { segment: "root-keys", label: "Root Keys" },
  { segment: "billing", label: "Billing" },
] as const;

export default function SettingsLayout({ children }: { children: ReactNode }) {
  const newNavigation = useFlag("newNavigation");
  const workspace = useWorkspaceNavigation();
  const segments = useSelectedLayoutSegments();
  const active = segments[0] ?? "general";

  // The redesigned section sidebar only exists in the new shell. With the flag
  // off, pages render their own legacy Navbar and the layout is a passthrough.
  if (!newNavigation) {
    return <>{children}</>;
  }

  return (
    <div className="flex flex-col md:flex-row w-full flex-1 min-h-0">
      <SecondaryNav aria-label="Settings">
        <SecondaryNavTitle>Settings</SecondaryNavTitle>
        <SecondaryNavGroup>
          {ITEMS.map((item) => (
            <SecondaryNavItem key={item.segment} asChild active={active === item.segment}>
              <Link href={`/${workspace.slug}/settings/${item.segment}`}>{item.label}</Link>
            </SecondaryNavItem>
          ))}
        </SecondaryNavGroup>
      </SecondaryNav>
      <div className="flex-1 min-w-0">{children}</div>
    </div>
  );
}
