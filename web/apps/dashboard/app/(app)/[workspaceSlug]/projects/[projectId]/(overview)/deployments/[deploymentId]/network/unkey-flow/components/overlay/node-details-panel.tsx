import { RegionFlag } from "@/app/(app)/[workspaceSlug]/projects/[projectId]/components/region-flag";
import { Layers3 } from "@unkey/icons";
import { InfoTooltip } from "@unkey/ui";
import { cn } from "@unkey/ui/src/lib/utils";
import { useDeployment } from "../../../../layout-provider";
import {
  type DeploymentNode,
  type InstanceNode,
  REGION_INFO,
  type SentinelNode,
  isInstanceNode,
  isOriginNode,
  isSentinelNode,
  isSkeletonNode,
} from "../nodes/types";
import { NodeDetailsPanelHeader } from "./node-details-panel/components/header";
import { ResourceMetrics } from "./node-details-panel/components/resource-metrics";

const SentinelNodeDetails = ({
  node,
  onClose,
}: {
  node: SentinelNode;
  onClose: () => void;
}) => {
  const { flagCode, health } = node.metadata;
  const regionInfo = REGION_INFO[flagCode];

  return (
    <>
      <NodeDetailsPanelHeader
        onClose={onClose}
        subSection={{
          type: "sentinel",
          variant: "panel",
          icon: (
            <InfoTooltip
              content={`${regionInfo.name} (${regionInfo.location})`}
              variant="primary"
              className="px-2.5 py-1 rounded-[10px] bg-white dark:bg-blackA-12 text-xs z-30"
              position={{ align: "center", side: "top", sideOffset: 5 }}
            >
              <RegionFlag flagCode={flagCode} size="md" shape="rounded" />
            </InfoTooltip>
          ),
          title: node.label,
          subtitle: "Sentinel",
          health,
        }}
      />
      <ResourceMetrics resourceType="sentinel" resourceId={node.id} />
    </>
  );
};

type InstanceNodeDetailsProps = {
  node: InstanceNode;
  deploymentId: string;
  onClose: () => void;
};

const InstanceNodeDetails = ({ node, deploymentId, onClose }: InstanceNodeDetailsProps) => {
  const { health } = node.metadata;
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
      <ResourceMetrics
        resourceType="deployment"
        resourceId={deploymentId}
        storageMib={deployment.storageMib}
        instanceName={node.metadata.k8sName}
      />
    </>
  );
};

type Props = {
  node: DeploymentNode | null;
  deploymentId: string;
  onClose: () => void;
};

export function NodeDetailsPanel({ node, deploymentId, onClose }: Props) {
  if (!node) {
    return null;
  }

  const renderDetails = () => {
    if (isSkeletonNode(node) || isOriginNode(node)) {
      return null;
    }
    if (isSentinelNode(node)) {
      return <SentinelNodeDetails node={node} onClose={onClose} />;
    }
    if (isInstanceNode(node)) {
      return <InstanceNodeDetails node={node} deploymentId={deploymentId} onClose={onClose} />;
    }
    const _exhaustive: never = node;
    return _exhaustive;
  };

  const content = renderDetails();
  if (!content) {
    return null;
  }

  const isOpen = Boolean(node?.id);

  return (
    <div className="fixed top-40 right-0 bottom-14 z-20 flex flex-col gap-2 pointer-events-none">
      <div
        className={cn(
          // Fixed width so the panel doesn't resize when chart content
          // (Y-axis tick labels, loading skeleton, empty state) changes
          // between windows. `min-w-[360px]` let child content push the
          // panel wider on re-render, which the user perceives as a
          // flicker every time the window selector changes.
          "rounded-l-xl bg-white dark:bg-black border-l border-t border-b border-grayA-4 shadow-[0_2px_8px_-2px_rgba(0,0,0,0.1)] pointer-events-auto w-[400px] max-h-[calc(100vh-300px)] flex flex-col pb-6",
          "transition-all duration-300 ease-out",
          isOpen ? "opacity-100 translate-y-0" : "opacity-0 -translate-y-2 pointer-events-none",
        )}
      >
        <div className="flex flex-col items-stretch overflow-y-auto max-h-full pb-4">{content}</div>
      </div>
    </div>
  );
}
