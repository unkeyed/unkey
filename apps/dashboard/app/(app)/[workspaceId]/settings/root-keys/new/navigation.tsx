"use client";

import { QuickNavPopover } from "@/components/navbar-popover";
import { Navbar } from "@/components/navigation/navbar";
import { ChevronExpandY, Gear } from "@unkey/icons";

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

export function Navigation({ workspaceId }: { workspaceId: string }) {
  return (
    <Navbar>
      <Navbar.Breadcrumbs icon={<Gear />}>
        <Navbar.Breadcrumbs.Link href={`/${workspaceId}/settings`}>
          Settings
        </Navbar.Breadcrumbs.Link>
        <Navbar.Breadcrumbs.Link href={`/${workspaceId}/settings/root-keys`} noop active>
          <QuickNavPopover
            items={settingsNavbar.flatMap((setting) => [
              {
                id: setting.href,
                label: setting.text,
                href: `/${workspaceId}/settings/${setting.href}`,
              },
            ])}
            shortcutKey="M"
          >
            <div className="flex items-center gap-1 p-1 rounded-lg hover:bg-gray-3">
              {"Root Keys"}
              <ChevronExpandY className="size-4" />
            </div>
          </QuickNavPopover>
        </Navbar.Breadcrumbs.Link>
        <Navbar.Breadcrumbs.Link href={`/${workspaceId}/settings/root-keys/new`}>
          New
        </Navbar.Breadcrumbs.Link>
      </Navbar.Breadcrumbs>
    </Navbar>
  );
}
