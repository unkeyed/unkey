"use client";

import { Navbar } from "@/components/navigation/navbar";
import { Nodes } from "@unkey/icons";
import { CreateKeyDialog } from "../../../../_components/create-key";

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
          {apiKey.id?.substring(0, 8)}...
          {apiKey.id?.substring(apiKey.id?.length - 4)}
        </Navbar.Breadcrumbs.Link>
        <Navbar.Breadcrumbs.Link
          href={`/apis/${apiId}/keys/${apiKey.keyAuth.id}/${apiKey.id}/settings`}
          active
        >
          Settings
        </Navbar.Breadcrumbs.Link>
      </Navbar.Breadcrumbs>
      <div className="items-center flex">
        <CreateKeyDialog keyspaceId={apiKey.keyAuth.id} apiId={apiId} copyIdValue={apiKey.id} />
      </div>
    </Navbar>
  );
}
