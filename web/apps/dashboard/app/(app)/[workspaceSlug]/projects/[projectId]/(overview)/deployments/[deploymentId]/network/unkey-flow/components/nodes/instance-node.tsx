import { Layers3 } from "@unkey/icons";
import { CardFooter } from "./components/card-footer";
import { CardHeader } from "./components/card-header";
import { NodeWrapper } from "./node-wrapper/node-wrapper";
import type { InstanceNode as InstanceNodeType, SentinelNode as SentinelNodeType } from "./types";

type InstanceNodeProps = {
  node: InstanceNodeType;
  flagCode: SentinelNodeType["metadata"]["flagCode"];
};

export function InstanceNode({ node, flagCode }: InstanceNodeProps) {
  const { rps, cpu, memory, health } = node.metadata;

  return (
    <NodeWrapper health={health}>
      <CardHeader
        icon={
          <div className="border rounded-[10px] size-9 flex items-center justify-center border-grayA-5 bg-grayA-2">
            <Layers3 iconSize="lg-medium" className="text-gray-11" />
          </div>
        }
        title={node.label}
        subtitle="Instance Replica"
        health={health}
      />
      <CardFooter type="instance" flagCode={flagCode} rps={rps} cpu={cpu} memory={memory} />
    </NodeWrapper>
  );
}
