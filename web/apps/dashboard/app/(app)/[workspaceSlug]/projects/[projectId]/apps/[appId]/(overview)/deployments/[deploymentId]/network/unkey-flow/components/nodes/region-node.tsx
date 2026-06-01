import { RegionFlag } from "@/app/(app)/[workspaceSlug]/projects/[projectId]/apps/[appId]/components/region-flag";
import { trpc } from "@/lib/trpc/client";
import { InfoTooltip } from "@unkey/ui";
import { CardFooter } from "./components/card-footer";
import { CardHeader } from "./components/card-header";
import { NodeWrapper } from "./node-wrapper/node-wrapper";
import { REGION_INFO, type RegionNode as RegionNodeType } from "./types";

type RegionNodeProps = {
  node: RegionNodeType;
  deploymentId?: string;
};

export function RegionNode({ node, deploymentId }: RegionNodeProps) {
  const { flagCode, health, instances } = node.metadata;
  const regionInfo = REGION_INFO[flagCode];

  // node.label is the region's name as it appears on
  // sentinel_requests_raw_v1.region, so we can filter ClickHouse by it
  // directly without an extra DB lookup.
  const { data: rps } = trpc.deploy.network.getRegionRps.useQuery(
    {
      deploymentId: deploymentId ?? "",
      region: node.label,
    },
    {
      enabled: Boolean(deploymentId),
      refetchInterval: 5000,
    },
  );

  const instanceText =
    instances === 0
      ? "No running instances"
      : `${instances} ${instances === 1 ? "instance" : "instances"}`;

  return (
    <NodeWrapper health={health}>
      <CardHeader
        type="region"
        icon={
          <InfoTooltip
            content={`${regionInfo.name} (${regionInfo.location})`}
            variant="primary"
            className="px-2.5 py-1 rounded-[10px] bg-white dark:bg-blackA-12 text-xs z-30"
            position={{ align: "center", side: "top", sideOffset: 5 }}
          >
            <RegionFlag flagCode={flagCode} size="md" shape="rounded" />
          </InfoTooltip>
        }
        title={node.label}
        subtitle={instanceText}
        health={health}
      />
      <CardFooter type="region" rps={rps} />
    </NodeWrapper>
  );
}
