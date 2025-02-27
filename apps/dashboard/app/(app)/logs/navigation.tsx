"use client";

import { Navbar } from "@/components/navbar";
import { Layers3 } from "@unkey/icons";

export function Navigation() {
  return (
    <Navbar>
      <Navbar.Breadcrumbs icon={<Layers3 />}>
          <Navbar.Breadcrumbs.Link href="/logs">Logs</Navbar.Breadcrumbs.Link>
        </Navbar.Breadcrumbs>
    </Navbar>
  );
}