"use client";

import { CopyButton } from "@/components/dashboard/copy-button";
import { CreateKeyButton } from "@/components/dashboard/create-key-button";
import { Navbar } from "@/components/navigation/navbar";
import { Badge } from "@/components/ui/badge";
import type { Api } from "@unkey/db";
import { Nodes } from "@unkey/icons";

export function Navigation({ api }: { api: Api }) {
  return (
    <Navbar>
      <Navbar.Breadcrumbs icon={<Nodes />}>
        <Navbar.Breadcrumbs.Link href="/apis">APIs</Navbar.Breadcrumbs.Link>
        <Navbar.Breadcrumbs.Link href={`/apis/${api.id}`} isIdentifier>
          {api.name}
        </Navbar.Breadcrumbs.Link>
        <Navbar.Breadcrumbs.Link active href={`/apis/${api.id}/settings`}>
          Settings
        </Navbar.Breadcrumbs.Link>
      </Navbar.Breadcrumbs>
      <Navbar.Actions>
        <Badge
          key="apiId"
          variant="secondary"
          className="flex justify-between w-full gap-2 font-mono font-medium ph-no-capture"
        >
          {api.id}
          <CopyButton value={api.id} />
        </Badge>
        <CreateKeyButton apiId={api.id} keyAuthId={api.keyAuthId!} />
      </Navbar.Actions>
    </Navbar>
  );
}
