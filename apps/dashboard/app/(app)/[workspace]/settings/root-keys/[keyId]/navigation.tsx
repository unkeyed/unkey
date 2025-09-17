"use client";

import { Navbar } from "@/components/navigation/navbar";
import { useWorkspaceWithRedirect } from "@/hooks/use-workspace-with-redirect";
import { Gear } from "@unkey/icons";

export function Navigation({ keyId }: { keyId: string }) {
  const { workspace } = useWorkspaceWithRedirect();

  return (
    <Navbar>
      <Navbar.Breadcrumbs icon={<Gear />}>
        <Navbar.Breadcrumbs.Link href={`/${workspace.slug}/settings`}>
          Settings
        </Navbar.Breadcrumbs.Link>
        <Navbar.Breadcrumbs.Link href={`/${workspace.slug}/settings/root-keys`}>
          Root Keys
        </Navbar.Breadcrumbs.Link>
        <Navbar.Breadcrumbs.Link
          href={`/${workspace.slug}/settings/root-keys/${keyId}`}
          active
          isIdentifier
          className="w-[100px] truncate"
        >
          {keyId}
        </Navbar.Breadcrumbs.Link>
      </Navbar.Breadcrumbs>
    </Navbar>
  );
}
