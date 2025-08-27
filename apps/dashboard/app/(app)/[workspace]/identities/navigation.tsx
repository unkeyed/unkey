"use client";

import { Navbar } from "@/components/navigation/navbar";
import { Fingerprint } from "@unkey/icons";

export function Navigation({ workspaceId }: { workspaceId: string }) {
  return (
    <Navbar>
      <Navbar.Breadcrumbs icon={<Fingerprint />}>
        <Navbar.Breadcrumbs.Link href={`/${workspaceId}/identities`} active>
          Identities
        </Navbar.Breadcrumbs.Link>
      </Navbar.Breadcrumbs>
    </Navbar>
  );
}
