import { RegionFlag } from "@/components/region-flag";
import { regionInfo } from "@/lib/regions";
import { trpc } from "@/lib/trpc/client";
import { InfoTooltip } from "@unkey/ui";
import { CardFooter } from "./components/card-footer";
import { CardHeader } from "./components/card-header";
import { NodeWrapper } from "./node-wrapper/node-wrapper";
import type { RegionNode as RegionNodeType } from "./types";

type RegionNodeProps = {
  node: RegionNodeType;
  deploymentId?: string;
};

export function RegionNode({ node, deploymentId }: RegionNodeProps) {
  const { health, instances } = node.metadata;

  // node.label is the region's name as it appears on
  // frontline_requests_raw_v1.region, so we can filter ClickHouse by it
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
            content={regionInfo(node.label).city}
            className="z-30"
            position={{ align: "center", side: "top", sideOffset: 5 }}
          >
            <RegionFlag region={node.label} size="md" shape="rounded" />
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
