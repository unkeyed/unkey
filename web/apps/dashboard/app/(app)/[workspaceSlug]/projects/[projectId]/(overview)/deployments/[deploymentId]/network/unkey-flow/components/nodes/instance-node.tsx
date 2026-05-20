import { LastExitBadge } from "@/app/(app)/[workspaceSlug]/projects/[projectId]/components/active-deployment-card";
import { trpc } from "@/lib/trpc/client";
import { Layers3 } from "@unkey/icons";
import { CardFooter } from "./components/card-footer";
import { CardHeader } from "./components/card-header";
import { NodeWrapper } from "./node-wrapper/node-wrapper";
import type { InstanceNode as InstanceNodeType, RegionNode as RegionNodeType } from "./types";

type InstanceNodeProps = {
  node: InstanceNodeType;
  flagCode: RegionNodeType["metadata"]["flagCode"];
  deploymentId?: string;
};

// RECENT_CRASH_WINDOW_MS bounds how long a stale `lastExit` keeps the
// instance card painted red. Without this, a one-off crash from hours ago
// would mark a since-recovered pod as unhealthy forever (we never clear
// last_exit_* on a successful run; ctrl only updates on newer events).
// 30 minutes is wide enough to surface flapping pods that briefly stabilise
// between restarts but narrow enough that a card recovering after a fix
// returns to its normal styling within one product feedback loop.
const RECENT_CRASH_WINDOW_MS = 30 * 60 * 1000;

export function InstanceNode({ node, flagCode, deploymentId }: InstanceNodeProps) {
  const { cpu, memory, health, lastExit } = node.metadata;

  const { data: rps } = trpc.deploy.network.getInstanceRps.useQuery(
    {
      instanceId: node.id,
    },
    {
      enabled: Boolean(deploymentId),
      refetchInterval: 5000,
    },
  );

  // Promote the card to "unhealthy" styling whenever there's something
  // actively wrong. Two triggers:
  //   - statusReason === "CrashLoopBackOff" — kubelet is currently
  //     throttling restarts; ALWAYS unhealthy regardless of finishedAt
  //   - finishedAt within the recent window — there was a crash
  //     recently enough that the user still cares
  // Otherwise, defer to the kubelet-derived health (running pod with
  // an old crash 6h ago should look normal).
  const isInActiveCrashloop = lastExit?.statusReason === "CrashLoopBackOff";
  const isRecentCrash =
    lastExit?.finishedAt != null && Date.now() - lastExit.finishedAt < RECENT_CRASH_WINDOW_MS;
  const effectiveHealth = isInActiveCrashloop || isRecentCrash ? "unhealthy" : health;

  // Replace the static "Instance Replica" subtitle with the exit badge
  // when ctrl has recorded a crash. The diagnostic should be readable
  // directly on the graph card — clicking into the panel shouldn't be
  // required to see "this pod is OOMKilling".
  const subtitle = lastExit ? <LastExitBadge lastExit={lastExit} /> : "Instance Replica";

  return (
    <NodeWrapper health={effectiveHealth}>
      <CardHeader
        type="instance"
        icon={
          <div className="border rounded-[10px] size-9 flex items-center justify-center border-grayA-5 bg-grayA-2">
            <Layers3 iconSize="lg-medium" className="text-gray-11" />
          </div>
        }
        title={node.label}
        subtitle={subtitle}
        health={effectiveHealth}
      />
      <CardFooter type="instance" flagCode={flagCode} rps={rps} cpu={cpu} memory={memory} />
    </NodeWrapper>
  );
}
