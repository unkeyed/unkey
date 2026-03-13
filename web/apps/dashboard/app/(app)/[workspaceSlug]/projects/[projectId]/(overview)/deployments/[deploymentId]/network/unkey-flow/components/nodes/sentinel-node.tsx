import { RegionFlag } from "@/app/(app)/[workspaceSlug]/projects/[projectId]/components/region-flag";
import { trpc } from "@/lib/trpc/client";
import { InfoTooltip } from "@unkey/ui";
import { CardFooter } from "./components/card-footer";
import { CardHeader } from "./components/card-header";
import { NodeWrapper } from "./node-wrapper/node-wrapper";
import { REGION_INFO, type SentinelNode as SentinelNodeType } from "./types";

type SentinelNodeProps = {
  node: SentinelNodeType;
  deploymentId?: string;
  isCollapsed?: boolean;
  onToggleCollapse?: () => void;
};

export function SentinelNode({ node, deploymentId, isCollapsed, onToggleCollapse }: SentinelNodeProps) {
  const { flagCode, cpu, memory, health, replicas, instances } = node.metadata;

  const { data: rps } = trpc.deploy.network.getSentinelRps.useQuery(
    {
      sentinelId: node.id,
    },
    {
      enabled: Boolean(deploymentId),
      refetchInterval: 5000,
    },
  );
  const regionInfo = REGION_INFO[flagCode];

  const replicaText =
    replicas === 0
      ? "No available replicas"
      : `${replicas} available ${replicas === 1 ? "replica" : "replicas"}`;

  return (
    <div className="relative flex flex-col items-center">
      <NodeWrapper health={health}>
        <CardHeader
          type="sentinel"
          icon={
            <InfoTooltip
              content={`AWS region ${node.label} (${regionInfo.location})`}
              variant="primary"
              className="px-2.5 py-1 rounded-[10px] bg-white dark:bg-blackA-12 text-xs z-30"
              position={{ align: "center", side: "top", sideOffset: 5 }}
            >
              <RegionFlag flagCode={flagCode} size="md" shape="rounded" />
            </InfoTooltip>
          }
          title={node.label}
          subtitle={replicaText}
          health={health}
        />
        <CardFooter type="sentinel" rps={rps} cpu={cpu} memory={memory} />
      </NodeWrapper>

      {instances > 3 && onToggleCollapse && (
        <button
          onClick={onToggleCollapse}
          className="mt-1 text-[11px] text-gray-9 hover:text-gray-11 transition-colors px-2 py-0.5 rounded-md hover:bg-grayA-3"
          type="button"
        >
          {isCollapsed ? "Show instances" : "Hide instances"}
        </button>
      )}
    </div>
  );
}
