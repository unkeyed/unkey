"use client";

import { Navbar } from "@/components/navigation/navbar";
import { useWorkspaceNavigation } from "@/hooks/use-workspace-navigation";
import { Gear } from "@unkey/icons";

export function Navigation({ keyId }: { keyId: string }) {
  const workspace = useWorkspaceNavigation();

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
          className="w-[200px] truncate"
          active
          isIdentifier
        >
          {keyId}
        </Navbar.Breadcrumbs.Link>
      </Navbar.Breadcrumbs>
    </Navbar>
  );
}
