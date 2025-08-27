"use client";

import { Navbar } from "@/components/navigation/navbar";
import { Fingerprint } from "@unkey/icons";

export function Navigation({ workspaceId }: { workspaceId: string }) {
  return (
    <Navbar>
      <Navbar.Breadcrumbs icon={<Fingerprint aria-hidden="true" focusable={false} />}>
        <Navbar.Breadcrumbs.Link href={`/${encodeURIComponent(workspaceId)}/identities`} active>
          Identities
        </Navbar.Breadcrumbs.Link>
      </Navbar.Breadcrumbs>
    </Navbar>
  );
}
