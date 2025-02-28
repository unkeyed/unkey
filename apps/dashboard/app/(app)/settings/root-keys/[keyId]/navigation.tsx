"use client";

import { Navbar } from "@/components/navigation/navbar";
import { Gear } from "@unkey/icons";

export function Navigation({ keyId }: { keyId: string }) {
  return (
    <Navbar>
      <Navbar.Breadcrumbs icon={<Gear />}>
        <Navbar.Breadcrumbs.Link href="/settings/root-keys">Root Keys</Navbar.Breadcrumbs.Link>
        <Navbar.Breadcrumbs.Link
          href={`/settings/root-keys/${keyId}`}
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
