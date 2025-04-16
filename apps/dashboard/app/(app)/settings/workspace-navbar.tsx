"use client";

import { CopyButton } from "@/components/dashboard/copy-button";
import { QuickNavPopover } from "@/components/navbar-popover";
import { Navbar } from "@/components/navigation/navbar";
import { Badge } from "@/components/ui/badge";
import { ChevronExpandY, Gear } from "@unkey/icons";
import { Button } from "@unkey/ui";
import Link from "next/link";

const settingsNavbar = [
  {
    id: "general",
    href: "general",
    text: "General",
  },
  {
    id: "team",
    href: "team",
    text: "Team",
  },
  {
    id: "root-keys",
    href: "root-keys",
    text: "Root Keys",
  },
  {
    id: "billing",
    href: "billing",
    text: "Billing",
  },
];

export const WorkspaceNavbar = ({
  workspace,
  activePage,
}: {
  workspace: {
    id: string;
    name: string;
  };
  activePage: {
    href: string;
    text: string;
  };
}) => {
  return (
    <div className="flex flex-col w-full h-full">
      <Navbar>
        <Navbar.Breadcrumbs icon={<Gear />}>
          <Navbar.Breadcrumbs.Link href="/settings">Settings</Navbar.Breadcrumbs.Link>
          <Navbar.Breadcrumbs.Link href={activePage.href} noop active>
            <QuickNavPopover
              items={settingsNavbar.flatMap((setting) => [
                {
                  id: setting.href,
                  label: setting.text,
                  href: `/settings/${setting.href}`,
                },
              ])}
              shortcutKey="M"
            >
              <div className="flex items-center gap-1 p-1 rounded-lg hover:bg-gray-3">
                {activePage.text}
                <ChevronExpandY className="size-4" />
              </div>
            </QuickNavPopover>
          </Navbar.Breadcrumbs.Link>
        </Navbar.Breadcrumbs>
        <Navbar.Actions>
          {activePage.href === "general" && (
            <Badge
              key="namespaceId"
              variant="secondary"
              className="flex justify-between w-full gap-2 font-mono font-medium ph-no-capture"
            >
              {workspace.id}
              <CopyButton value={workspace.id} />
            </Badge>
          )}
          {activePage.href === "root-keys" && (
            <Link key="create-root-key" href="/settings/root-keys/new">
              <Button variant="primary">Create New Root Key</Button>
            </Link>
          )}
          {activePage.href === "billing" && (
            <Navbar.Actions>
              <Button variant="outline">
                <Link href="https://cal.com/james-r-perkins/sales" target="_blank">
                  Schedule a call
                </Link>
              </Button>
              <Button variant="primary">
                <Link href="mailto:support@unkey.dev">Contact us</Link>
              </Button>
            </Navbar.Actions>
          )}
        </Navbar.Actions>
      </Navbar>
    </div>
  );
};
