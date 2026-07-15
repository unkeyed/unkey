"use client";

import { useWorkspaceNavigation } from "@/hooks/use-workspace-navigation";
import { collection } from "@/lib/collections";
import { routes } from "@/lib/navigation/routes";
import { useLiveQuery } from "@tanstack/react-db";
import { Gauge, Plus } from "@unkey/icons";
import { Crumb } from "./crumb";
import type { CrumbPopoverItem } from "./crumb-popover";

export function NamespaceCrumb({ namespaceId }: { namespaceId: string }) {
  const workspace = useWorkspaceNavigation();
  const namespacesQuery = useLiveQuery((q) =>
    q.from({ namespace: collection.ratelimitNamespaces }).select(({ namespace }) => ({
      id: namespace.id,
      name: namespace.name,
    })),
  );
  const namespaces = namespacesQuery.data ?? [];
  const current = namespaces.find((n) => n.id === namespaceId);
  const loading = namespacesQuery.isLoading;

  const items: CrumbPopoverItem[] = namespaces.map((n) => ({
    id: n.id,
    label: n.name,
    href: routes.ratelimits.detail({ workspaceSlug: workspace.slug, namespaceId: n.id }),
  }));

  return (
    <Crumb
      icon={<Gauge className="size-3.5 text-accent-11" iconSize="sm-regular" />}
      label={current?.name ?? namespaceId}
      loading={loading}
      href={routes.ratelimits.detail({ workspaceSlug: workspace.slug, namespaceId })}
      items={items}
      currentId={namespaceId}
      searchPlaceholder="Find namespace..."
      emptyText="No namespaces found"
      footer={{
        icon: Plus,
        label: "All namespaces",
        href: routes.ratelimits.list({ workspaceSlug: workspace.slug }),
      }}
    />
  );
}
