"use client";

import { Navbar } from "@/components/navigation/navbar";
import { InputSearch } from "@unkey/icons";

export function Navigation({ workspaceId }: { workspaceId: string }) {
  return (
    <Navbar>
      <Navbar.Breadcrumbs icon={<InputSearch />}>
        <Navbar.Breadcrumbs.Link href={`/${workspaceId}/audit`}>Audit</Navbar.Breadcrumbs.Link>
      </Navbar.Breadcrumbs>
    </Navbar>
  );
}
