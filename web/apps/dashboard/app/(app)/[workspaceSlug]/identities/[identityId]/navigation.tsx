"use client";
import { Navbar } from "@/components/navigation/navbar";
import { useWorkspaceNavigation } from "@/hooks/use-workspace-navigation";
import { trpc } from "@/lib/trpc/client";
import { Fingerprint } from "@unkey/icons";

import type { JSX } from "react";
import { IdentitySettings } from "./components/identity-settings";

type NavigationProps = {
  readonly identityId: string;
};

export function Navigation({ identityId }: NavigationProps): JSX.Element {
  const workspace = useWorkspaceNavigation();
  const { data: identity } = trpc.identity.getById.useQuery({ identityId });

  return (
    <Navbar>
      <Navbar.Breadcrumbs icon={<Fingerprint aria-hidden="true" focusable={false} />}>
        <Navbar.Breadcrumbs.Link href={`/${workspace.slug}/identities`}>
          Identities
        </Navbar.Breadcrumbs.Link>
        <Navbar.Breadcrumbs.Link
          href={`/${workspace.slug}/identities/${encodeURIComponent(identityId)}`}
          className="w-[200px] truncate"
          active
        >
          {identityId}
        </Navbar.Breadcrumbs.Link>
      </Navbar.Breadcrumbs>
      {identity ? (
        <Navbar.Actions>
          <IdentitySettings identity={identity} />
        </Navbar.Actions>
      ) : null}
    </Navbar>
  );
}
