"use client";

import { Navbar } from "@/components/navigation/navbar";
import { Gear } from "@unkey/icons";

export function Navigation() {
  return (
    <Navbar>
      <Navbar.Breadcrumbs icon={<Gear />}>
        <Navbar.Breadcrumbs.Link href="/settings">Settings</Navbar.Breadcrumbs.Link>
        <Navbar.Breadcrumbs.Link href="/settings/root-keys">Root Keys</Navbar.Breadcrumbs.Link>
        <Navbar.Breadcrumbs.Link href="/settings/root-keys/new">New</Navbar.Breadcrumbs.Link>
      </Navbar.Breadcrumbs>
    </Navbar>
  );
}
