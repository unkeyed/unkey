"use client";
import { NavbarActionButton } from "@/components/navigation/action-button";
import { CopyableIDButton } from "@/components/navigation/copyable-id-button";
import { Navbar } from "@/components/navigation/navbar";
import { useWorkspaceNavigation } from "@/hooks/use-workspace-navigation";
import { collection } from "@/lib/collections";
import { routes } from "@/lib/navigation/routes";
import { useLiveQuery } from "@tanstack/react-db";
import { ChevronExpandY, Gauge, Plus, TaskUnchecked } from "@unkey/icons";
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
  activePage: {
    href: string;
    text: string;
  };
};

export const NamespaceNavbar = ({ namespaceId, activePage }: NamespaceNavbarProps) => {
  const [open, setOpen] = useState(false);
  const workspace = useWorkspaceNavigation();

  const { data } = useLiveQuery((q) => q.from({ namespace: collection.ratelimitNamespaces }));

  if (!data) {
    return (
      <Navbar>
        <Navbar.Breadcrumbs icon={<Gauge />}>
          <Navbar.Breadcrumbs.Link
            href={routes.ratelimits.list({ workspaceSlug: workspace.slug })}
          >
            Ratelimits
          </Navbar.Breadcrumbs.Link>
          <Navbar.Breadcrumbs.Link href="#" className="group" noop>
            <div className="h-6 w-20 bg-grayA-3 rounded-sm animate-pulse transition-all " />
          </Navbar.Breadcrumbs.Link>
          <Navbar.Breadcrumbs.Link href="#" noop active>
            <div className="hover:bg-gray-3 rounded-lg flex items-center gap-1 p-1">
              <div className="h-6 w-16 bg-grayA-3 rounded-sm animate-pulse transition-all " />
              <ChevronExpandY className="size-4" />
            </div>
          </Navbar.Breadcrumbs.Link>
        </Navbar.Breadcrumbs>
        <Navbar.Actions>
          <div className="h-7 w-[146px] bg-grayA-3 rounded-md animate-pulse border border-grayA-4 transition-all " />
          <div className="h-7 bg-grayA-2 border border-gray-6 rounded-md animate-pulse px-3 flex gap-2 items-center justify-center w-[260px] transition-all ">
            <div className="h-3 w-[260px] bg-grayA-3 rounded-sm" />
            <div>
              <TaskUnchecked iconSize="sm-regular" className="size-4!" />
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
          <Navbar.Breadcrumbs.Link
            href={routes.ratelimits.list({ workspaceSlug: workspace.slug })}
          >
            Ratelimits
          </Navbar.Breadcrumbs.Link>
          <Navbar.Breadcrumbs.Link
            href={routes.ratelimits.detail({ workspaceSlug: workspace.slug, namespaceId })}
            className="group"
            noop
          >
            {namespace?.name}
          </Navbar.Breadcrumbs.Link>
          <Navbar.Breadcrumbs.Link href={activePage.href} noop active>
            {activePage.text}
          </Navbar.Breadcrumbs.Link>
        </Navbar.Breadcrumbs>
        <Navbar.Actions>
          <NavbarActionButton title="Override Identifier" onClick={() => setOpen(true)}>
            <Plus />
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
