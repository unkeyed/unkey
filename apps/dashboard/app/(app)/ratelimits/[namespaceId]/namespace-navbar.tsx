"use client";

import { CopyButton } from "@/components/dashboard/copy-button";
import { Navbar } from "@/components/navigation/navbar";
import { QuickNavPopover } from "@/components/navbar-popover";
import { Badge } from "@/components/ui/badge";
import { ChevronExpandY, Gauge } from "@unkey/icons";

export const NamespaceNavbar = ({
  namespace,
  ratelimitNamespaces,
  activePage,
}: {
  namespace: {
    id: string;
    name: string;
    workspaceId: string;
  };
  ratelimitNamespaces: {
    id: string;
    name: string;
  }[];
  activePage: {
    href: string;
    text: string;
  };
}) => {
  return (
    <>
      <Navbar>
        <Navbar.Breadcrumbs icon={<Gauge />}>
          <Navbar.Breadcrumbs.Link href="/ratelimits">Ratelimits</Navbar.Breadcrumbs.Link>
          <Navbar.Breadcrumbs.Link
            href={`/ratelimits/${namespace.id}`}
            isIdentifier
            className="group"
            noop
          >
            <QuickNavPopover
              items={ratelimitNamespaces.map((ns) => ({
                id: ns.id,
                label: ns.name,
                href: `/ratelimits/${ns.id}`,
              }))}
              shortcutKey="N"
            >
              <div className="text-accent-10 group-hover:text-accent-12">{namespace.name}</div>
            </QuickNavPopover>
          </Navbar.Breadcrumbs.Link>
          <Navbar.Breadcrumbs.Link href={activePage.href} noop active>
            <QuickNavPopover
              items={[
                {
                  id: "logs",
                  label: "Logs",
                  href: `/ratelimits/${namespace.id}/logs`,
                },
                {
                  id: "settings",
                  label: "Settings",
                  href: `/ratelimits/${namespace.id}/settings`,
                },
                {
                  id: "overrides",
                  label: "Overrides",
                  href: `/ratelimits/${namespace.id}/overrides`,
                },
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
            className="flex justify-between w-full gap-2 font-mono font-medium ph-no-capture"
          >
            {namespace.id}
            <CopyButton value={namespace.id} />
          </Badge>
        </Navbar.Actions>
      </Navbar>
    </>
  );
};
