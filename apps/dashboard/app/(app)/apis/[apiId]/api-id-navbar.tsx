"use client";

import { CopyButton } from "@/components/dashboard/copy-button";
import { QuickNavPopover } from "@/components/navbar-popover";
import { Navbar } from "@/components/navigation/navbar";
import { Badge } from "@/components/ui/badge";
import { ChevronExpandY, Gauge } from "@unkey/icons";
import { CreateKeyDialog } from "./_components/create-key";

export const ApisNavbar = ({
  api,
  apis,
  activePage,
  keyId,
}: {
  api: {
    id: string;
    name: string;
    keyAuthId: string | null;
  };
  apis: {
    id: string;
    name: string;
  }[];
  activePage: {
    href: string;
    text: string;
  };
  keyId?: string;
}) => {
  return (
    <>
      <Navbar>
        <Navbar.Breadcrumbs icon={<Gauge />}>
          <Navbar.Breadcrumbs.Link href="/apis">APIs</Navbar.Breadcrumbs.Link>
          <Navbar.Breadcrumbs.Link href={`/apis/${api.id}`} isIdentifier className="group" noop>
            <QuickNavPopover
              items={apis.map((api) => ({
                id: api.id,
                label: api.name,
                href: `/apis/${api.id}`,
              }))}
              shortcutKey="N"
            >
              <div className="text-accent-10 group-hover:text-accent-12">{api.name}</div>
            </QuickNavPopover>
          </Navbar.Breadcrumbs.Link>
          <Navbar.Breadcrumbs.Link href={activePage.href} noop active>
            <QuickNavPopover
              items={[
                {
                  id: "requests",
                  label: "Requests",
                  href: `/apis/${api.id}`,
                },
                {
                  id: "keys",
                  label: "Keys",
                  href: `/apis/${api.id}/keys/${api.keyAuthId}`,
                },
                {
                  id: "settings",
                  label: "Settings",
                  href: `/apis/${api.id}/settings`,
                },
                ...(keyId
                  ? [
                      {
                        id: "settings",
                        label: `${keyId.substring(0, 8)}...${keyId.substring(keyId.length - 4)}`,
                        href: `/apis/${api.id}/keys/${api.keyAuthId}/${keyId}`,
                      },
                    ]
                  : []),
              ]}
              shortcutKey="M"
            >
              <div className="hover:bg-gray-3 rounded-lg flex items-center gap-1 p-1">
                {activePage.text}
                <ChevronExpandY className="size-4" />
              </div>
            </QuickNavPopover>
          </Navbar.Breadcrumbs.Link>
        </Navbar.Breadcrumbs>
        <Navbar.Actions>
          <Badge
            key="namespaceId"
            variant="secondary"
            className="flex justify-between w-full gap-2 font-mono font-medium ph-no-capture rounded-lg h-7"
          >
            {api.id}
            <CopyButton value={api.id} />
          </Badge>

          <CreateKeyDialog keyspaceId={api.keyAuthId} apiId={api.id} />
        </Navbar.Actions>
      </Navbar>
    </>
  );
};
