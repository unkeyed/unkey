"use client";
import { Navbar } from "@/components/navigation/navbar";
import { ShieldKey } from "@unkey/icons";

export function Navigation() {
  return (
    <Navbar>
      <Navbar.Breadcrumbs icon={<ShieldKey />}>
        <Navbar.Breadcrumbs.Link href="/authorization/roles">Authorization</Navbar.Breadcrumbs.Link>
        <Navbar.Breadcrumbs.Link href="/authorization/permissions" active>
          Roles
        </Navbar.Breadcrumbs.Link>
      </Navbar.Breadcrumbs>
    </Navbar>
  );
}
