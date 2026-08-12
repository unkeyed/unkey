"use client";

import { Avatar, AvatarFallback } from "@/components/ui/avatar";
import { useWorkspaceNavigation } from "@/hooks/use-workspace-navigation";
import { routes } from "@/lib/navigation/routes";
import { trpc } from "@/lib/trpc/client";
import { Plus } from "@unkey/icons";
import { useMemo, useState } from "react";
import { Crumb } from "./crumb";
import type { CrumbPopoverItem } from "./crumb-popover";

export function WorkspaceCrumb({ href }: { href: string }) {
  const workspace = useWorkspaceNavigation();
  const { data: user } = trpc.user.getCurrentUser.useQuery();
  const { data: memberships } = trpc.user.listMemberships.useQuery(user?.id ?? "", {
    enabled: !!user?.id,
  });
  const orgs = memberships?.data ?? [];
  const [switchingOrgId, setSwitchingOrgId] = useState<string | null>(null);
  const items: CrumbPopoverItem[] = useMemo(
    () =>
      orgs.map((m) => ({
        id: m.organization.id,
        label: m.organization.name,
        onClick: () => {
          if (m.organization.id !== workspace.orgId && !switchingOrgId) {
            setSwitchingOrgId(m.organization.id);
            window.location.assign(
              routes.auth.switchOrganization({
                organizationId: m.organization.id,
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
        <Avatar className="size-4 rounded-sm border border-grayA-6 shrink-0">
          <AvatarFallback name={workspace.name} variant="marble" square />
        </Avatar>
      }
      label={workspace.name}
      href={href}
      items={items}
      currentId={workspace.orgId}
      searchPlaceholder="Find workspace..."
      emptyText="No workspaces found"
      footer={{ icon: Plus, label: "New workspace", href: routes.workspaces.create() }}
    />
  );
}
