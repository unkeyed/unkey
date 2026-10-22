import { LastExitCard } from "@/app/(app)/[workspaceSlug]/projects/[projectId]/apps/[appId]/components/last-exit";
import { summarizeExit } from "@/lib/collections/deploy/exit-reasons";
import { IconLayers3Outline18 } from "@unkey/icons";
import { SlidePanel, SlidePanelContent } from "@unkey/ui";
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
            <div className="border rounded-xl size-9 flex items-center justify-center bg-grayA-2">
              <IconLayers3Outline18 className="size-4 text-gray-11" />
            </div>
          ),
          title: node.label,
          subtitle: "Instance replica",
          health,
        }}
      />
      {lastExit && (
        <div className="mt-5 w-full px-4">
          <LastExitCard
            summary={summarizeExit(lastExit)}
            restartCount={lastExit.restartCount}
            finishedAt={lastExit.finishedAt}
          />
        </div>
      )}
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
    <SlidePanel
      isOpen={isOpen}
      onClose={onClose}
      side="right"
      widthClassName="w-[600px]"
      backdrop="none"
      fitContent
    >
      <SlidePanelContent className="overflow-y-auto pb-6">
        {node && isInstanceNode(node) && (
          <InstanceNodeDetails node={node} deploymentId={deploymentId} onClose={onClose} />
        )}
      </SlidePanelContent>
    </SlidePanel>
  );
}
