import { Layers3 } from "@unkey/icons";
import { SlidePanel } from "@unkey/ui";
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
  const isOpen = Boolean(node?.id) && node !== null && isInstanceNode(node);

  return (
    <SlidePanel.Root
      isOpen={isOpen}
      onClose={onClose}
      side="right"
      widthClassName="w-[600px]"
      backdrop="dim"
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
