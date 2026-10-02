"use client";

import { useWorkspaceNavigation } from "@/hooks/use-workspace-navigation";
import { useIdentities } from "@/lib/identities-query";
import { routes } from "@/lib/navigation/routes";
import { IconFingerprintOutline18, IconPlusOutline18 } from "@unkey/icons";
import { Crumb } from "./crumb";
import type { CrumbPopoverItem } from "./crumb-popover";

export function IdentityCrumb({
  identityId,
  projectId,
}: { identityId: string; projectId?: string }) {
  const workspace = useWorkspaceNavigation();
  const { identities } = useIdentities();

  const items: CrumbPopoverItem[] = identities.slice(0, 50).map((identity) => ({
    id: identity.id,
    label: identity.id,
    href: routes.identities.detail({
      workspaceSlug: workspace.slug,
      projectId,
      identityId: identity.id,
    }),
  }));

  return (
    <Crumb
      icon={<IconFingerprintOutline18 className="size-3.5 text-gray-11" />}
      label={identityId}
      href={routes.identities.detail({ workspaceSlug: workspace.slug, projectId, identityId })}
      items={items}
      currentId={identityId}
      searchPlaceholder="Find identity..."
      emptyText="No identities found"
      footer={{
        icon: IconPlusOutline18,
        label: "All identities",
        href: routes.identities.list({ workspaceSlug: workspace.slug, projectId }),
      }}
    />
  );
}
