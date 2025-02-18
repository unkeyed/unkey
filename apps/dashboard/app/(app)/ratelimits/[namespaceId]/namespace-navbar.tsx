"use client";

import { CopyButton } from "@/components/dashboard/copy-button";
import { Navbar } from "@/components/navbar";
import { QuickNavPopover } from "@/components/navbar-popover";
import { Badge } from "@/components/ui/badge";
import { toast } from "@/components/ui/toaster";
import { cn } from "@/lib/utils";
import { Dots, Gauge } from "@unkey/icons";
import { Button } from "@unkey/ui";
import { useState } from "react";
import { DeleteNamespaceDialog } from "./_components/namespace-delete-dialog";
import { NamespaceUpdateNameDialog } from "./_components/namespace-update-dialog";

export const NamespaceNavbar = ({
  namespace,
  ratelimitNamespaces,
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
}) => {
  const [isNamespaceNameUpdateModalOpen, setIsNamespaceNameUpdateModalOpen] = useState(false);

  const [isNamespaceNameDeleteModalOpen, setIsNamespaceNameDeleteModalOpen] = useState(false);
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
                    <div className="text-accent-10 group-hover:text-accent-12">
                      {namespace.name}
                    </div>
                  </Button>
                </QuickNavPopover>

                <QuickNavPopover
                  shortcutKey="D"
                  title="Namespace actions..."
                  items={[
                    {
                      id: "edit",
                      hideRightIcon: true,
                      label: "Edit namespace",
                      onClick() {
                        setIsNamespaceNameUpdateModalOpen(true);
                      },
                    },
                    {
                      id: "copy",
                      label: "Copy ID",
                      hideRightIcon: true,
                      onClick: () => {
                        navigator.clipboard.writeText(namespace.id);
                        toast.success("Copied to clipboard", {
                          description: namespace.id,
                        });
                      },
                    },
                    {
                      id: "delete",
                      hideRightIcon: true,
                      itemClassName: "hover:bg-error-3",
                      label: <div className="text-error-11">Delete namespace</div>,
                      onClick() {
                        setIsNamespaceNameDeleteModalOpen(true);
                      },
                    },
                  ]}
                >
                  <Button
                    variant="ghost"
                    size="icon"
                    className="h-8 w-8 text-accent-10 group-hover:text-accent-12 hover:bg-transparent"
                    aria-label="More options"
                  >
                    <Dots />
                  </Button>
                </QuickNavPopover>
              </div>
            </div>
          </Navbar.Breadcrumbs.Link>
          <Navbar.Breadcrumbs.Link href={`/ratelimits/${namespace.id}`} noop active>
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
            {namespace.id}
            <CopyButton value={namespace.id} />
          </Badge>
        </Navbar.Actions>
      </Navbar>
      <NamespaceUpdateNameDialog
        namespace={namespace}
        onOpenChange={setIsNamespaceNameUpdateModalOpen}
        isModalOpen={isNamespaceNameUpdateModalOpen}
      />
      <DeleteNamespaceDialog
        namespace={namespace}
        onOpenChange={setIsNamespaceNameDeleteModalOpen}
        isModalOpen={isNamespaceNameDeleteModalOpen}
      />
    </>
  );
};
