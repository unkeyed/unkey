"use client";

import { Navbar } from "@/components/navbar";
import { Gauge } from "@unkey/icons";
import { CreateNamespaceButton } from "./_components/create-namespace-button";
export function Navigation() {
  return (
    <Navbar>
      <Navbar.Breadcrumbs icon={<Gauge />}>
        <Navbar.Breadcrumbs.Link href="/ratelimits">Ratelimits</Navbar.Breadcrumbs.Link>
      </Navbar.Breadcrumbs>
      <Navbar.Actions>
        <CreateNamespaceButton />
      </Navbar.Actions>
    </Navbar>
  );
}
