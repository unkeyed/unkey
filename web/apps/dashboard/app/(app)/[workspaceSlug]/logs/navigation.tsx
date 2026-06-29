"use client";

import { Navbar } from "@/components/navigation/navbar";
import { useWorkspaceNavigation } from "@/hooks/use-workspace-navigation";
import { routes } from "@/lib/navigation/routes";
import { Layers3 } from "@unkey/icons";

export function Navigation() {
  const workspace = useWorkspaceNavigation();

  return (
    <Navbar>
      <Navbar.Breadcrumbs icon={<Layers3 />}>
        <Navbar.Breadcrumbs.Link href={routes.logs.list({ workspaceSlug: workspace.slug })}>
          Logs
        </Navbar.Breadcrumbs.Link>
      </Navbar.Breadcrumbs>
    </Navbar>
  );
}
