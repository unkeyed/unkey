"use client";

import { Navbar } from "@/components/navigation/navbar";
import { Nodes } from "@unkey/icons";
import { CreateKeyButton } from "@/components/dashboard/create-key-button";
import { CopyButton } from "@/components/dashboard/copy-button";
import { Badge } from "@/components/ui/badge";

type Key = {
  id: string;
  keyAuth: {
    id: string;
    api: {
      id: string;
      name: string;
    };
  };
};

interface NavigationProps {
  apiId: string;
  apiKey: Key;
}

export function Navigation({ apiId, apiKey }: NavigationProps) {
  return (
    <Navbar>
        <Navbar.Breadcrumbs icon={<Nodes />}>
          <Navbar.Breadcrumbs.Link href="/apis">APIs</Navbar.Breadcrumbs.Link>
          <Navbar.Breadcrumbs.Link href={`/apis/${apiId}`} isIdentifier>
            {apiKey.keyAuth.api.name}
          </Navbar.Breadcrumbs.Link>
          <Navbar.Breadcrumbs.Ellipsis />
          <Navbar.Breadcrumbs.Link
            href={`/apis/${apiId}/keys/${apiKey.keyAuth.id}/${apiKey.id}`}
            isIdentifier
          >
            {apiKey.id}
          </Navbar.Breadcrumbs.Link>
          <Navbar.Breadcrumbs.Link
            href={`/apis/${apiId}/keys/${apiKey.keyAuth.id}/${apiKey.id}/settings`}
            active
          >
            Settings
          </Navbar.Breadcrumbs.Link>
        </Navbar.Breadcrumbs>
        <Navbar.Actions>
          <Badge
            variant="secondary"
            className="flex justify-between w-full gap-2 font-mono font-medium ph-no-capture"
          >
            {apiKey.id}
            <CopyButton value={apiKey.id} />
          </Badge>
          <CreateKeyButton apiId={apiKey.keyAuth.api.id} keyAuthId={apiKey.keyAuth.id} />
        </Navbar.Actions>
      </Navbar>
  )
}
