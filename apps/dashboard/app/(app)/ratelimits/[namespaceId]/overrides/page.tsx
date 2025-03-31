import { clickhouse } from "@/lib/clickhouse";
import { NamespaceNavbar } from "../namespace-navbar";
import { getWorkspaceDetailsWithOverrides } from "../namespace.actions";
import { OverridesTable } from "./overrides-table";

export const dynamic = "force-dynamic";

export default async function OverridePage({
  params: { namespaceId },
}: {
  params: { namespaceId: string };
}) {
  const { namespace, workspace } = await getWorkspaceDetailsWithOverrides(namespaceId);

  const lastUsedTimes = namespace.overrides?.length
    ? await getLastUsedTimes(
        namespace.workspaceId,
        namespace.id,
        namespace.overrides.map((o) => o.identifier),
      )
    : {};

  return (
    <div>
      <NamespaceNavbar
        activePage={{
          href: `/ratelimits/${namespace.id}/overrides`,
          text: "Overrides",
        }}
        namespace={namespace}
        ratelimitNamespaces={workspace.ratelimitNamespaces}
      />

      <OverridesTable
        namespaceId={namespace.id}
        ratelimits={namespace.overrides}
        lastUsedTimes={lastUsedTimes}
      />
    </div>
  );
}

async function getLastUsedTimes(workspaceId: string, namespaceId: string, identifiers: string[]) {
  const results = await Promise.all(
    identifiers.map(async (identifier) => {
      const lastUsed = await clickhouse.ratelimits.latest({
        workspaceId,
        namespaceId,
        identifier: [identifier],
        limit: 1,
      });
      return {
        identifier,
        lastUsed: lastUsed.val?.at(0)?.time ?? null,
      };
    }),
  );

  return Object.fromEntries(results.map(({ identifier, lastUsed }) => [identifier, lastUsed]));
}
