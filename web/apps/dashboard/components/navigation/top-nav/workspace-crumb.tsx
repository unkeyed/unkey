"use client";

import { Avatar, AvatarFallback } from "@/components/ui/avatar";
import { useWorkspaceNavigation } from "@/hooks/use-workspace-navigation";
import { setLastUsedOrgCookie, setSessionCookie } from "@/lib/auth/cookies-actions";
import { routes } from "@/lib/navigation/routes";
import { trpc } from "@/lib/trpc/client";
import { Plus } from "@unkey/icons";
import { toast } from "@unkey/ui";
import { useMemo } from "react";
import { Crumb } from "./crumb";
import type { CrumbPopoverItem } from "./crumb-popover";

export function WorkspaceCrumb({ href }: { href: string }) {
  const workspace = useWorkspaceNavigation();
  const { data: user } = trpc.user.getCurrentUser.useQuery();
  const { data: memberships } = trpc.user.listMemberships.useQuery(user?.id ?? "", {
    enabled: !!user?.id,
  });
  const orgs = memberships?.data ?? [];

  const switchOrg = trpc.user.switchOrg.useMutation({
    async onSuccess(sessionData, orgId) {
      if (!sessionData.token || !sessionData.expiresAt) {
        toast.error("Failed to switch workspace. Invalid session data.");
        return;
      }
      try {
        await setSessionCookie({
          token: sessionData.token,
          expiresAt: sessionData.expiresAt,
        });
      } catch {
        toast.error("Failed to complete workspace switch. Please try again.");
        return;
      }
      try {
        await setLastUsedOrgCookie({ orgId });
      } catch {}
      // Full reload re-fetches the new org's workspace + permissions; a
      // soft router.refresh() leaves stale providers tied to the old org.
      window.location.replace(routes.workspaces.root());
    },
    onError() {
      toast.error("Failed to switch workspace. Contact support if error persists.");
    },
  });

  const switchOrgMutate = switchOrg.mutate;
  const switchOrgLoading = switchOrg.isLoading;
  const items: CrumbPopoverItem[] = useMemo(
    () =>
      orgs.map((m) => ({
        id: m.organization.id,
        label: m.organization.name,
        onClick: () => {
          if (m.organization.id !== workspace.orgId && !switchOrgLoading) {
            switchOrgMutate(m.organization.id);
          }
        },
      })),
    [orgs, switchOrgMutate, switchOrgLoading, workspace.orgId],
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
