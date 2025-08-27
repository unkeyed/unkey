"use client";

import { Navbar } from "@/components/navigation/navbar";
import { Gear } from "@unkey/icons";

export function Navigation({ keyId, workspaceSlug }: { keyId: string; workspaceSlug: string }) {
  return (
    <Navbar>
      <Navbar.Breadcrumbs icon={<Gear />}>
        <Navbar.Breadcrumbs.Link href={`/${workspaceSlug}/settings`}>
          Settings
        </Navbar.Breadcrumbs.Link>
        <Navbar.Breadcrumbs.Link href={`/${workspaceSlug}/settings/root-keys`}>
          Root Keys
        </Navbar.Breadcrumbs.Link>
        <Navbar.Breadcrumbs.Link
          href={`/${workspaceSlug}/settings/root-keys/${keyId}`}
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
