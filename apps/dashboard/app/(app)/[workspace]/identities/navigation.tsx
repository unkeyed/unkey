"use client";

import { Navbar } from "@/components/navigation/navbar";
import { useWorkspaceNavigation } from "@/hooks/use-workspace-navigation";
import { Fingerprint } from "@unkey/icons";
import { redirect } from "next/navigation";
import { Loading } from "@unkey/ui";

export function Navigation() {
  const workspace = useWorkspaceNavigation();

  return (
    <Navbar>
      <Navbar.Breadcrumbs
        icon={<Fingerprint aria-hidden="true" focusable={false} />}
      >
        <Navbar.Breadcrumbs.Link href={`/${workspace.slug}/identities`} active>
          Identities
        </Navbar.Breadcrumbs.Link>
      </Navbar.Breadcrumbs>
    </Navbar>
  );
}
