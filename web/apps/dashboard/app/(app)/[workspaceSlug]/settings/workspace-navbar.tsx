"use client";
import { QuickNavPopover } from "@/components/navbar-popover";
import { CopyableIDButton } from "@/components/navigation/copyable-id-button";
import { Navbar } from "@/components/navigation/navbar";
import { useWorkspaceNavigation } from "@/hooks/use-workspace-navigation";
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
  activePage,
}: {
  activePage: {
    href: string;
    text: string;
  };
}) => {
  const workspace = useWorkspaceNavigation();

  return (
    <div className="flex flex-col w-full h-full">
      <Navbar>
        <Navbar.Breadcrumbs icon={<Gear />}>
          <Navbar.Breadcrumbs.Link href={`/${workspace.slug}/settings`}>
            Settings
          </Navbar.Breadcrumbs.Link>
          <Navbar.Breadcrumbs.Link href={activePage.href} noop active>
            <QuickNavPopover
              items={settingsNavbar.flatMap((setting) => [
                {
                  id: setting.href,
                  label: setting.text,
                  href: `/${workspace.slug}/settings/${setting.href}`,
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
          {activePage.href === "general" && workspace && <CopyableIDButton value={workspace.id} />}
          {activePage.href === "root-keys" && (
            <Link key="create-root-key" href={`/${workspace.slug}/settings/root-keys/new`}>
              <Button variant="primary">Create New Root Key</Button>
            </Link>
          )}
          {activePage.href === "billing" && (
            <>
              <Link
                href="https://cal.com/james-r-perkins/sales"
                target="_blank"
                rel="noopener noreferrer"
              >
                <Button type="button" variant="outline">
                  Schedule a call
                </Button>
              </Link>
              <Link href="mailto:support@unkey.com">
                <Button type="button" variant="primary">
                  Contact us
                </Button>
              </Link>
            </>
          )}
        </Navbar.Actions>
      </Navbar>
    </div>
  );
};
