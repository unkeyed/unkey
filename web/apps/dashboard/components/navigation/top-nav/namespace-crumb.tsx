"use client";

import { useWorkspaceNavigation } from "@/hooks/use-workspace-navigation";
import { routes } from "@/lib/navigation/routes";
import { useNamespace, useNamespaces } from "@/lib/queries/ratelimit-namespaces";
import { IconGaugeOutline18, IconPlusOutline18 } from "@unkey/icons";
import { Crumb } from "./crumb";
import type { CrumbPopoverItem } from "./crumb-popover";

export function NamespaceCrumb({
  namespaceId,
  projectId,
}: { namespaceId: string; projectId?: string }) {
  const workspace = useWorkspaceNavigation();
  const currentQuery = useNamespace(namespaceId);
  const { namespaces } = useNamespaces({ search: "" });
  const current = currentQuery.data;
  const loading = currentQuery.isLoading;

  const items: CrumbPopoverItem[] = namespaces.map((n) => ({
    id: n.id,
    label: n.name,
    href: routes.ratelimits.detail({ workspaceSlug: workspace.slug, projectId, namespaceId: n.id }),
  }));

  return (
    <Crumb
      icon={<IconGaugeOutline18 className="size-3.5 text-gray-11" />}
      label={current?.name ?? namespaceId}
      loading={loading}
      href={routes.ratelimits.detail({ workspaceSlug: workspace.slug, projectId, namespaceId })}
      items={items}
      currentId={namespaceId}
      searchPlaceholder="Find namespace..."
      emptyText="No namespaces found"
      footer={{
        icon: IconPlusOutline18,
        label: "All namespaces",
        href: routes.ratelimits.list({ workspaceSlug: workspace.slug, projectId }),
      }}
    />
  );
}
