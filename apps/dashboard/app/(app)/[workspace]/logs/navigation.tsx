"use client";

import { Navbar } from "@/components/navigation/navbar";
import { Layers3 } from "@unkey/icons";

export function Navigation({ workspaceId }: { workspaceId: string }) {
  return (
    <Navbar>
      <Navbar.Breadcrumbs icon={<Layers3 />}>
        <Navbar.Breadcrumbs.Link href={`/${workspaceId}/logs`}>Logs</Navbar.Breadcrumbs.Link>
      </Navbar.Breadcrumbs>
    </Navbar>
  );
}
