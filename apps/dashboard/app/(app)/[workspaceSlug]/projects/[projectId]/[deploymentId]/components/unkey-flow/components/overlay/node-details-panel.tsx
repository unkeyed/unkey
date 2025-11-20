import { ChartActivity, Layers3 } from "@unkey/icons";
import { InfoTooltip } from "@unkey/ui";
import { cn } from "@unkey/ui/src/lib/utils";
import { useEffect, useState } from "react";
import { type DeploymentNode, type NodeMetadata, REGION_INFO } from "../nodes/types";
import { NodeDetailsPanelHeader } from "./node-details-panel/components/header";
import { Metrics } from "./node-details-panel/components/metrics";
import { SettingsSection } from "./node-details-panel/components/settings-row";
import { metrics } from "./node-details-panel/constants";
import { GatewayInstances } from "./node-details-panel/region-node/gateway-instances";

type RegionNodeDetailsProps = {
  node: DeploymentNode & {
    metadata: Extract<NodeMetadata, { type: "region" }>;
  };
  onClose: () => void;
};

const RegionNodeDetails = ({ node, onClose }: RegionNodeDetailsProps) => {
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
      <GatewayInstances instances={node.children ?? []} />
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

type GatewayNodeDetailsProps = {
  node: DeploymentNode & {
    metadata: Extract<NodeMetadata, { type: "gateway" }>;
  };
  onClose: () => void;
};

const GatewayNodeDetails = ({ node, onClose }: GatewayNodeDetailsProps) => {
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
        title="Gateway settings"
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

const assertUnreachable = (value: never): never => {
  throw new Error(`Unhandled case: ${JSON.stringify(value)}`);
};

type NodeDetailsPanelProps = {
  node?: DeploymentNode;
};

export const NodeDetailsPanel = ({ node }: NodeDetailsPanelProps) => {
  const [isOpen, setIsOpen] = useState(false);

  useEffect(() => {
    if (node?.id) {
      setIsOpen(true);
    } else {
      setIsOpen(false);
    }
  }, [node?.id]);

  const handleClose = () => {
    setIsOpen(false);
  };

  if (!node) {
    return null;
  }

  const renderDetails = () => {
    switch (node.metadata.type) {
      case "origin":
        return null;
      case "region":
        return (
          <RegionNodeDetails
            node={
              node as DeploymentNode & {
                metadata: Extract<NodeMetadata, { type: "region" }>;
              }
            }
            onClose={handleClose}
          />
        );
      case "gateway":
        return (
          <GatewayNodeDetails
            node={
              node as DeploymentNode & {
                metadata: Extract<NodeMetadata, { type: "gateway" }>;
              }
            }
            onClose={handleClose}
          />
        );
      default:
        return assertUnreachable(node.metadata);
    }
  };

  const content = renderDetails();

  if (!content) {
    return null;
  }

  return (
    <div
      className={cn(
        "absolute top-14 right-4 bottom-14 rounded-xl bg-white dark:bg-black border border-grayA-4 shadow-[0_2px_8px_-2px_rgba(0,0,0,0.1)] pointer-events-auto min-w-[360px] max-h-[calc(100vh-80px)] flex flex-col pb-6",
        "transition-all duration-300 ease-out",
        isOpen ? "opacity-100 translate-y-0" : "opacity-0 -translate-y-2 pointer-events-none",
      )}
    >
      <div className="flex flex-col items-center overflow-y-auto max-h-full pb-4">{content}</div>
    </div>
  );
};
