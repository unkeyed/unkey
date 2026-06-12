"use client";

import { useWorkspaceNavigation } from "@/hooks/use-workspace-navigation";
import { SecondaryNav, SecondaryNavGroup, SecondaryNavItem, SecondaryNavTitle } from "@unkey/ui";
import Link from "next/link";
import { useSelectedLayoutSegments } from "next/navigation";
import type { ReactNode } from "react";

const ITEMS = [
  { segment: "roles", label: "Roles" },
  { segment: "permissions", label: "Permissions" },
] as const;

export default function AuthorizationLayout({ children }: { children: ReactNode }) {
  const workspace = useWorkspaceNavigation();
  const segments = useSelectedLayoutSegments();
  const active = segments[0] ?? "roles";

  return (
    <div className="flex flex-col md:flex-row w-full flex-1 min-h-0">
      <SecondaryNav aria-label="Authorization">
        <SecondaryNavTitle>Authorization</SecondaryNavTitle>
        <SecondaryNavGroup>
          {ITEMS.map((item) => (
            <SecondaryNavItem key={item.segment} asChild active={active === item.segment}>
              <Link href={`/${workspace.slug}/authorization/${item.segment}`}>{item.label}</Link>
            </SecondaryNavItem>
          ))}
        </SecondaryNavGroup>
      </SecondaryNav>
      <div className="flex-1 min-w-0">{children}</div>
    </div>
  );
}
