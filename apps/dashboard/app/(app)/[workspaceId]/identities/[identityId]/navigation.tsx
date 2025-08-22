"use client";

import { Navbar } from "@/components/navigation/navbar";
import { Fingerprint } from "@unkey/icons";

type NavigationProps = {
  readonly identityId: string;
  readonly workspaceId: string;
};

export function Navigation({ identityId, workspaceId }: NavigationProps): JSX.Element {
  return (
    <Navbar>
      <Navbar.Breadcrumbs icon={<Fingerprint aria-hidden="true" focusable={false} />}>
        <Navbar.Breadcrumbs.Link href={`/${encodeURIComponent(workspaceId)}/identities`}>
          Identities
        </Navbar.Breadcrumbs.Link>
        <Navbar.Breadcrumbs.Link
          href={`/${encodeURIComponent(workspaceId)}/identities/${encodeURIComponent(identityId)}`}
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
