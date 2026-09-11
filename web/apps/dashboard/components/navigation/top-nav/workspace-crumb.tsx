"use client";

import { Avatar, AvatarFallback } from "@/components/ui/avatar";
import { useWorkspaceNavigation } from "@/hooks/use-workspace-navigation";
import { setLastUsedOrgCookie, setSessionCookie } from "@/lib/auth/cookies-actions";
import { routes } from "@/lib/navigation/routes";
import { trpc } from "@/lib/trpc/client";
import { Plus } from "@unkey/icons";
import { Button, toast } from "@unkey/ui";
import { useMemo } from "react";
import { Crumb } from "./crumb";
import type { CrumbPopoverItem } from "./crumb-popover";

export function WorkspaceCrumb({ href }: { href: string }) {
  const workspace = useWorkspaceNavigation();
  const available = trpc.workspace.listAvailable.useQuery();
  const orgs = available.isError ? [] : (available.data ?? []);

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
        id: m.orgId,
        label: m.name,
        onClick: () => {
          if (m.orgId !== workspace.orgId && !switchOrgLoading) {
            switchOrgMutate(m.orgId);
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
      footer={{ icon: Plus, label: "New workspace", href: routes.workspaces.create() }}
    />
  );
}
