import { Layers3 } from "@unkey/icons";
import { cn } from "@unkey/ui/src/lib/utils";
import { useDeployment } from "../../../../layout-provider";
import {
  type DeploymentNode,
  type InstanceNode,
  isInstanceNode,
  isOriginNode,
  isRegionNode,
  isSkeletonNode,
} from "../nodes/types";
import { NodeDetailsPanelHeader } from "./node-details-panel/components/header";
import { ResourceMetrics } from "./node-details-panel/components/resource-metrics";

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
    if (isSkeletonNode(node) || isOriginNode(node) || isRegionNode(node)) {
      return null;
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
