"use client";

import { Navbar } from "@/components/navigation/navbar";
import { Fingerprint } from "@unkey/icons";

type NavigationProps = {
  identityId: string;
};

export function Navigation({ identityId }: NavigationProps) {
  return (
    <Navbar>
      <Navbar.Breadcrumbs icon={<Fingerprint />}>
        <Navbar.Breadcrumbs.Link href="/identities">Identities</Navbar.Breadcrumbs.Link>
        <Navbar.Breadcrumbs.Link
          href={`/identities/${identityId}`}
          className="w-[200px] truncate"
          active
          isIdentifier
        >
          {identityId}
        </Navbar.Breadcrumbs.Link>
      </Navbar.Breadcrumbs>
    </Navbar>
  );
}
