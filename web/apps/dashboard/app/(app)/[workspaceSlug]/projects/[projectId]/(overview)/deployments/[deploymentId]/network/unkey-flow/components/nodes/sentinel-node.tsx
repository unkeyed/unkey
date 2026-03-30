import { RegionFlag } from "@/app/(app)/[workspaceSlug]/projects/[projectId]/components/region-flag";
import { trpc } from "@/lib/trpc/client";
import { ChevronDown, ChevronUp, Layers3 } from "@unkey/icons";
import { InfoTooltip } from "@unkey/ui";
import { CardFooter } from "./components/card-footer";
import { CardHeader } from "./components/card-header";
import { NodeWrapper } from "./node-wrapper/node-wrapper";
import { COLLAPSE_THRESHOLD, REGION_INFO, type SentinelNode as SentinelNodeType } from "./types";

type SentinelNodeProps = {
  node: SentinelNodeType;
  deploymentId?: string;
  isCollapsed?: boolean;
  onToggleCollapse?: () => void;
};

export function SentinelNode({
  node,
  deploymentId,
  isCollapsed,
  onToggleCollapse,
}: SentinelNodeProps) {
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
    <div className="relative flex flex-col items-end">
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

      {instances > COLLAPSE_THRESHOLD && onToggleCollapse && (
        <button
          onClick={onToggleCollapse}
          className="mt-2 mb-5 flex items-center gap-1.5 border border-grayA-4 bg-grayA-2 hover:bg-grayA-3 pl-2 pr-2.5 py-1 rounded-full text-xs text-gray-11 font-medium transition-colors shadow-sm"
          type="button"
        >
          <Layers3 iconSize="sm-regular" className="text-gray-9" />
          <span>{isCollapsed ? `Show ${instances}` : "Hide"}</span> instances
          {isCollapsed ? (
            <ChevronDown iconSize="sm-regular" className="text-gray-9" />
          ) : (
            <ChevronUp iconSize="sm-regular" className="text-gray-9" />
          )}
        </button>
      )}
    </div>
  );
}
