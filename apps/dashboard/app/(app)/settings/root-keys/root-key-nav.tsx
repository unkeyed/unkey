"use client";

import { QuickNavPopover } from "@/components/navbar-popover";
import { Navbar } from "@/components/navigation/navbar";
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

export const RootKeyNav = ({
  activePage,
}: {
  activePage: {
    href: string;
    text: string;
  };
}) => {
  return (
    <Navbar>
      <Navbar.Breadcrumbs icon={<Gear />}>
        <Navbar.Breadcrumbs.Link href="/settings/root-keys">Root Keys</Navbar.Breadcrumbs.Link>
        <Navbar.Breadcrumbs.Link href={activePage.href} noop active>
          <QuickNavPopover
            items={settingsNavbar.flatMap((setting) => [
              {
                id: setting.href,
                label: setting.text,
                href: `/settings/root-keys/${setting.href}`,
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
        <Link key="create-root-key" href="/settings/root-keys/new">
          <Button asChild variant="primary">
            Create New Root Key
          </Button>
        </Link>
      </Navbar.Actions>
    </Navbar>
  );
};
