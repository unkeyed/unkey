"use client";
import { QuickNavPopover } from "@/components/navbar-popover";
import { NavbarActionButton } from "@/components/navigation/action-button";
import { CopyableIDButton } from "@/components/navigation/copyable-id-button";
import { Navbar } from "@/components/navigation/navbar";
import { trpc } from "@/lib/trpc/client";
import { useWorkspace } from "@/providers/workspace-provider";
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

export const NamespaceNavbar = ({
  namespaceId,
  includeOverrides = false,
  activePage,
}: NamespaceNavbarProps) => {
  const [open, setOpen] = useState(false);
  const { workspace } = useWorkspace();
  const { data, isLoading } = trpc.ratelimit.namespace.queryDetails.useQuery({
    namespaceId,
    includeOverrides,
  });

  if (!data || isLoading) {
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

  const { namespace, ratelimitNamespaces } = data;

  return (
    <>
      <Navbar>
        <Navbar.Breadcrumbs icon={<Gauge />}>
          <Navbar.Breadcrumbs.Link href={`/${workspace?.slug}/ratelimits`}>
            Ratelimits
          </Navbar.Breadcrumbs.Link>
          <Navbar.Breadcrumbs.Link
            href={`/${workspace?.slug}/ratelimits/${namespace.id}`}
            isIdentifier
            className="group"
            noop
          >
            <QuickNavPopover
              items={ratelimitNamespaces.map((ns) => ({
                id: ns.id,
                label: ns.name,
                href: `/${workspace?.slug}/ratelimits/${ns.id}`,
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
                  id: "requests",
                  label: "Requests",
                  href: `/${workspace?.slug}/ratelimits/${namespace.id}`,
                },
                {
                  id: "logs",
                  label: "Logs",
                  href: `/${workspace?.slug}/ratelimits/${namespace.id}/logs`,
                },
                {
                  id: "settings",
                  label: "Settings",
                  href: `/${workspace?.slug}/ratelimits/${namespace.id}/settings`,
                },
                {
                  id: "overrides",
                  label: "Overrides",
                  href: `/${workspace?.slug}/ratelimits/${namespace.id}/overrides`,
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
          <CopyableIDButton value={namespace.id} />
        </Navbar.Actions>
      </Navbar>
      {open && (
        <IdentifierDialog onOpenChange={setOpen} isModalOpen={open} namespaceId={namespace.id} />
      )}
    </>
  );
};
