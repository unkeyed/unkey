"use client";

import { useWorkspaceNavigation } from "@/hooks/use-workspace-navigation";
import { useIdentities } from "@/lib/identities-query";
import { routes } from "@/lib/navigation/routes";
import { Fingerprint, Plus } from "@unkey/icons";
import { Crumb } from "./crumb";
import type { CrumbPopoverItem } from "./crumb-popover";

export function IdentityCrumb({ identityId }: { identityId: string }) {
  const workspace = useWorkspaceNavigation();
  const { identities } = useIdentities();

  const items: CrumbPopoverItem[] = identities.slice(0, 50).map((identity) => ({
    id: identity.id,
    label: identity.id,
    href: routes.identities.detail({ workspaceSlug: workspace.slug, identityId: identity.id }),
  }));

  return (
    <Crumb
      icon={<Fingerprint className="size-3.5 text-accent-11" iconSize="sm-regular" />}
      label={identityId}
      href={routes.identities.detail({ workspaceSlug: workspace.slug, identityId })}
      items={items}
      currentId={identityId}
      searchPlaceholder="Find identity..."
      emptyText="No identities found"
      footer={{
        icon: Plus,
        label: "All identities",
        href: routes.identities.list({ workspaceSlug: workspace.slug }),
      }}
    />
  );
}
