"use client";

import { Navbar } from "@/components/navigation/navbar";
import { useWorkspace } from "@/providers/workspace-provider";
import { Layers3 } from "@unkey/icons";

export function Navigation() {
  const { workspace } = useWorkspace();

  return (
    workspace && (
      <Navbar>
        <Navbar.Breadcrumbs icon={<Layers3 />}>
          <Navbar.Breadcrumbs.Link href={`/${workspace?.slug}/logs`}>Logs</Navbar.Breadcrumbs.Link>
        </Navbar.Breadcrumbs>
      </Navbar>
    )
  );
}
