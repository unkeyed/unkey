"use client";

import { IconNodesOutline18, IconPlusOutline18 } from "nucleo-ui-outline-18";
import { useApiName } from "@/hooks/use-api-name";
import { useWorkspaceNavigation } from "@/hooks/use-workspace-navigation";
import { routes } from "@/lib/navigation/routes";
import { trpc } from "@/lib/trpc/client";
import { Crumb } from "./crumb";
import type { CrumbPopoverItem } from "./crumb-popover";

export function ApiCrumb({ apiId }: { apiId: string }) {
  const workspace = useWorkspaceNavigation();
  const { name, isLoading } = useApiName(apiId);
  const { data } = trpc.api.queryApiKeyDetails.useQuery({ apiId }, { enabled: !!apiId });

  const siblings = data?.workspaceApis ?? [];

  const items: CrumbPopoverItem[] = siblings.map((api) => ({
    id: api.id,
    label: api.name,
    href: routes.apis.detail({ workspaceSlug: workspace.slug, apiId: api.id }),
  }));

  return (
    <Crumb
      icon={<IconNodesOutline18 className="size-3.5 text-accent-11" />}
      label={name ?? apiId}
      loading={isLoading}
      href={routes.apis.detail({ workspaceSlug: workspace.slug, apiId })}
      items={items}
      currentId={apiId}
      searchPlaceholder="Find keyspace..."
      emptyText="No keyspaces found"
      footer={{
        icon: IconPlusOutline18,
        label: "All Keyspaces (APIs)",
        href: routes.apis.list({ workspaceSlug: workspace.slug }),
      }}
    />
  );
}
