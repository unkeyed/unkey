"use client";

import { Navbar } from "@/components/navigation/navbar";
import { useWorkspaceNavigation } from "@/hooks/use-workspace-navigation";
import { routes } from "@/lib/navigation/routes";
import { Coins } from "@unkey/icons";

export function Navigation() {
  const workspace = useWorkspaceNavigation();

  return (
    <Navbar>
      <Navbar.Breadcrumbs icon={<Coins aria-hidden="true" focusable={false} />}>
        <Navbar.Breadcrumbs.Link
          href={routes.monetization.overview({ workspaceSlug: workspace.slug })}
          active
        >
          Monetization
        </Navbar.Breadcrumbs.Link>
      </Navbar.Breadcrumbs>
    </Navbar>
  );
}
