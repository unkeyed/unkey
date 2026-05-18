"use client";

import { useWorkspaceNavigation } from "@/hooks/use-workspace-navigation";
import { collection } from "@/lib/collections";
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
  const loading = !current && namespaces.length === 0;

  const items: CrumbPopoverItem[] = namespaces.map((n) => ({
    id: n.id,
    label: n.name,
    href: `/${workspace.slug}/ratelimits/${n.id}`,
  }));

  return (
    <Crumb
      icon={<Gauge className="size-3.5 text-accent-11" iconSize="sm-regular" />}
      label={current?.name ?? namespaceId}
      loading={loading}
      href={`/${workspace.slug}/ratelimits/${namespaceId}`}
      items={items}
      currentId={namespaceId}
      searchPlaceholder="Find namespace..."
      emptyText="No namespaces found"
      footer={{
        icon: Plus,
        label: "All namespaces",
        href: `/${workspace.slug}/ratelimits`,
      }}
    />
  );
}
