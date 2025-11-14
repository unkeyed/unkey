import {
  Bolt,
  ChartActivity,
  CircleCheck,
  Focus,
  Heart,
  Layers3,
} from "@unkey/icons";
import { cn } from "@unkey/ui/src/lib/utils";
import type { DeploymentNode, RegionMetadata, HealthStatus } from "./types";
import { FLAGS } from "./flags";
import { StatusDot } from "./status-dot";
import { HealthBanner } from "./health-banner";
import { STATUS_CONFIG } from "./status-config";
import type { PropsWithChildren } from "react";

function getRegionFlagCode(
  regionId: string
): RegionMetadata["flagCode"] | undefined {
  switch (true) {
    case regionId.startsWith("us-"):
      return "us";
    case regionId.startsWith("ap-south"):
      return "in";
    case regionId.startsWith("ap-east"):
      return "hk";
    case regionId.startsWith("eu-"):
      return "eu";
    default:
      return undefined;
  }
}

function getHealthStyles(health: HealthStatus): { ring: string; glow: string } {
  const styleMap: Record<HealthStatus, { ring: string; glow: string }> = {
    normal: {
      ring: "hover:ring-grayA-2",
      glow: "hover:shadow-[0_4px_16px_-4px_rgba(0,0,0,0.15),0_0_0_1px_hsl(var(--grayA-6)),0_0_30px_color-mix(in_srgb,hsl(var(--grayA-9))_17.5%,transparent)]",
    },
    unstable: {
      ring: "hover:ring-orangeA-3",
      glow: "hover:shadow-[0_4px_16px_-4px_rgba(0,0,0,0.15),0_0_0_1px_hsl(var(--orangeA-3)),0_0_30px_color-mix(in_srgb,hsl(var(--orangeA-9))_20%,transparent)]",
    },
    degraded: {
      ring: "hover:ring-warningA-3",
      glow: "hover:shadow-[0_4px_16px_-4px_rgba(0,0,0,0.15),0_0_0_1px_hsl(var(--warningA-3)),0_0_30px_color-mix(in_srgb,hsl(var(--warningA-9))_20%,transparent)]",
    },
    unhealthy: {
      ring: "hover:ring-errorA-3",
      glow: "hover:shadow-[0_4px_16px_-4px_rgba(0,0,0,0.15),0_0_0_1px_hsl(var(--errorA-3)),0_0_30px_color-mix(in_srgb,hsl(var(--errorA-9))_20%,transparent)]",
    },
    recovering: {
      ring: "hover:ring-featureA-3",
      glow: "hover:shadow-[0_4px_16px_-4px_rgba(0,0,0,0.15),0_0_0_1px_hsl(var(--featureA-3)),0_0_30px_color-mix(in_srgb,hsl(var(--featureA-9))_20%,transparent)]",
    },
    health_syncing: {
      ring: "hover:ring-infoA-3",
      glow: "hover:shadow-[0_4px_16px_-4px_rgba(0,0,0,0.15),0_0_0_1px_hsl(var(--infoA-3)),0_0_30px_color-mix(in_srgb,hsl(var(--infoA-9))_20%,transparent)]",
    },
    unknown: {
      ring: "hover:ring-grayA-2",
      glow: "hover:shadow-[0_4px_16px_-4px_rgba(0,0,0,0.15),0_0_0_1px_hsl(var(--grayA-6)),0_0_30px_color-mix(in_srgb,hsl(var(--grayA-9))_17.5%,transparent)]",
    },
    disabled: {
      ring: "hover:ring-grayA-2",
      glow: "hover:shadow-[0_4px_16px_-4px_rgba(0,0,0,0.15),0_0_0_1px_hsl(var(--grayA-6)),0_0_30px_color-mix(in_srgb,hsl(var(--grayA-9))_17.5%,transparent)]",
    },
  };

  return styleMap[health];
}

type CardHeaderProps = {
  icon: React.ReactNode;
  title: string;
  subtitle: string;
  health: HealthStatus;
};

function CardHeader({ icon, title, subtitle, health }: CardHeaderProps) {
  const { colors } = STATUS_CONFIG[health];
  const heartBoxShadow = `0 0 8px 1px ${colors.dotRing} inset, 0 0 0 1px var(--color-grayA-gray-a5, rgba(0, 9, 50, 0.12)) inset`;

  return (
    <div
      className="border-b border-grayA-4 flex px-3 py-2.5 rounded-t-[14px]"
      style={{
        background:
          "radial-gradient(circle at 5% 15%, hsl(var(--grayA-3)) 0%, transparent 20%), light-dark(#FFF, #000)",
      }}
    >
      <div className="flex items-center justify-between gap-3">
        {icon}
        <div className="flex flex-col gap-[3px] justify-center h-9 py-2">
          <div className="text-accent-12 font-medium text-xs font-mono">
            {title}
          </div>
          <div className="text-gray-9 text-[11px]">{subtitle}</div>
        </div>
      </div>
      <div className="flex gap-2 items-center ml-auto">
        <div className="border bg-gray-1 border-grayA-3 h-11 rounded-lg w-8">
          <div className="h-6 border-b border-grayA-3 relative">
            <StatusDot healthStatus="normal" />
          </div>
          <div className="h-5 bg-grayA-2 pl-1 pt-[3px]">
            <CircleCheck className="text-gray-9" iconSize="sm-regular" />
          </div>
        </div>
        <div
          className="border bg-gray-1 border-grayA-3 h-11 rounded-lg w-8 ml-auto"
          style={{ boxShadow: heartBoxShadow }}
        >
          <div className="h-6 border-b border-grayA-3 relative">
            <StatusDot healthStatus={health} />
          </div>
          <div className="h-5 bg-grayA-2 pl-1 pt-[3px]">
            <Heart className={colors.textColor} iconSize="sm-regular" />
          </div>
        </div>
      </div>
    </div>
  );
}

