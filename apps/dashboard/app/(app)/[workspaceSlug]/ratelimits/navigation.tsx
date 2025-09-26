"use client";

import { Navbar } from "@/components/navigation/navbar";
import { useWorkspaceNavigation } from "@/hooks/use-workspace-navigation";
import { Gauge } from "@unkey/icons";
import { CreateNamespaceButton } from "./_components/create-namespace-button";

export function Navigation() {
  const workspace = useWorkspaceNavigation();
  return (
    <Navbar>
      <Navbar.Breadcrumbs icon={<Gauge />}>
        <Navbar.Breadcrumbs.Link href={`/${workspace.slug}/ratelimits`} active>
          Ratelimits
        </Navbar.Breadcrumbs.Link>
      </Navbar.Breadcrumbs>
      <Navbar.Actions>
        <CreateNamespaceButton />
      </Navbar.Actions>
    </Navbar>
  );
}
