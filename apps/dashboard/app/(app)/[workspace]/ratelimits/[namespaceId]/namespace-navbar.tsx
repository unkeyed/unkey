"use client";
import { QuickNavPopover } from "@/components/navbar-popover";
import { NavbarActionButton } from "@/components/navigation/action-button";
import { CopyableIDButton } from "@/components/navigation/copyable-id-button";
import { Navbar } from "@/components/navigation/navbar";
import { collection } from "@/lib/collections";
import { useWorkspace } from "@/providers/workspace-provider";
import { useLiveQuery } from "@tanstack/react-db";
import { ChevronExpandY, Gauge, TaskUnchecked } from "@unkey/icons";
import dynamic from "next/dynamic";
import { useState } from "react";

const IdentifierDialog = dynamic(
  () => import("./_components/identifier-dialog").then((mod) => mod.IdentifierDialog),
  {
    loading: () => null,
    ssr: false,
  },
);

type NamespaceNavbarProps = {
  namespaceId: string;
  includeOverrides?: boolean;
  activePage: {
    href: string;
    text: string;
  };
};

export const NamespaceNavbar = ({ namespaceId, activePage }: NamespaceNavbarProps) => {
  const [open, setOpen] = useState(false);
  const { workspace } = useWorkspace();
  const { data } = useLiveQuery((q) => q.from({ namespace: collection.ratelimitNamespaces }));

  if (!data) {
    return (
      <Navbar>
        <Navbar.Breadcrumbs icon={<Gauge />}>
          <Navbar.Breadcrumbs.Link href={`/${workspace?.slug}/ratelimits`}>
            Ratelimits
          </Navbar.Breadcrumbs.Link>
          <Navbar.Breadcrumbs.Link href="#" isIdentifier className="group" noop>
            <div className="h-6 w-20 bg-grayA-3 rounded animate-pulse transition-all " />
          </Navbar.Breadcrumbs.Link>
          <Navbar.Breadcrumbs.Link href="#" noop active>
            <div className="hover:bg-gray-3 rounded-lg flex items-center gap-1 p-1">
              <div className="h-6 w-16 bg-grayA-3 rounded animate-pulse transition-all " />
              <ChevronExpandY className="size-4" />
            </div>
          </Navbar.Breadcrumbs.Link>
        </Navbar.Breadcrumbs>
        <Navbar.Actions>
          <div className="h-7 w-[146px] bg-grayA-3 rounded-md animate-pulse border border-grayA-4 transition-all " />
          <div className="h-7 bg-grayA-2 border border-gray-6 rounded-md animate-pulse px-3 flex gap-2 items-center justify-center w-[260px] transition-all ">
            <div className="h-3 w-[260px] bg-grayA-3 rounded" />
            <div>
              <TaskUnchecked size="sm-regular" className="!size-4" />
            </div>
          </div>
        </Navbar.Actions>
      </Navbar>
    );
  }

  const namespace = data.find((ns) => ns.id === namespaceId);

  return (
    <>
      <Navbar>
        <Navbar.Breadcrumbs icon={<Gauge />}>
          <Navbar.Breadcrumbs.Link href={`/${workspace?.slug}/ratelimits`}>
            Ratelimits
          </Navbar.Breadcrumbs.Link>
          <Navbar.Breadcrumbs.Link
            href={`/${workspace?.slug}/ratelimits/${namespaceId}`}
            isIdentifier
            className="group"
            noop
          >
            <QuickNavPopover
              items={data.map((ns) => ({
                id: ns.id,
                label: ns.name,
                href: `/${workspace?.slug}/ratelimits/${ns.id}`,
              }))}
              shortcutKey="N"
            >
              <div className="text-accent-10 group-hover:text-accent-12">{namespace?.name}</div>
            </QuickNavPopover>
          </Navbar.Breadcrumbs.Link>
          <Navbar.Breadcrumbs.Link href={activePage.href} noop active>
            <QuickNavPopover
              items={[
                {
                  id: "requests",
                  label: "Requests",
                  href: `/${workspace?.slug}/ratelimits/${namespaceId}`,
                },
                {
                  id: "logs",
                  label: "Logs",
                  href: `/${workspace?.slug}/ratelimits/${namespaceId}/logs`,
                },
                {
                  id: "settings",
                  label: "Settings",
                  href: `/${workspace?.slug}/ratelimits/${namespaceId}/settings`,
                },
                {
                  id: "overrides",
                  label: "Overrides",
                  href: `/${workspace?.slug}/ratelimits/${namespaceId}/overrides`,
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
          <NavbarActionButton title="Override Identifier" onClick={() => setOpen(true)}>
            Override Identifier
          </NavbarActionButton>
          <CopyableIDButton value={namespaceId} />
        </Navbar.Actions>
      </Navbar>
      {open && (
        <IdentifierDialog onOpenChange={setOpen} isModalOpen={open} namespaceId={namespaceId} />
      )}
    </>
  );
};
