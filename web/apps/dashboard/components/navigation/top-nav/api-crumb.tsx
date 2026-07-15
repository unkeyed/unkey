"use client";

import { useWorkspaceNavigation } from "@/hooks/use-workspace-navigation";
import { routes } from "@/lib/navigation/routes";
import { trpc } from "@/lib/trpc/client";
import { getUnkeyClient } from "@/lib/unkey-client";
import { useQuery } from "@tanstack/react-query";
import { Nodes, Plus } from "@unkey/icons";
import { Crumb } from "./crumb";
import type { CrumbPopoverItem } from "./crumb-popover";

export function ApiCrumb({ apiId }: { apiId: string }) {
  const workspace = useWorkspaceNavigation();
  const { data } = trpc.api.queryApiKeyDetails.useQuery({ apiId }, { enabled: !!apiId });
  const { data: proxiedApi } = useQuery({
    queryKey: ["dashboard-api-proxy", "apis.getApi", apiId],
    enabled: Boolean(apiId),
    queryFn: async () => {
      const response = await getUnkeyClient().apis.getApi({ apiId });
      return response.data;
    },
  });

  const apiName = proxiedApi?.name ?? data?.currentApi?.name ?? apiId;
  const siblings = data?.workspaceApis ?? [];
  const loading = !proxiedApi && !data;

  const items: CrumbPopoverItem[] = siblings.map((api) => ({
    id: api.id,
    label: api.name,
    href: routes.apis.detail({ workspaceSlug: workspace.slug, apiId: api.id }),
  }));

  return (
    <Crumb
      icon={<Nodes className="size-3.5 text-accent-11" iconSize="sm-regular" />}
      label={apiName}
      loading={loading}
      href={routes.apis.detail({ workspaceSlug: workspace.slug, apiId })}
      items={items}
      currentId={apiId}
      searchPlaceholder="Find keyspace..."
      emptyText="No keyspaces found"
      footer={{
        icon: Plus,
        label: "All Keyspaces (APIs)",
        href: routes.apis.list({ workspaceSlug: workspace.slug }),
      }}
    />
  );
}
