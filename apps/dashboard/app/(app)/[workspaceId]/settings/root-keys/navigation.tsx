"use client";
import { QuickNavPopover } from "@/components/navbar-popover";
import { Navbar } from "@/components/navigation/navbar";
import { ChevronExpandY, Gear } from "@unkey/icons";
import { Badge, Button, CopyButton } from "@unkey/ui";
import Link from "next/link";
import { CreateRootKeyButton } from "./components/root-key/create-rootkey-button";

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

export const Navigation = ({
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
          <Navbar.Breadcrumbs.Link href={`/${workspace.id}/settings`}>
            Settings
          </Navbar.Breadcrumbs.Link>
          <Navbar.Breadcrumbs.Link
            href={`/${workspace.id}/settings/${activePage.href}`}
            noop
            active
          >
            <QuickNavPopover
              items={settingsNavbar.flatMap((setting) => [
                {
                  id: setting.href,
                  label: setting.text,
                  href: `/${workspace.id}/settings/${setting.href}`,
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
              variant="secondary"
              className="max-w-[160px] truncate whitespace-nowrap"
              title={workspace.id}
            >
              {workspace.id}
              <CopyButton value={workspace.id} />
            </Badge>
          )}
          {activePage.href === "root-keys" && <CreateRootKeyButton />}
          {activePage.href === "billing" && (
            <>
              <Button asChild variant="outline">
                <Link
                  href="https://cal.com/james-r-perkins/sales"
                  target="_blank"
                  rel="noopener noreferrer"
                >
                  Schedule a call
                </Link>
              </Button>
              <Button asChild variant="primary">
                <Link href="mailto:support@unkey.dev">Contact us</Link>
              </Button>
            </>
          )}
        </Navbar.Actions>
      </Navbar>
    </div>
  );
};
