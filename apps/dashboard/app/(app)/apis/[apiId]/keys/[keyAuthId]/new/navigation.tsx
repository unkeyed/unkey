"use client";

import { CopyButton } from "@/components/dashboard/copy-button";
import { Navbar } from "@/components/navigation/navbar";
import { Badge } from "@/components/ui/badge";
import { Nodes } from "@unkey/icons";

type KeyAuthProps = {
  id: string;
  api: {
    id: string;
    name: string;
    keyAuthId: string | null;
  };
};

interface NavigationProps {
  apiId: string;
  keyAuth: KeyAuthProps;
}

export function Navigation({ apiId, keyAuth }: NavigationProps) {
  return (
    <Navbar>
      <Navbar.Breadcrumbs icon={<Nodes />}>
        <Navbar.Breadcrumbs.Link href="/apis">APIs</Navbar.Breadcrumbs.Link>
        <Navbar.Breadcrumbs.Link href={`/apis/${apiId}`} isIdentifier>
          {keyAuth.api.name}
        </Navbar.Breadcrumbs.Link>
        <Navbar.Breadcrumbs.Link href={`/apis/${apiId}/keys/${keyAuth.id}`}>
          Keys
        </Navbar.Breadcrumbs.Link>
        <Navbar.Breadcrumbs.Link active href={`/apis/${apiId}/keys/${keyAuth.id}/new`}>
          Create new key
        </Navbar.Breadcrumbs.Link>
      </Navbar.Breadcrumbs>
      <Navbar.Actions>
        <Badge
          key="apiId"
          variant="secondary"
          className="flex justify-between w-full gap-2 font-mono font-medium ph-no-capture"
        >
          {keyAuth.api.id}
          <CopyButton value={keyAuth.api.id} />
        </Badge>
      </Navbar.Actions>
    </Navbar>
  );
}
