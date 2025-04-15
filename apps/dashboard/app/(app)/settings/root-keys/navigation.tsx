"use client";

import { Navbar } from "@/components/navigation/navbar";
import { Gear } from "@unkey/icons";
import { Button } from "@unkey/ui";
import Link from "next/link";

export function Navigation() {
  return (
    <Navbar>
      <Navbar.Breadcrumbs icon={<Gear />}>
        <Navbar.Breadcrumbs.Link href="/settings/general">
          Settings
        </Navbar.Breadcrumbs.Link>
        <Navbar.Breadcrumbs.Link href="/settings/root-keys" active>
          Root Keys
        </Navbar.Breadcrumbs.Link>
      </Navbar.Breadcrumbs>
      <Navbar.Actions>
        <Link key="create-root-key" href="/settings/root-keys/new">
          <Button variant="primary">Create New Root Key</Button>
        </Link>
      </Navbar.Actions>
    </Navbar>
  );
}
