import { InfoTooltip } from "@unkey/ui";
import { CardFooter } from "./components/card-footer";
import { CardHeader } from "./components/card-header";
import { NodeWrapper } from "./node-wrapper/node-wrapper";
import { REGION_INFO, type SentinelNode as SentinelNodeType } from "./types";

type SentinelNodeProps = {
  node: SentinelNodeType;
};

export function SentinelNode({ node }: SentinelNodeProps) {
  const { flagCode, rps, cpu, memory, health } = node.metadata;
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
        subtitle="Sentinel"
        health={health}
      />
      <CardFooter type="sentinel" rps={rps} cpu={cpu} memory={memory} />
    </NodeWrapper>
  );
}