type CardFooterProps = {
  flagCode?: RegionMetadata["flagCode"];
  instances?: number;
  power: string | number;
  storage?: string;
};

function CardFooter({ flagCode, instances, power, storage }: CardFooterProps) {
  const FlagComponent = flagCode ? FLAGS[flagCode] : null;

  return (
    <div className="p-1 flex items-center h-full bg-grayA-2 rounded-b-[14px]">
      {FlagComponent && (
        <div className="size-[22px] bg-grayA-3 rounded-full p-[3px] flex items-center justify-center mr-1.5">
          <FlagComponent />
        </div>
      )}
      {instances !== undefined && (
        <div className="bg-grayA-3 p-1.5 flex items-center justify-between rounded-full h-5 gap-1.5">
          <ChartActivity iconSize="sm-medium" className="shrink-0" />
          <span className="text-gray-9 text-[10px] tabular-nums">
            {instances}
          </span>
        </div>
      )}
      <div className="flex items-center gap-2 ml-auto">
        <div className="bg-grayA-3 p-1.5 flex items-center justify-between rounded-full h-5 gap-1.5">
          <Bolt iconSize="sm-medium" className="shrink-0" />
          <span className="text-gray-9 text-[10px] tabular-nums">
            {typeof power === "number" ? `${power}%` : power}
          </span>
        </div>
        {storage && (
          <div className="bg-grayA-3 p-1.5 flex items-center justify-between rounded-full h-5 gap-1.5">
            <Focus iconSize="sm-regular" className="shrink-0" />
            <span className="text-gray-9 text-[10px] tabular-nums">
              {storage}
            </span>
          </div>
        )}
      </div>
    </div>
  );
}

type NodeWrapperProps = PropsWithChildren<{
  health: HealthStatus;
}>;

function NodeWrapper({ health, children }: NodeWrapperProps) {
  const isDisabled = health === "disabled";
  const { ring, glow } = getHealthStyles(health);

  return (
    <div
      className={cn(
        "relative w-[282px] rounded-[14px]",
        isDisabled
          ? "grayscale opacity-90 cursor-not-allowed"
          : cn(
              "hover:scale-[1.01] transition-all duration-200 cursor-pointer",
              "hover:ring-2 hover:ring-offset-0",
              ring,
              glow
            )
      )}
    >
      <HealthBanner healthStatus={health} />
      <div
        className={cn(
          "relative z-20 w-[282px] h-[100px] border border-grayA-4 rounded-[14px] flex flex-col bg-white dark:bg-black shadow-[0_2px_8px_-2px_rgba(0,0,0,0.1)]"
        )}
      >
        {children}
      </div>
    </div>
  );
}

type RegionNodeProps = {
  node: DeploymentNode & { metadata: { type: "region" } };
};

export function RegionNode({ node }: RegionNodeProps) {
  const { flagCode, zones, instances, power, storage, health } = node.metadata;
  const FlagComponent = FLAGS[flagCode];
  const subtitle = `${zones} availability ${zones > 1 ? "zones" : "zone"}`;

  return (
    <NodeWrapper health={health}>
      <CardHeader
        icon={
          <div className="border rounded-[10px] border-grayA-3 size-9 bg-grayA-3 flex items-center justify-center">
            <FlagComponent />
          </div>
        }
        title={node.label}
        subtitle={subtitle}
        health={health}
      />
      <CardFooter instances={instances} power={power} storage={storage} />
    </NodeWrapper>
  );
}

type InstanceNodeProps = {
  node: DeploymentNode & { metadata: { type: "instance" } };
  parentRegionId: string;
};

export function InstanceNode({ node, parentRegionId }: InstanceNodeProps) {
  const { description, instances, power, storage, health } = node.metadata;
  const flagCode = getRegionFlagCode(parentRegionId);

  return (
    <NodeWrapper health={health}>
      <CardHeader
        icon={
          <div className="border rounded-[10px] size-9 flex items-center justify-center border-grayA-5 bg-grayA-2">
            <Layers3 iconSize="sm-medium" className="text-gray-11" />
          </div>
        }
        title={node.label}
        subtitle={description}
        health={health}
      />
      <CardFooter
        flagCode={flagCode}
        instances={instances}
        power={power}
        storage={storage}
      />
    </NodeWrapper>
  );
}
