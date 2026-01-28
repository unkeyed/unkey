import { ChartActivity, Dots, Layers3 } from "@unkey/icons";
import { Button, InfoTooltip } from "@unkey/ui";
import { cn } from "@unkey/ui/src/lib/utils";
import {
  type DeploymentNode,
  REGION_INFO,
  type RegionNode,
  type SentinelNode,
  isOriginNode,
  isRegionNode,
  isSentinelNode,
  isSkeletonNode,
} from "../nodes/types";
import { NodeDetailsPanelHeader } from "./node-details-panel/components/header";
import { Metrics } from "./node-details-panel/components/metrics";
import { SettingsSection } from "./node-details-panel/components/settings-row";
import { metrics } from "./node-details-panel/constants";
import { SentinelInstances } from "./node-details-panel/region-node/sentinel-instances";

const RegionNodeDetails = ({
  node,
  onClose,
}: {
  node: RegionNode;
  onClose: () => void;
}) => {
  const { flagCode, zones, health } = node.metadata;
  const regionInfo = REGION_INFO[flagCode];

  return (
    <>
      <NodeDetailsPanelHeader
        onClose={onClose}
        subSection={{
          variant: "panel",
          icon: (
            <InfoTooltip
              content={`AWS region ${node.label} (${regionInfo.location})`}
              variant="primary"
              className="px-2.5 py-1 rounded-[10px] bg-white dark:bg-blackA-12 text-xs z-30"
              position={{ align: "center", side: "top", sideOffset: 5 }}
            >
              <div className="border rounded-[10px] border-grayA-3 size-12 bg-grayA-3 flex items-center justify-center">
                <img src={`/images/flags/${flagCode}.svg`} alt={flagCode} className="size-[22px]" />
              </div>
            </InfoTooltip>
          ),
          title: node.label,
          subtitle: `${zones} availability ${zones === 1 ? "zone" : "zones"}`,
          health,
        }}
      />
      <Metrics metrics={metrics} />
      <SentinelInstances instances={node.children ?? []} />
      <SettingsSection
        title="Scaling Configuration"
        settings={[
          {
            label: "Scaling",
            value: (
              <div className="text-grayA-10">
                <div>
                  <span className="text-gray-12 font-medium">3</span> to{" "}
                  <span className="text-gray-12 font-medium">6</span> instances
                </div>
                <div className="mt-0.5">
                  at <span className="text-gray-12 font-medium">70%</span> CPU threshold
                </div>
              </div>
            ),
            icon: (
              <ChartActivity className="size-[14px] text-gray-12 shrink-0" iconSize="md-regular" />
            ),
          },
        ]}
      />
      <SettingsSection
        title="Regional settings"
        settings={[
          { label: "Provider", value: "AWS" },
          { label: "Region code", value: node.label },
          { label: "Availability zones", value: zones },
          {
            label: "Image",
            value: (
              <>
                unkey:<span className="text-grayA-10 font-normal">latest</span>
              </>
            ),
          },
        ]}
      />
    </>
  );
};

type SentinelNodeDetailsProps = {
  node: SentinelNode;
  onClose: () => void;
};

const SentinelNodeDetails = ({ node, onClose }: SentinelNodeDetailsProps) => {
  const { health } = node.metadata;

  return (
    <>
      <NodeDetailsPanelHeader
        onClose={onClose}
        subSection={{
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
      <Metrics metrics={metrics} />
      <SettingsSection
        title="Sentinel settings"
        settings={[
          { label: "Protocol", value: "HTTP/2" },
          { label: "Port", value: 8080 },
          { label: "Health check", value: "/healthz" },
          { label: "Request timeout", value: "30s" },
          { label: "Max connections", value: 1000 },
          { label: "TLS", value: "Enabled" },
        ]}
      />
    </>
  );
};

type Props = {
  node: DeploymentNode | null;
  onClose: () => void;
};

export function NodeDetailsPanel({ node, onClose }: Props) {
  if (!node) {
    return null;
  }

  const renderDetails = () => {
    if (isSkeletonNode(node) || isOriginNode(node)) {
      return null;
    }
    if (isRegionNode(node)) {
      return <RegionNodeDetails node={node} onClose={onClose} />;
    }
    if (isSentinelNode(node)) {
      return <SentinelNodeDetails node={node} onClose={onClose} />;
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
    <div className="fixed top-40 right-4 bottom-14 flex flex-col gap-2 pointer-events-none">
      <div
        className={cn(
          "rounded-xl bg-white dark:bg-black border border-grayA-4 shadow-[0_2px_8px_-2px_rgba(0,0,0,0.1)] pointer-events-auto min-w-[360px] max-h-[calc(100vh-300px)] flex flex-col pb-6",
          "transition-all duration-300 ease-out",
          isOpen ? "opacity-100 translate-y-0" : "opacity-0 -translate-y-2 pointer-events-none",
        )}
      >
        <div className="flex flex-col items-center overflow-y-auto max-h-full pb-4">{content}</div>
      </div>
      <div
        className={cn(
          "rounded-xl bg-white dark:bg-black border border-grayA-4 shadow-[0_2px_8px_-2px_rgba(0,0,0,0.1)] pointer-events-auto min-w-[360px] h-12 flex px-[11px] gap-2 items-center",
          "transition-all duration-300 ease-out",
          isOpen ? "opacity-100 translate-y-0" : "opacity-0 -translate-y-2 pointer-events-none",
        )}
      >
        <Button
          variant="outline"
          className="bg-gray-1 rounded-lg shadow-sm text-[13px] font-medium"
        >
          Logs
        </Button>
        <Button
          variant="outline"
          className="bg-gray-1 rounded-lg shadow-sm text-[13px] font-medium"
        >
          Restart
        </Button>
        <Button
          variant="outline"
          className="bg-gray-1 rounded-lg shadow-sm text-[13px] font-medium"
        >
          Drain
        </Button>
        <Button
          variant="outline"
          className="bg-gray-1 rounded-lg shadow-sm text-[13px] font-medium"
        >
          Shell
        </Button>
        <Button
          variant="outline"
          className="bg-gray-1 rounded-lg shadow-sm text-[13px] font-medium ml-auto size-[26px]"
        >
          <Dots className="text-gray-9" iconSize="sm-regular" />
        </Button>
      </div>
    </div>
  );
}
