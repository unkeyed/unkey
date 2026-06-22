"use client";
import { Navbar } from "@/components/navigation/navbar";
import { useWorkspaceNavigation } from "@/hooks/use-workspace-navigation";
import { routes } from "@/lib/navigation/routes";
import { Fingerprint } from "@unkey/icons";

import type { JSX } from "react";
import { IdentitySettingsDialog } from "./components/identity-settings-dialog";

type NavigationProps = {
  readonly identityId: string;
};

export function Navigation({ identityId }: NavigationProps): JSX.Element {
  const workspace = useWorkspaceNavigation();

  return (
    <Navbar>
      <Navbar.Breadcrumbs icon={<Fingerprint aria-hidden="true" focusable={false} />}>
        <Navbar.Breadcrumbs.Link href={routes.identities.list({ workspaceSlug: workspace.slug })}>
          Identities
        </Navbar.Breadcrumbs.Link>
        <Navbar.Breadcrumbs.Link
          href={routes.identities.detail({ workspaceSlug: workspace.slug, identityId })}
          className="w-[200px] truncate"
          active
        >
          {identityId}
        </Navbar.Breadcrumbs.Link>
      </Navbar.Breadcrumbs>
      <Navbar.Actions>
        <IdentitySettingsDialog identityId={identityId} />
      </Navbar.Actions>
    </Navbar>
  );
}
