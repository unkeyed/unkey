import { trpc } from "@/lib/trpc/client";
import { Layers3 } from "@unkey/icons";
import { CardFooter } from "./components/card-footer";
import { CardHeader } from "./components/card-header";
import { NodeWrapper } from "./node-wrapper/node-wrapper";
import type { InstanceNode as InstanceNodeType, SentinelNode as SentinelNodeType } from "./types";

type InstanceNodeProps = {
  node: InstanceNodeType;
  flagCode: SentinelNodeType["metadata"]["flagCode"];
  deploymentId?: string;
};

export function InstanceNode({ node, flagCode, deploymentId }: InstanceNodeProps) {
  const { cpu, memory, health } = node.metadata;

  const { data: rps } = trpc.deploy.network.getInstanceRps.useQuery(
    {
      instanceId: node.id,
    },
    {
      enabled: Boolean(deploymentId),
      refetchInterval: 5000,
    },
  );

  return (
    <NodeWrapper health={health}>
      <CardHeader
        type="instance"
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
