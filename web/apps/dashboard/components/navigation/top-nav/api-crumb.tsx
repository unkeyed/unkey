"use client";

import { useWorkspaceNavigation } from "@/hooks/use-workspace-navigation";
import { trpc } from "@/lib/trpc/client";
import { Nodes, Plus } from "@unkey/icons";
import { Crumb } from "./crumb";
import type { CrumbPopoverItem } from "./crumb-popover";

export function ApiCrumb({ apiId }: { apiId: string }) {
  const workspace = useWorkspaceNavigation();
  const { data } = trpc.api.queryApiKeyDetails.useQuery({ apiId }, { enabled: !!apiId });

  const apiName = data?.currentApi?.name ?? apiId;
  const siblings = data?.workspaceApis ?? [];
  const loading = !data;

  const items: CrumbPopoverItem[] = siblings.map((api) => ({
    id: api.id,
    label: api.name,
    href: `/${workspace.slug}/apis/${api.id}`,
  }));

  return (
    <Crumb
      icon={<Nodes className="size-3.5 text-accent-11" iconSize="sm-regular" />}
      label={apiName}
      loading={loading}
      href={`/${workspace.slug}/apis/${apiId}`}
      items={items}
      currentId={apiId}
      searchPlaceholder="Find API..."
      emptyText="No APIs found"
      footer={{
        icon: Plus,
        label: "All Keyspaces (APIs)",
        href: `/${workspace.slug}/apis`,
      }}
    />
  );
}
