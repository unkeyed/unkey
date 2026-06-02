import { LastExitBadge } from "@/app/(app)/[workspaceSlug]/projects/[projectSlug]/apps/[appSlug]/components/active-deployment-card";
import { Layers3, TriangleWarning2 } from "@unkey/icons";
import { SlidePanel, TimestampInfo } from "@unkey/ui";
import { useDeployment } from "../../../../layout-provider";
import { type DeploymentNode, type InstanceNode, isInstanceNode } from "../nodes/types";
import { NodeDetailsPanelHeader } from "./node-details-panel/components/header";
import { ResourceMetrics } from "./node-details-panel/components/resource-metrics";

type InstanceNodeDetailsProps = {
  node: InstanceNode;
  deploymentId: string;
  onClose: () => void;
};

const InstanceNodeDetails = ({ node, deploymentId, onClose }: InstanceNodeDetailsProps) => {
  const { health, lastExit } = node.metadata;
  const { deployment } = useDeployment();

  return (
    <>
      <NodeDetailsPanelHeader
        onClose={onClose}
        subSection={{
          type: "instance",
          variant: "panel",
          icon: (
            <div className="border rounded-[10px] size-9 flex items-center justify-center border-grayA-5 bg-grayA-2">
              <Layers3 iconSize="lg-medium" className="text-gray-11" />
            </div>
          ),
          title: node.label,
          subtitle: "Instance replica",
          health,
        }}
      />
      {lastExit && <LastExitSection lastExit={lastExit} />}
      <ResourceMetrics
        resourceId={deploymentId}
        storageMib={deployment.storageMib}
        instanceName={node.metadata.k8sName}
      />
    </>
  );
};

// LastExitSection surfaces a crashlooping or recently-exited pod's reason
// at the top of the panel. The health banner alone reads "is starting up"
// for a CrashLoopBackOff pod on its 17th retry, which buries the actual
// failure. Layout mirrors the section rows in ResourceMetrics so the panel
// reads as one consistent column.
function LastExitSection({
  lastExit,
}: { lastExit: NonNullable<InstanceNode["metadata"]["lastExit"]> }) {
  return (
    <div className="flex flex-col gap-2 px-4 w-full mt-5">
      <div className="flex items-center gap-3 flex-wrap">
        <div className="bg-grayA-3 text-gray-12 rounded-md size-[22px] items-center flex justify-center">
          <TriangleWarning2 iconSize="sm-regular" className="shrink-0" />
        </div>
        <span className="text-gray-11 text-xs">Last exit</span>
        <div className="ml-auto">
          <LastExitBadge lastExit={lastExit} />
        </div>
      </div>
      <div className="flex items-baseline gap-1.5 text-[12px] tabular-nums text-grayA-9 ml-[34px]">
        <span>
          <span className="text-grayA-11">Restarts</span>{" "}
          <span className="text-gray-12 font-medium">{lastExit.restartCount}</span>
        </span>
        {lastExit.finishedAt && (
          <>
            <span>·</span>
            <TimestampInfo
              value={lastExit.finishedAt}
              displayType="relative"
              className="text-grayA-11"
            />
          </>
        )}
      </div>
    </div>
  );
}

type Props = {
  node: DeploymentNode | null;
  deploymentId: string;
  onClose: () => void;
};

export function NodeDetailsPanel({ node, deploymentId, onClose }: Props) {
  const isOpen = Boolean(node?.id) && node !== null && isInstanceNode(node);

  return (
    <SlidePanel.Root
      isOpen={isOpen}
      onClose={onClose}
      side="right"
      widthClassName="w-[600px]"
      backdrop={false}
      topOffset={140}
      fitContent
    >
      <SlidePanel.Content className="overflow-y-auto pb-6" stagger={false}>
        {node && isInstanceNode(node) && (
          <InstanceNodeDetails node={node} deploymentId={deploymentId} onClose={onClose} />
        )}
      </SlidePanel.Content>
    </SlidePanel.Root>
  );
}
