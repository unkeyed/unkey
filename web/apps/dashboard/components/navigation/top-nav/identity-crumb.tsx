"use client";

import { useWorkspaceNavigation } from "@/hooks/use-workspace-navigation";
import { trpc } from "@/lib/trpc/client";
import { Fingerprint, Plus } from "@unkey/icons";
import { Crumb } from "./crumb";
import type { CrumbPopoverItem } from "./crumb-popover";

export function IdentityCrumb({ identityId }: { identityId: string }) {
  const workspace = useWorkspaceNavigation();
  const { data } = trpc.identity.query.useQuery({});

  const identities = data?.identities ?? [];

  const items: CrumbPopoverItem[] = identities.map((identity) => ({
    id: identity.id,
    label: identity.id,
    href: `/${workspace.slug}/identities/${identity.id}`,
  }));

  return (
    <Crumb
      icon={<Fingerprint className="size-3.5 text-accent-11" iconSize="sm-regular" />}
      label={identityId}
      href={`/${workspace.slug}/identities/${identityId}`}
      items={items}
      currentId={identityId}
      searchPlaceholder="Find identity..."
      emptyText="No identities found"
      footer={{
        icon: Plus,
        label: "All identities",
        href: `/${workspace.slug}/identities`,
      }}
    />
  );
}
