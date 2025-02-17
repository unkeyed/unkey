"use client";

import { CopyButton } from "@/components/dashboard/copy-button";
import { Navbar } from "@/components/navbar";
import { QuickNavPopover } from "@/components/navbar-popover";
import { Badge } from "@/components/ui/badge";
import { toast } from "@/components/ui/toaster";
import { cn } from "@/lib/utils";
import { Dots, Gauge } from "@unkey/icons";
import { Button } from "@unkey/ui";

export const NamespaceNavbar = ({
  namespaceId,
  namespaceName,
  ratelimitNamespaces,
}: {
  namespaceId: string;
  namespaceName: string;
  ratelimitNamespaces: {
    id: string;
    name: string;
  }[];
}) => {
  return (
    <Navbar>
      <Navbar.Breadcrumbs icon={<Gauge />}>
        <Navbar.Breadcrumbs.Link href="/ratelimits">Ratelimits</Navbar.Breadcrumbs.Link>
        <Navbar.Breadcrumbs.Link
          href={`/ratelimits/${namespaceId}`}
          isIdentifier
          className="group"
          noop
        >
          <div className="flex gap-[10px] items-center">
            <div className="flex items-center gap-1 group hover:bg-gray-3 rounded-lg">
              <QuickNavPopover
                items={ratelimitNamespaces.map((ns) => ({
                  id: ns.id,
                  label: ns.name,
                  href: `/ratelimits/${ns.id}`,
                }))}
                shortcutKey="R"
              >
                <Button
                  variant="ghost"
                  className={cn("group-data-[state=open]:bg-gray-4 px-1")}
                  aria-label="Select ratelimit"
                  aria-haspopup="true"
                  title="Press 'R' to toggle namespaces"
                >
                  <div>{namespaceName}</div>
                </Button>
              </QuickNavPopover>

              <QuickNavPopover
                shortcutKey="D"
                title="Namespace actions..."
                items={[
                  {
                    id: "edit",
                    hideRightIcon: true,
                    label: "Edit namespace name",
                  },
                  {
                    id: "copy",
                    label: "Copy ID",
                    hideRightIcon: true,
                    onClick: () => {
                      navigator.clipboard.writeText(namespaceId);
                      toast.success("Copied to clipboard", {
                        description: namespaceId,
                      });
                    },
                  },
                  {
                    id: "delete",
                    hideRightIcon: true,
                    label: <div className="text-error-11">Delete namespace</div>,
                  },
                ]}
              >
                <Button variant="ghost" size="icon" className="h-8 w-8" aria-label="More options">
                  <Dots />
                </Button>
              </QuickNavPopover>
            </div>
          </div>
        </Navbar.Breadcrumbs.Link>
        <Navbar.Breadcrumbs.Link href={`/ratelimits/${namespaceId}`} noop active>
          <Button
            variant="ghost"
            className={cn("group-data-[state=open]:bg-gray-4 px-1")}
            aria-label="Select ratelimit"
            aria-haspopup="true"
            title="Press 'M' to toggle other pages"
          >
            <QuickNavPopover
              items={[
                {
                  id: "logs",
                  label: "Logs",
                  href: `/ratelimits/${namespaceId}/logs`,
                },
                {
                  id: "settings",
                  label: "Settings",
                  href: `/ratelimits/${namespaceId}/settings`,
                },
                {
                  id: "overrides",
                  label: "Overrides",
                  href: `/ratelimits/${namespaceId}/overrides`,
                },
              ]}
              shortcutKey="M"
            >
              <div>Requests</div>
            </QuickNavPopover>
          </Button>
        </Navbar.Breadcrumbs.Link>
      </Navbar.Breadcrumbs>
      <Navbar.Actions>
        <Badge
          key="namespaceId"
          variant="secondary"
          className="flex justify-between w-full gap-2 font-mono font-medium ph-no-capture"
        >
          {namespaceId}
          <CopyButton value={namespaceId} />
        </Badge>
      </Navbar.Actions>
    </Navbar>
  );
};
