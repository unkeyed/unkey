"use client";

import { Navbar } from "@/components/navbar";
import { Gear } from "@unkey/icons";

// Reusable for settings where we only change the link
export function Navigation() {
  return (
    <Navbar>
      <Navbar.Breadcrumbs icon={<Gear />}>
        <Navbar.Breadcrumbs.Link href="/settings/billing">Billing</Navbar.Breadcrumbs.Link>
        <Navbar.Breadcrumbs.Link href="/settings/billing/plans" active>
          Plans
        </Navbar.Breadcrumbs.Link>
      </Navbar.Breadcrumbs>
    </Navbar>
  );
}
