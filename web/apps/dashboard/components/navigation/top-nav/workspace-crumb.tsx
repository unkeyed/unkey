"use client";

import { Avatar, AvatarFallback } from "@/components/ui/avatar";
import { useWorkspaceNavigation } from "@/hooks/use-workspace-navigation";
import { routes } from "@/lib/navigation/routes";
import { trpc } from "@/lib/trpc/client";
import { useWorkspace } from "@/providers/workspace-provider";
import { IconPlusOutline18 } from "@unkey/icons";
import { Button } from "@unkey/ui";
import { useMemo } from "react";
import { Crumb } from "./crumb";
import type { CrumbPopoverItem } from "./crumb-popover";

export function WorkspaceCrumb({ href }: { href: string }) {
  const workspace = useWorkspaceNavigation();
  const { switchWorkspace } = useWorkspace();
  const available = trpc.workspace.listAvailable.useQuery();
  const orgs = available.isError ? [] : (available.data ?? []);
  const items: CrumbPopoverItem[] = useMemo(
    () =>
      orgs.map((m) => ({
        id: m.orgId,
        label: m.name,
        onClick: () => switchWorkspace(m.orgId, routes.workspaces.root()),
      })),
    [orgs, switchWorkspace],
  );

  return (
    <Crumb
      icon={
        <Avatar className="size-4 rounded-sm border border-input shrink-0">
          <AvatarFallback name={workspace.name} variant="marble" square />
        </Avatar>
      }
      label={workspace.name}
      href={href}
      items={items}
      currentId={workspace.orgId}
      searchPlaceholder="Find workspace..."
      emptyText="No workspaces found"
      listStatus={
        available.isError ? (
          <div role="alert" className="flex flex-col items-center gap-2 px-3 py-4 text-sm">
            <span>Unable to load workspaces</span>
            <Button
              variant="outline"
              size="sm"
              disabled={available.isFetching}
              onClick={() => available.refetch()}
            >
              Try again
            </Button>
          </div>
        ) : available.isLoading ? (
          <output className="block px-3 py-4 text-sm">Loading workspaces...</output>
        ) : orgs.length === 0 ? (
          <output className="block px-3 py-4 text-sm">No workspaces found</output>
        ) : undefined
      }
      footer={{ icon: IconPlusOutline18, label: "New workspace", href: routes.workspaces.create() }}
    />
  );
}
