"use client";

import { Navbar } from "@/components/navigation/navbar";
import { useWorkspace } from "@/providers/workspace-provider";
import { Fingerprint } from "@unkey/icons";
import { useRouter } from "next/navigation";

type NavigationProps = {
  readonly identityId: string;
};

export function Navigation({ identityId }: NavigationProps): JSX.Element {
  const { workspace, isLoading } = useWorkspace();
  const router = useRouter();

  if (!workspace && !isLoading) {
    router.replace("/new");
  }

  return (
    <Navbar>
      <Navbar.Breadcrumbs icon={<Fingerprint aria-hidden="true" focusable={false} />}>
        <Navbar.Breadcrumbs.Link href={`/${encodeURIComponent(workspace?.slug ?? "")}/identities`}>
          Identities
        </Navbar.Breadcrumbs.Link>
        <Navbar.Breadcrumbs.Link
          href={`/${encodeURIComponent(workspace?.slug ?? "")}/identities/${encodeURIComponent(identityId)}`}
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
