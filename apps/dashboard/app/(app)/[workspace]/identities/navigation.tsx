"use client";

import { Navbar } from "@/components/navigation/navbar";
import { useWorkspaceNavigation } from "@/hooks/use-workspace-navigation";
import { Fingerprint } from "@unkey/icons";
import { Loading } from "@unkey/ui";
import { Suspense } from "react";

export function Navigation() {
  const workspace = useWorkspaceNavigation();

  return (
    <Suspense fallback={<Loading type="spinner" />}>
      <Navbar>
        <Navbar.Breadcrumbs icon={<Fingerprint aria-hidden="true" focusable={false} />}>
          <Navbar.Breadcrumbs.Link href={`/${workspace.slug}/identities`} active>
            Identities
          </Navbar.Breadcrumbs.Link>
        </Navbar.Breadcrumbs>
      </Navbar>
    </Suspense>
  );
}
