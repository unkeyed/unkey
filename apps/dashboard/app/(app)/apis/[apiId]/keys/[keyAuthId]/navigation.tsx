"use client";

import { CopyButton } from "@/components/dashboard/copy-button";
import { CreateKeyButton } from "@/components/dashboard/create-key-button";
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
  workspace: {
    tenantId: string;
  };
};

interface NavigationProps {
  apiId: string;
  keyA: KeyAuthProps;
}

export function Navigation({ apiId, keyA }: NavigationProps) {
  return (
    <Navbar>
      <Navbar.Breadcrumbs icon={<Nodes />}>
        <Navbar.Breadcrumbs.Link href="/apis">APIs</Navbar.Breadcrumbs.Link>
        <Navbar.Breadcrumbs.Link href={`/apis/${apiId}`} isIdentifier>
          {keyA.api.name}
        </Navbar.Breadcrumbs.Link>
        <Navbar.Breadcrumbs.Link active href={`/apis/${apiId}/keys/${keyA.id}`}>
          Keys
        </Navbar.Breadcrumbs.Link>
      </Navbar.Breadcrumbs>
      <Navbar.Actions>
        <Badge
          key="apiId"
          variant="secondary"
          className="flex justify-between w-full gap-2 font-mono font-medium ph-no-capture"
        >
          {keyA.api.id}
          <CopyButton value={keyA.api.id} />
        </Badge>
        <CreateKeyButton apiId={keyA.api.id} keyAuthId={keyA.api.keyAuthId!} />
      </Navbar.Actions>
    </Navbar>
  );
}
