import { InfoTooltip } from "@unkey/ui";
import { CardFooter } from "./components/card-footer";
import { CardHeader } from "./components/card-header";
import { NodeWrapper } from "./node-wrapper/node-wrapper";
import { REGION_INFO, type RegionNode as RegionNodeType } from "./types";

type RegionNodeProps = {
  node: RegionNodeType;
};

export function RegionNode({ node }: RegionNodeProps) {
  const { flagCode, rps, cpu, memory, health, zones } = node.metadata;
  const regionInfo = REGION_INFO[flagCode];

  return (
    <NodeWrapper health={health}>
      <CardHeader
        icon={
          <InfoTooltip
            content={`AWS region ${node.label} (${regionInfo.location})`}
            variant="primary"
            className="px-2.5 py-1 rounded-[10px] bg-white dark:bg-blackA-12 text-xs z-30"
            position={{ align: "center", side: "top", sideOffset: 5 }}
          >
            <div className="border rounded-[10px] border-grayA-3 size-9 bg-grayA-3 flex items-center justify-center">
              <img src={`/images/flags/${flagCode}.svg`} alt={flagCode} className="size-4" />
            </div>
          </InfoTooltip>
        }
        title={node.label}
        subtitle={`${zones} availability ${zones === 1 ? "zone" : "zones"}`}
        health={health}
      />
      <CardFooter type="region" rps={rps} cpu={cpu} memory={memory} />
    </NodeWrapper>
  );
}
