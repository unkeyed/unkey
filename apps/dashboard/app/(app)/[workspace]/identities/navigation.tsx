"use client";

import { Navbar } from "@/components/navigation/navbar";
import { useWorkspace } from "@/providers/workspace-provider";
import { Fingerprint } from "@unkey/icons";

export function Navigation() {
  const { workspace } = useWorkspace();

  if (!workspace) {
    return null;
  }

  return (
    <Navbar>
      <Navbar.Breadcrumbs icon={<Fingerprint aria-hidden="true" focusable={false} />}>
        <Navbar.Breadcrumbs.Link href={`/${encodeURIComponent(workspace.slug)}/identities`} active>
          Identities
        </Navbar.Breadcrumbs.Link>
      </Navbar.Breadcrumbs>
    </Navbar>
  );
}
