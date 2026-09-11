"use client";

import { Avatar, AvatarFallback } from "@/components/ui/avatar";
import { useWorkspaceNavigation } from "@/hooks/use-workspace-navigation";
import { routes } from "@/lib/navigation/routes";
import { trpc } from "@/lib/trpc/client";
import { IconPlusOutline18 } from "@unkey/icons";
import { Button } from "@unkey/ui";
import { useMemo, useState } from "react";
import { Crumb } from "./crumb";
import type { CrumbPopoverItem } from "./crumb-popover";

export function WorkspaceCrumb({
  href,
  compactOnMobile,
}: {
  href: string;
  compactOnMobile: boolean;
}) {
  const workspace = useWorkspaceNavigation();
  const available = trpc.workspace.listAvailable.useQuery();
  const orgs = available.isError ? [] : (available.data ?? []);
  const [switchingOrgId, setSwitchingOrgId] = useState<string | null>(null);
  const items: CrumbPopoverItem[] = useMemo(
    () =>
      orgs.map((m) => ({
        id: m.orgId,
        label: m.name,
        onClick: () => {
          if (m.orgId !== workspace.orgId && !switchingOrgId) {
            setSwitchingOrgId(m.orgId);
            window.location.assign(
              routes.auth.switchOrganization({
                organizationId: m.orgId,
                returnTo: routes.workspaces.root(),
              }),
            );
          }
        },
      })),
    [orgs, switchingOrgId, workspace.orgId],
  );

  return (
    <Crumb
      icon={
        <Avatar className="size-4 rounded-sm border border-input shrink-0">
          <AvatarFallback name={workspace.name} variant="marble" square />
        </Avatar>
      }
      label={workspace.name}
      compactOnMobile={compactOnMobile}
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
