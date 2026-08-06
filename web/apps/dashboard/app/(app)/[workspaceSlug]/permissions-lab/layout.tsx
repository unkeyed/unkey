"use client";

import { useWorkspaceNavigation } from "@/hooks/use-workspace-navigation";
import { routes } from "@/lib/navigation/routes";
import { SecondaryNav, SecondaryNavGroup, SecondaryNavItem, SecondaryNavTitle } from "@unkey/ui";
import Link from "next/link";
import { useSelectedLayoutSegments } from "next/navigation";
import type { ReactNode } from "react";
import { CONCEPTS } from "./concepts";
import { PermissionsLabProvider } from "./lib/store";

export default function PermissionsLabLayout({ children }: { children: ReactNode }) {
  const workspace = useWorkspaceNavigation();
  const segments = useSelectedLayoutSegments();
  const active = segments[0] ?? "";
  const scope = { workspaceSlug: workspace.slug };

  return (
    <PermissionsLabProvider>
      <div className="flex flex-col md:flex-row w-full flex-1 min-h-0">
        <SecondaryNav aria-label="Permissions lab">
          <SecondaryNavTitle>Permissions lab</SecondaryNavTitle>
          <SecondaryNavGroup>
            <SecondaryNavItem
              active={active === ""}
              render={<Link href={routes.permissionsLab.index(scope)} />}
            >
              Overview
            </SecondaryNavItem>
            {CONCEPTS.map((concept, i) => (
              <SecondaryNavItem
                key={concept.segment}
                active={active === concept.segment}
                render={<Link href={routes.permissionsLab.concept(scope, concept.segment)} />}
              >
                {`${i + 1}. ${concept.title}`}
              </SecondaryNavItem>
            ))}
          </SecondaryNavGroup>
        </SecondaryNav>
        <div className="flex-1 min-w-0">{children}</div>
      </div>
    </PermissionsLabProvider>
  );
}
