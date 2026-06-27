"use client";

import { useWorkspaceNavigation } from "@/hooks/use-workspace-navigation";
import { SecondaryNav, SecondaryNavGroup, SecondaryNavItem, SecondaryNavTitle } from "@unkey/ui";
import Link from "next/link";
import { useSelectedLayoutSegments } from "next/navigation";
import type { ReactNode } from "react";
import { navigation } from "./constants";

export default function AuthorizationLayout({ children }: { children: ReactNode }) {
  const workspace = useWorkspaceNavigation();
  const segments = useSelectedLayoutSegments();
  const active = segments[0] ?? "roles";
  const items = navigation(workspace.slug);

  return (
    <div className="flex flex-col md:flex-row w-full flex-1 min-h-0">
      <SecondaryNav aria-label="Authorization">
        <SecondaryNavTitle>Authorization</SecondaryNavTitle>
        <SecondaryNavGroup>
          {items.map((item) => (
            <SecondaryNavItem key={item.segment} asChild active={active === item.segment}>
              <Link href={item.href}>{item.label}</Link>
            </SecondaryNavItem>
          ))}
        </SecondaryNavGroup>
      </SecondaryNav>
      <div className="flex-1 min-w-0">{children}</div>
    </div>
  );
}
