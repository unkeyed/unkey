"use client";

import { Navbar } from "@/components/navigation/navbar";
import { useWorkspaceWithRedirect } from "@/hooks/use-workspace-with-redirect";
import { Fingerprint } from "@unkey/icons";

export function Navigation() {
  const { workspace } = useWorkspaceWithRedirect();

  return (
    <Navbar>
      <Navbar.Breadcrumbs icon={<Fingerprint aria-hidden="true" focusable={false} />}>
        <Navbar.Breadcrumbs.Link href={`/${workspace.slug}/identities`} active>
          Identities
        </Navbar.Breadcrumbs.Link>
      </Navbar.Breadcrumbs>
    </Navbar>
  );
}
