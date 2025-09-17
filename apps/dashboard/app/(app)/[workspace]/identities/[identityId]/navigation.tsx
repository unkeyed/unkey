"use client";

import { Navbar } from "@/components/navigation/navbar";
import { Fingerprint } from "@unkey/icons";
import { useWorkspaceWithRedirect } from "@/hooks/use-workspace-with-redirect";

type NavigationProps = {
  readonly identityId: string;
};

export function Navigation({ identityId }: NavigationProps): JSX.Element {
  const { workspace } = useWorkspaceWithRedirect();

  return (
    <Navbar>
      <Navbar.Breadcrumbs icon={<Fingerprint aria-hidden="true" focusable={false} />}>
        <Navbar.Breadcrumbs.Link href={`/${workspace?.slug}/identities`}>
          Identities
        </Navbar.Breadcrumbs.Link>
        <Navbar.Breadcrumbs.Link
          href={`/${workspace?.slug}/identities/${encodeURIComponent(identityId)}`}
          className="w-[200px] truncate"
          active
          isIdentifier
        >
          {identityId}
        </Navbar.Breadcrumbs.Link>
      </Navbar.Breadcrumbs>
    </Navbar>
  );
}
