import { RegionFlag } from "@/app/(app)/[workspaceSlug]/projects/[projectId]/apps/[appId]/components/region-flag";
import { InfoTooltip } from "@unkey/ui";
import { CardFooter } from "./components/card-footer";
import { CardHeader } from "./components/card-header";
import { NodeWrapper } from "./node-wrapper/node-wrapper";
import { REGION_INFO, type RegionNode as RegionNodeType } from "./types";

type RegionNodeProps = {
  node: RegionNodeType;
  rps?: number;
};

export function RegionNode({ node, rps }: RegionNodeProps) {
  const { flagCode, health, instances } = node.metadata;
  const regionInfo = REGION_INFO[flagCode];

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
