"use client";

import { CopyButton } from "@/components/dashboard/copy-button";
import { CreateKeyButton } from "@/components/dashboard/create-key-button";
import { Navbar } from "@/components/navigation/navbar";
import { Badge } from "@/components/ui/badge";
import type { Api } from "@unkey/db";
import { Nodes } from "@unkey/icons";

type Key = {
  id: string;
  keyAuth: {
    id: string;
  };
};

interface NavigationProps {
  api: Api;
  apiKey: Key;
}

export function Navigation({ api, apiKey }: NavigationProps) {
  return (
    <Navbar>
      <Navbar.Breadcrumbs icon={<Nodes />}>
        <Navbar.Breadcrumbs.Link href="/apis">APIs</Navbar.Breadcrumbs.Link>
        <Navbar.Breadcrumbs.Link href={`/apis/${api.id}`} isIdentifier>
          {api.name}
        </Navbar.Breadcrumbs.Link>
        <Navbar.Breadcrumbs.Ellipsis />
        <Navbar.Breadcrumbs.Link
          href={`/apis/${api.id}/keys/${apiKey.keyAuth.id}/${apiKey.id}`}
          isIdentifier
          active
        >
          {apiKey.id}
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

        <CreateKeyButton apiId={api.id} keyAuthId={apiKey.keyAuth.id} />
      </Navbar.Actions>
    </Navbar>
  );
}
