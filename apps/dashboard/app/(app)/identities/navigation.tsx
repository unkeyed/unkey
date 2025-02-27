"use client";

import { Navbar } from "@/components/navbar";
import { Fingerprint } from "@unkey/icons";

export function Navigation() {
  return (
    <Navbar>
      <Navbar.Breadcrumbs icon={<Fingerprint />}>
        <Navbar.Breadcrumbs.Link href="/identities" active>
          Identities
        </Navbar.Breadcrumbs.Link>
      </Navbar.Breadcrumbs>
    </Navbar>
  );
}
