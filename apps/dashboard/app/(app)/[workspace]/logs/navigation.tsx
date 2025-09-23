"use client";

import { Navbar } from "@/components/navigation/navbar";
import { useWorkspaceNavigation } from "@/hooks/use-workspace-navigation";
import { Layers3 } from "@unkey/icons";
import { Loading } from "@unkey/ui";
import { Suspense } from "react";

export function Navigation() {
  const workspace = useWorkspaceNavigation();

  return (
    <Suspense fallback={<Loading type="spinner" />}>
      <Navbar>
        <Navbar.Breadcrumbs icon={<Layers3 />}>
          <Navbar.Breadcrumbs.Link href={`/${workspace.slug}/logs`}>Logs</Navbar.Breadcrumbs.Link>
        </Navbar.Breadcrumbs>
      </Navbar>
    </Suspense>
  );
}
