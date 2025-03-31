"use client";

import { Navbar } from "@/components/navigation/navbar";
import { InputSearch } from "@unkey/icons";

export function Navigation() {
  return (
    <Navbar>
      <Navbar.Breadcrumbs icon={<InputSearch />}>
        <Navbar.Breadcrumbs.Link href="/audit">Audit</Navbar.Breadcrumbs.Link>
      </Navbar.Breadcrumbs>
    </Navbar>
  );
}
